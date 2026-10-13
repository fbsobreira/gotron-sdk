package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests run upgradeTo against a fake GitHub release served by an
// httptest TLS server. The upgrade target is a temp file that stands in for the
// installed tronctl; the test binary is never replaced. Release binaries that
// must run are copies of the test binary acting as tronctl (see TestMain).

const (
	upgradeTag   = "v9.9.9"
	plainTarName = "tronctl_9.9.9_linux_amd64.tar.gz"
	pqTarName    = "tronctl-pq_9.9.9_linux_amd64.tar.gz"
	plainZipName = "tronctl_9.9.9_windows_amd64.zip"
	pqZipName    = "tronctl-pq_9.9.9_windows_amd64.zip"
	oldBinary    = "old tronctl binary\n"
	targetMode   = fs.FileMode(0o775)
	// currentTag is the version the fixture's target is upgraded from.
	currentTag = "v9.9.8"
)

// goodBinary stands in for the release binary in tests that fail before it
// would run; tests that run it use fakeTronctl.
var goodBinary = []byte("release binary placeholder, never run\n")

var (
	fakeTronctlOnce  sync.Once
	fakeTronctlBytes []byte
	errFakeTronctl   error
)

// fakeTronctl returns the test binary, which acts as tronctl when run with
// fakeTronctlEnv=1. It is read once: it is tens of megabytes.
func fakeTronctl(t *testing.T) []byte {
	t.Helper()
	fakeTronctlOnce.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			errFakeTronctl = err
			return
		}
		fakeTronctlBytes, errFakeTronctl = os.ReadFile(exe)
	})
	require.NoError(t, errFakeTronctl)
	return fakeTronctlBytes
}

// setFakeVersion makes the fake tronctl report version and exit with code.
func setFakeVersion(t *testing.T, version string, code int) {
	t.Helper()
	t.Setenv(fakeVersionEnv, version)
	t.Setenv(fakeExitEnv, fmt.Sprint(code))
}

type archiveEntry struct {
	name     string
	body     []byte
	typeflag byte        // tar only; 0 means a regular file
	linkname string      // tar only
	mode     fs.FileMode // zip only; 0 means a regular 0o755 file
}

func releaseEntries(binName string, bin []byte) []archiveEntry {
	return []archiveEntry{
		{name: "LICENSE", body: []byte("license\n")},
		{name: binName, body: bin},
		{name: "README.md", body: []byte("readme\n")},
	}
}

func makeTarGz(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	// BestSpeed: some entries are the multi-megabyte test binary.
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	require.NoError(t, err)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		typeflag := e.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		hdr := &tar.Header{Name: e.name, Mode: 0o755, Typeflag: typeflag, Linkname: e.linkname}
		if typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		require.NoError(t, tw.WriteHeader(hdr))
		if typeflag == tar.TypeReg {
			_, err := tw.Write(e.body)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func makeZip(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if len(e.body) > 1<<20 {
			hdr.Method = zip.Store // the test binary: skip compressing it
		}
		mode := e.mode
		if mode == 0 {
			mode = 0o755
		}
		hdr.SetMode(mode)
		w, err := zw.CreateHeader(hdr)
		require.NoError(t, err)
		_, err = w.Write(e.body)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func checksumLine(name string, data []byte) string {
	return sha256Hex(data) + "  " + name
}

type upgradeFixture struct {
	t        *testing.T
	srv      *httptest.Server
	mu       sync.Mutex
	files    map[string][]byte
	requests []string
	release  *GitHubRelease
	target   string
	// targetMode is the mode the target was created with.
	targetMode fs.FileMode
	opts       upgradeOptions
	out        *bytes.Buffer
}

func newUpgradeFixture(t *testing.T) *upgradeFixture {
	t.Helper()
	f := &upgradeFixture{t: t, files: map[string][]byte{}, out: &bytes.Buffer{}}
	f.srv = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	f.release = &GitHubRelease{TagName: upgradeTag}
	f.setTarget(t.TempDir(), targetMode)
	f.opts = upgradeOptions{client: f.srv.Client(), out: f.out, current: currentTag, goos: "linux", goarch: "amd64"}
	// Any binary the version check runs is the fake tronctl, reporting the
	// release's version unless the test says otherwise.
	t.Setenv(fakeTronctlEnv, "1")
	setFakeVersion(t, upgradeTag+"-abc1234", 0)
	return f
}

// setTarget (re)creates the installed binary in dir with mode. It is named
// like the real one, so on Windows the version check runs a .exe file.
func (f *upgradeFixture) setTarget(dir string, mode fs.FileMode) {
	f.t.Helper()
	if f.target != "" {
		require.NoError(f.t, os.Remove(f.target))
	}
	f.target = filepath.Join(dir, upgradeBinaryName(runtime.GOOS))
	f.targetMode = mode
	require.NoError(f.t, os.WriteFile(f.target, []byte(oldBinary), 0o600))
	require.NoError(f.t, os.Chmod(f.target, mode))
}

// serve answers /download/<name> from files (404 when absent) and
// /redirect/<name> with a redirect to the plain-http /download/<name>.
func (f *upgradeFixture) serve(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.URL.Path)
	f.mu.Lock()
	f.requests = append(f.requests, r.URL.Path)
	body, ok := f.files[name]
	f.mu.Unlock()
	if strings.HasPrefix(r.URL.Path, "/redirect/") {
		http.Redirect(w, r, "http://"+r.Host+"/download/"+name, http.StatusFound)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, err := w.Write(body); err != nil {
		f.t.Errorf("serve %s: %v", name, err)
	}
}

// publish serves content under name and lists it as a release asset.
func (f *upgradeFixture) publish(name string, content []byte) {
	f.files[name] = content
	f.release.Assets = append(f.release.Assets, GitHubReleaseAssets{
		Name: name,
		URL:  f.srv.URL + "/download/" + name,
	})
}

func (f *upgradeFixture) publishChecksums(lines ...string) {
	f.publish("checksums.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// publishArchive publishes an archive and a checksums.txt that lists it.
func (f *upgradeFixture) publishArchive(name string, archive []byte) {
	f.publish(name, archive)
	f.publishChecksums(
		checksumLine("tronctl_9.9.9_darwin_arm64.tar.gz", []byte("other")),
		checksumLine(name, archive),
	)
}

func (f *upgradeFixture) setAssetURL(name, url string) {
	for i := range f.release.Assets {
		if f.release.Assets[i].Name == name {
			f.release.Assets[i].URL = url
			return
		}
	}
	f.t.Fatalf("no asset %s", name)
}

func (f *upgradeFixture) removeAsset(name string) {
	assets := f.release.Assets[:0]
	for _, a := range f.release.Assets {
		if a.Name != name {
			assets = append(assets, a)
		}
	}
	f.release.Assets = assets
}

func (f *upgradeFixture) run() error {
	return upgradeTo(context.Background(), f.release, f.target, f.opts)
}

func (f *upgradeFixture) requested(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.requests {
		if path.Base(p) == name {
			return true
		}
	}
	return false
}

// requireOnlyTarget checks that no temp or backup file was left next to the
// target, apart from extra.
func (f *upgradeFixture) requireOnlyTarget(extra ...string) {
	f.t.Helper()
	entries, err := os.ReadDir(filepath.Dir(f.target))
	require.NoError(f.t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := append([]string{filepath.Base(f.target)}, extra...)
	require.ElementsMatch(f.t, want, names, "leftover files next to the target")
}

func (f *upgradeFixture) requireUnchanged() {
	f.t.Helper()
	got, err := os.ReadFile(f.target)
	require.NoError(f.t, err)
	require.Equal(f.t, oldBinary, string(got), "target was modified")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(f.target)
		require.NoError(f.t, err)
		require.Equal(f.t, f.targetMode, info.Mode().Perm())
	}
	f.requireOnlyTarget()
}

func setUpgradeCap(t *testing.T, p *int64, v int64) {
	t.Helper()
	old := *p
	*p = v
	t.Cleanup(func() { *p = old })
}

func TestUpgradeTo_ReplacesBinary(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(f *upgradeFixture, bin []byte)
		wantFetch string
		forbidden []string
	}{
		{
			name: "tar.gz",
			setup: func(f *upgradeFixture, bin []byte) {
				f.publishArchive(plainTarName, makeTarGz(f.t, releaseEntries("tronctl", bin)...))
			},
			wantFetch: plainTarName,
		},
		{
			name: "zip for a windows asset, uppercase hex",
			setup: func(f *upgradeFixture, bin []byte) {
				f.opts.goos = "windows"
				archive := makeZip(f.t, releaseEntries("tronctl.exe", bin)...)
				f.publish(plainZipName, archive)
				f.publishChecksums(strings.ToUpper(sha256Hex(archive)) + "  " + plainZipName)
			},
			wantFetch: plainZipName,
		},
		{
			name: "pq build picks tronctl-pq, binary-mode checksum line",
			setup: func(f *upgradeFixture, bin []byte) {
				f.opts.pq = true
				setFakeVersion(f.t, upgradeTag+"-fa1c0de", 0)
				plain := makeTarGz(f.t, releaseEntries("tronctl", goodBinary)...)
				pq := makeTarGz(f.t, releaseEntries("tronctl", bin)...)
				f.publish(plainTarName, plain)
				f.publish(pqTarName, pq)
				f.publishChecksums(
					checksumLine(plainTarName, plain),
					sha256Hex(pq)+" *"+pqTarName,
				)
			},
			wantFetch: pqTarName,
			forbidden: []string{plainTarName},
		},
		{
			// `tronctl version` still resolves the signer, so the user's
			// TRONCTL_* settings and home directory must not reach it.
			name: "version check ignores the user's tronctl settings",
			setup: func(f *upgradeFixture, bin []byte) {
				home := f.t.TempDir()
				f.t.Setenv("HOME", home)
				f.t.Setenv("USERPROFILE", home)
				f.t.Setenv(fakeHomeEnv, home)
				f.t.Setenv("TRONCTL_SIGNER", "nosuchaccount")
				f.publishArchive(plainTarName, makeTarGz(f.t, releaseEntries("tronctl", bin)...))
			},
			wantFetch: plainTarName,
		},
		{
			// tronctl loads .env from its working directory, so the version
			// check must not run in the user's.
			name: "version check ignores a .env in the working directory",
			setup: func(f *upgradeFixture, bin []byte) {
				wd := f.t.TempDir()
				require.NoError(f.t, os.WriteFile(filepath.Join(wd, ".env"), []byte("TRONCTL_SIGNER=x\n"), 0o600))
				f.t.Chdir(wd)
				f.publishArchive(plainTarName, makeTarGz(f.t, releaseEntries("tronctl", bin)...))
			},
			wantFetch: plainTarName,
		},
		{
			// main prints os.Args[0] in the version line, unsplit on Windows;
			// a folder name holding "version <tag>" must not break the check.
			name: "installed in a folder whose name holds a version",
			setup: func(f *upgradeFixture, bin []byte) {
				dir := filepath.Join(f.t.TempDir(), "a version v0.0.1 b")
				require.NoError(f.t, os.Mkdir(dir, 0o755))
				f.setTarget(dir, targetMode)
				f.publishArchive(plainTarName, makeTarGz(f.t, releaseEntries("tronctl", bin)...))
			},
			wantFetch: plainTarName,
		},
		{
			// The new binary keeps the installed binary's mode; upgrading
			// must not open a 0o750 install up to everyone.
			name: "keeps a 0o750 mode",
			setup: func(f *upgradeFixture, bin []byte) {
				f.setTarget(filepath.Dir(f.target), 0o750)
				f.publishArchive(plainTarName, makeTarGz(f.t, releaseEntries("tronctl", bin)...))
			},
			wantFetch: plainTarName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := fakeTronctl(t)
			f := newUpgradeFixture(t)
			tt.setup(f, bin)

			require.NoError(t, f.run(), f.out.String())

			got, err := os.ReadFile(f.target)
			require.NoError(t, err)
			require.True(t, bytes.Equal(bin, got), "the target is not the release binary")
			if runtime.GOOS == "windows" {
				// The running binary cannot be deleted there, so it stays as .old.
				old, err := os.ReadFile(f.target + ".old")
				require.NoError(t, err)
				require.Equal(t, oldBinary, string(old))
				f.requireOnlyTarget(filepath.Base(f.target) + ".old")
			} else {
				info, err := os.Stat(f.target)
				require.NoError(t, err)
				require.Equal(t, f.targetMode, info.Mode().Perm())
				f.requireOnlyTarget()
			}
			require.Contains(t, f.out.String(), "Upgrading tronctl: "+currentTag+" → "+upgradeTag)
			require.Contains(t, f.out.String(), "Checksum verified.")
			require.Contains(t, f.out.String(), tt.wantFetch)
			require.True(t, f.requested(tt.wantFetch), "did not download %s", tt.wantFetch)
			for _, name := range tt.forbidden {
				require.False(t, f.requested(name), "downloaded %s", name)
			}
		})
	}
}

func TestUpgradeTo_FailsClosed(t *testing.T) {
	goodTar := func(f *upgradeFixture) []byte {
		return makeTarGz(f.t, releaseEntries("tronctl", goodBinary)...)
	}
	tests := []struct {
		name      string
		setup     func(f *upgradeFixture)
		wantErr   string
		forbidden []string // assets that must not be downloaded
		// quiet: refused before both assets are found, so nothing may claim
		// that an upgrade started.
		quiet bool
	}{
		{
			name: "pq build without a pq archive never takes the plain one",
			setup: func(f *upgradeFixture) {
				f.opts.pq = true
				f.publishArchive(plainTarName, goodTar(f))
			},
			wantErr:   "no tronctl-pq archive",
			forbidden: []string{plainTarName},
			quiet:     true,
		},
		{
			name: "pq build on windows/arm64",
			setup: func(f *upgradeFixture) {
				f.opts.pq = true
				f.opts.goos, f.opts.goarch = "windows", "arm64"
				f.publishArchive("tronctl_9.9.9_windows_arm64.zip", makeZip(f.t, releaseEntries("tronctl.exe", goodBinary)...))
				f.publishArchive(pqZipName, makeZip(f.t, releaseEntries("tronctl.exe", goodBinary)...))
			},
			wantErr:   "no tronctl-pq archive for windows/arm64",
			forbidden: []string{"tronctl_9.9.9_windows_arm64.zip", pqZipName},
			quiet:     true,
		},
		{
			name: "checksums.txt missing from the release",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, goodTar(f))
				f.removeAsset("checksums.txt")
			},
			wantErr:   "checksums.txt",
			forbidden: []string{plainTarName},
			quiet:     true,
		},
		{
			name: "archive missing from the release",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, goodTar(f))
				f.removeAsset(plainTarName)
			},
			wantErr: "has no " + plainTarName,
			quiet:   true,
		},
		{
			name: "archive not listed in checksums.txt",
			setup: func(f *upgradeFixture) {
				f.publish(plainTarName, goodTar(f))
				f.publishChecksums(checksumLine(pqTarName, []byte("x")))
			},
			wantErr:   "not listed in checksums.txt",
			forbidden: []string{plainTarName},
		},
		{
			name: "archive listed twice in checksums.txt",
			setup: func(f *upgradeFixture) {
				archive := goodTar(f)
				f.publish(plainTarName, archive)
				f.publishChecksums(checksumLine(plainTarName, archive), checksumLine(plainTarName, []byte("x")))
			},
			wantErr:   "more than once",
			forbidden: []string{plainTarName},
		},
		{
			name: "checksum mismatch",
			setup: func(f *upgradeFixture) {
				f.publish(plainTarName, goodTar(f))
				f.publishChecksums(checksumLine(plainTarName, []byte("tampered")))
			},
			wantErr: "checksum mismatch",
		},
		{
			name: "malformed hash",
			setup: func(f *upgradeFixture) {
				f.publish(plainTarName, goodTar(f))
				f.publishChecksums(strings.Repeat("z", 64) + "  " + plainTarName)
			},
			wantErr:   "malformed checksum",
			forbidden: []string{plainTarName},
		},
		{
			name: "short hash",
			setup: func(f *upgradeFixture) {
				archive := goodTar(f)
				f.publish(plainTarName, archive)
				f.publishChecksums(sha256Hex(archive)[:62] + "  " + plainTarName)
			},
			wantErr:   "malformed checksum",
			forbidden: []string{plainTarName},
		},
		{
			name: "non-https asset URL",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, goodTar(f))
				f.setAssetURL(plainTarName, strings.Replace(f.srv.URL, "https://", "http://", 1)+"/download/"+plainTarName)
			},
			wantErr: "non-https",
		},
		{
			name: "redirect to plain http",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, goodTar(f))
				f.setAssetURL(plainTarName, f.srv.URL+"/redirect/"+plainTarName)
			},
			wantErr: "non-https",
		},
		{
			name: "HTTP 404",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, goodTar(f))
				delete(f.files, plainTarName)
			},
			wantErr: "HTTP 404",
		},
		{
			name: "oversized archive",
			setup: func(f *upgradeFixture) {
				setUpgradeCap(f.t, &maxArchiveSize, 64)
				f.publishArchive(plainTarName, goodTar(f))
			},
			wantErr: "larger than",
		},
		{
			name: "oversized checksums.txt",
			setup: func(f *upgradeFixture) {
				setUpgradeCap(f.t, &maxChecksumsSize, 16)
				f.publishArchive(plainTarName, goodTar(f))
			},
			wantErr:   "larger than",
			forbidden: []string{plainTarName},
		},
		{
			name: "oversized binary in the archive",
			setup: func(f *upgradeFixture) {
				setUpgradeCap(f.t, &maxBinarySize, 16)
				f.publishArchive(plainTarName, goodTar(f))
			},
			wantErr: "larger than",
		},
		{
			name: "path traversal entry",
			setup: func(f *upgradeFixture) {
				entries := append(releaseEntries("tronctl", goodBinary), archiveEntry{name: "../tronctl", body: goodBinary})
				f.publishArchive(plainTarName, makeTarGz(f.t, entries...))
			},
			wantErr: "unsafe archive entry",
		},
		{
			name: "path traversal in another entry",
			setup: func(f *upgradeFixture) {
				entries := append(releaseEntries("tronctl", goodBinary), archiveEntry{name: "../LICENSE", body: []byte("x")})
				f.publishArchive(plainTarName, makeTarGz(f.t, entries...))
			},
			wantErr: "unsafe archive entry",
		},
		{
			name: "absolute entry",
			setup: func(f *upgradeFixture) {
				entries := append(releaseEntries("tronctl", goodBinary), archiveEntry{name: "/tmp/x", body: []byte("x")})
				f.publishArchive(plainTarName, makeTarGz(f.t, entries...))
			},
			wantErr: "unsafe archive entry",
		},
		{
			name: "nested entry",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, makeTarGz(f.t, archiveEntry{name: "bin/tronctl", body: goodBinary}))
			},
			wantErr: "unsafe archive entry",
		},
		{
			name: "symlink named tronctl",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, makeTarGz(f.t, archiveEntry{name: "tronctl", typeflag: tar.TypeSymlink, linkname: "/bin/sh"}))
			},
			wantErr: "not a regular file",
		},
		{
			name: "hard link named tronctl",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, makeTarGz(f.t,
					archiveEntry{name: "LICENSE", body: []byte("x")},
					archiveEntry{name: "tronctl", typeflag: tar.TypeLink, linkname: "LICENSE"}))
			},
			wantErr: "not a regular file",
		},
		{
			name: "zip symlink named tronctl.exe",
			setup: func(f *upgradeFixture) {
				f.opts.goos = "windows"
				f.publishArchive(plainZipName, makeZip(f.t, archiveEntry{name: "tronctl.exe", body: []byte("/bin/sh"), mode: fs.ModeSymlink | 0o777}))
			},
			wantErr: "not a regular file",
		},
		{
			name: "zip backslash entry",
			setup: func(f *upgradeFixture) {
				f.opts.goos = "windows"
				f.publishArchive(plainZipName, makeZip(f.t, archiveEntry{name: `..\tronctl.exe`, body: goodBinary}))
			},
			wantErr: "unsafe archive entry",
		},
		{
			name: "tronctl missing from the archive",
			setup: func(f *upgradeFixture) {
				f.publishArchive(plainTarName, makeTarGz(f.t, archiveEntry{name: "LICENSE", body: []byte("x")}))
			},
			wantErr: "does not contain tronctl",
		},
		{
			name: "tronctl.exe twice in a zip",
			setup: func(f *upgradeFixture) {
				f.opts.goos = "windows"
				entries := append(releaseEntries("tronctl.exe", goodBinary), archiveEntry{name: "tronctl.exe", body: goodBinary})
				f.publishArchive(plainZipName, makeZip(f.t, entries...))
			},
			wantErr: "more than once",
		},
		{
			name: "tronctl twice in the archive",
			setup: func(f *upgradeFixture) {
				entries := append(releaseEntries("tronctl", goodBinary), archiveEntry{name: "tronctl", body: goodBinary})
				f.publishArchive(plainTarName, makeTarGz(f.t, entries...))
			},
			wantErr: "more than once",
		},
		{
			name: "new binary reports the wrong version",
			setup: func(f *upgradeFixture) {
				setFakeVersion(f.t, "v9.9.90-abc1234", 0)
				f.publishArchive(plainTarName, makeTarGz(f.t, releaseEntries("tronctl", fakeTronctl(f.t))...))
			},
			wantErr: "reports version v9.9.90-abc1234",
		},
		{
			name: "new binary reports a dirty build of the version",
			setup: func(f *upgradeFixture) {
				setFakeVersion(f.t, upgradeTag+"-abc1234-dirty", 0)
				f.publishArchive(plainTarName, makeTarGz(f.t, releaseEntries("tronctl", fakeTronctl(f.t))...))
			},
			wantErr: "reports version " + upgradeTag + "-abc1234-dirty",
		},
		{
			name: "new binary exits non-zero",
			setup: func(f *upgradeFixture) {
				setFakeVersion(f.t, upgradeTag, 3)
				f.publishArchive(plainTarName, makeTarGz(f.t, releaseEntries("tronctl", fakeTronctl(f.t))...))
			},
			wantErr: "failed to run",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newUpgradeFixture(t)
			tt.setup(f)

			err := f.run()

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
			f.requireUnchanged()
			for _, name := range tt.forbidden {
				require.False(t, f.requested(name), "downloaded %s", name)
			}
			if tt.quiet {
				require.NotContains(t, f.out.String(), "Upgrading tronctl")
			}
		})
	}
}

func TestUpgradeTo_TargetDirNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions differ on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can write to a read-only directory")
	}
	f := newUpgradeFixture(t)
	f.publishArchive(plainTarName, makeTarGz(t, releaseEntries("tronctl", goodBinary)...))
	dir := filepath.Dir(f.target)
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Errorf("restore permissions: %v", err)
		}
	})
	require.NoError(t, os.Chmod(dir, 0o555))

	err := f.run()

	require.Error(t, err)
	require.Contains(t, err.Error(), "not writable")
	require.Contains(t, err.Error(), "sudo")
	f.requireUnchanged()
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "tronctl")
	newPath := filepath.Join(dir, ".tronctl-upgrade-1")
	require.NoError(t, os.WriteFile(target, []byte("old"), 0o755))
	require.NoError(t, os.WriteFile(newPath, []byte("new"), 0o755))

	require.NoError(t, replaceExecutable(target, newPath))

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "new", string(got))
	_, err = os.Stat(newPath)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

// The Windows replace sequence only uses renames, so it runs on every OS.
func TestReplaceExecutableWithBackup(t *testing.T) {
	tests := []struct {
		name      string
		staleOld  bool
		missingNP bool // the new file is gone, so the second rename fails
		wantErr   bool
		wantBody  string
		wantOld   string // "" means no .old file
	}{
		{name: "replaces and keeps .old", wantBody: "new", wantOld: "old"},
		{name: "removes a stale .old first", staleOld: true, wantBody: "new", wantOld: "old"},
		{name: "restores the target when the second rename fails", missingNP: true, wantErr: true, wantBody: "old"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "tronctl.exe")
			newPath := filepath.Join(dir, ".tronctl-upgrade-1.exe")
			require.NoError(t, os.WriteFile(target, []byte("old"), 0o755))
			if !tt.missingNP {
				require.NoError(t, os.WriteFile(newPath, []byte("new"), 0o755))
			}
			if tt.staleOld {
				require.NoError(t, os.WriteFile(target+".old", []byte("stale"), 0o755))
			}

			err := replaceExecutableWithBackup(target, newPath)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			got, err := os.ReadFile(target)
			require.NoError(t, err)
			require.Equal(t, tt.wantBody, string(got))
			old, err := os.ReadFile(target + ".old")
			if tt.wantOld == "" {
				require.ErrorIs(t, err, fs.ErrNotExist)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantOld, string(old))
			}
		})
	}
}

func TestUpgradeVersionMatches(t *testing.T) {
	tests := []struct {
		output string
		tag    string
		want   bool
	}{
		{"TronCTL. tronctl version v1.2.3-abc1234 (goreleaser now)", "v1.2.3", true},
		{"TronCTL. tronctl version v1.2.3", "v1.2.3", true},
		{"TronCTL. tronctl version v1.2.30-abc1234 (goreleaser now)", "v1.2.3", false},
		{"TronCTL. tronctl version v1.2.30", "v1.2.3", false},
		{"TronCTL. tronctl version v1.0.0-rc1-abc1234 (x y)", "v1.0.0-rc1", true},
		{"TronCTL. tronctl version v1.0.0-rc2-abc1234 (x y)", "v1.0.0-rc1", false},
		{"TronCTL. tronctl version v1.2.2-abc1234", "v1.2.3", false},
		{"no version here", "v1.2.3", false},
		{"TronCTL. tronctl version v1.2.3", "", false},
		// Only "-<7 to 40 lowercase hex digits>" may follow the tag.
		{"TronCTL. tronctl version v1.0.0-rc1-abc1234 (x y)", "v1.0.0", false},
		{"TronCTL. tronctl version v1.0.0-abc123 (x y)", "v1.0.0", false},
		{"TronCTL. tronctl version v1.0.0-abc1234-dirty (x y)", "v1.0.0", false},
		{"TronCTL. tronctl version v1.0.0-ABC1234 (x y)", "v1.0.0", false},
		{"TronCTL. tronctl version v1.0.0.1", "v1.0.0", false},
		{"TronCTL. tronctl version v1.0.0-" + strings.Repeat("a", 41), "v1.0.0", false},
		{"TronCTL. tronctl version v1.0.0-" + strings.Repeat("a", 40), "v1.0.0", true},
		{"TronCTL. tronctl version v0.26.0", "v0.26.0", true},
		// The real v0.26.0 release output.
		{"TronCTL. tronctl version v0.26.0-202a9d9e (goreleaser@gotron.sh 2026-03-22T11:40:43Z)", "v0.26.0", true},
		// The tag must be taken literally, not as a pattern.
		{"TronCTL. tronctl version v1x2x3-abc1234", "v1.2.3", false},
		// Only the version line counts: a path in front of it does not.
		{`TronCTL. C:\x\version v1.0.0 y\tronctl.exe version v2.0.0`, "v1.0.0", false},
		{"warning: something\nTronCTL. tronctl version v1.2.3-abc1234 (x y)", "v1.2.3", true},
		{"version v1.2.3", "v1.2.3", false},
	}
	for _, tt := range tests {
		t.Run(tt.output+"/"+tt.tag, func(t *testing.T) {
			require.Equal(t, tt.want, upgradeVersionMatches(tt.output, tt.tag))
		})
	}
}

func TestUpgradeArchiveName(t *testing.T) {
	tests := []struct {
		tag, goos, goarch string
		pq                bool
		want              string
	}{
		{"v0.26.1", "linux", "amd64", false, "tronctl_0.26.1_linux_amd64.tar.gz"},
		{"v0.26.1", "darwin", "arm64", true, "tronctl-pq_0.26.1_darwin_arm64.tar.gz"},
		{"v0.26.1", "windows", "amd64", false, "tronctl_0.26.1_windows_amd64.zip"},
		{"v0.26.1", "windows", "amd64", true, "tronctl-pq_0.26.1_windows_amd64.zip"},
		{"v1.0.0-rc1", "linux", "arm64", false, "tronctl_1.0.0-rc1_linux_arm64.tar.gz"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, upgradeArchiveName(tt.tag, tt.goos, tt.goarch, tt.pq))
		})
	}
}

func TestFetchRelease(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tag string
		switch r.URL.EscapedPath() {
		case "/releases/latest":
			tag = "v9.9.9"
		case "/releases/tags/v1.2.3":
			tag = "v1.2.3"
		case "/releases/tags/v7.7.7":
			http.Redirect(w, r, "http://"+r.Host+"/releases/latest", http.StatusFound)
			return
		default:
			http.NotFound(w, r)
			return
		}
		if _, err := fmt.Fprintf(w, `{"tag_name":%q}`, tag); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	oldBase, oldClient := releasesAPIBase, upgradeHTTPClient
	releasesAPIBase, upgradeHTTPClient = srv.URL+"/releases", srv.Client()
	t.Cleanup(func() { releasesAPIBase, upgradeHTTPClient = oldBase, oldClient })

	tests := []struct {
		version string
		want    string
		wantErr string
	}{
		{version: "", want: "v9.9.9"},
		{version: "1.2.3", want: "v1.2.3"},
		{version: "v1.2.3", want: "v1.2.3"},
		{version: "v1.2.4", wantErr: "release v1.2.4 not found (HTTP 404)"},
		{version: "v1.2.3/../../latest", wantErr: "not found (HTTP 404)"},
		{version: "v7.7.7", wantErr: "refusing redirect to non-https URL"},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			release, err := fetchRelease(tt.version)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, release.TagName)
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	old := VersionWrapDump
	t.Cleanup(func() { VersionWrapDump = old })
	tests := []struct {
		dump string
		want string
	}{
		{"v0.26.0-202a9d9e", "v0.26.0"},
		{"v1.0.0-rc1-abc1234", "v1.0.0-rc1"},
		{"v0.26.0", "v0.26.0"},
		{"v1.0.0-rc1", "v1.0.0-rc1"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.dump, func(t *testing.T) {
			VersionWrapDump = tt.dump
			require.Equal(t, tt.want, getCurrentVersion())
		})
	}
}

func TestManagedInstallHint(t *testing.T) {
	const (
		brew      = "brew upgrade tronctl"
		goInstall = "go install github.com/fbsobreira/gotron-sdk/cmd/tronctl@latest"
	)
	tests := []struct {
		name         string
		execPath     string
		releaseBuild bool
		mainVersion  string
		want         string // "" means not managed
	}{
		{"release build", "/usr/local/bin/tronctl", true, "v0.26.0", ""},
		{"release build without build info", "/usr/local/bin/tronctl", true, "", ""},
		{"go install", "/home/u/go/bin/tronctl", false, "v0.26.0", goInstall},
		{"dev build", "/home/u/src/gotron-sdk/tronctl", false, "(devel)", ""},
		{"no build info", "/home/u/src/gotron-sdk/tronctl", false, "", ""},
		{"Homebrew Cellar", "/usr/local/Cellar/tronctl/0.26.0/bin/tronctl", true, "v0.26.0", brew},
		{"Homebrew prefix", "/opt/homebrew/bin/tronctl", true, "v0.26.0", brew},
		{"Linuxbrew", "/home/linuxbrew/.linuxbrew/Cellar/tronctl/0.26.0/bin/tronctl", false, "(devel)", brew},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, managed := managedInstallHint(tt.execPath, tt.releaseBuild, tt.mainVersion)
			if tt.want == "" {
				require.False(t, managed, msg)
				require.Empty(t, msg)
				return
			}
			require.True(t, managed)
			require.Contains(t, msg, tt.want)
		})
	}
}

func TestUpgradeSignals(t *testing.T) {
	want := []os.Signal{os.Interrupt, syscall.SIGTERM}
	if runtime.GOOS != "windows" {
		want = append(want, syscall.SIGHUP)
	}
	require.ElementsMatch(t, want, upgradeSignals())
}
