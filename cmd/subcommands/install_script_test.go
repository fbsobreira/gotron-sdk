package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests run the repository's install.sh against a fake curl that serves
// a fake release from a temp folder, so every path is exercised offline.

const fakeBinary = "#!/bin/sh\necho fake tronctl\n"

// fakeCurl serves $FIXTURES/<basename of the URL> and logs each URL, so a test
// can check what was (or was not) downloaded. A missing file fails like
// curl -f does on HTTP 404.
const fakeCurl = `#!/bin/sh
out=""
url=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
echo "$url" >> "$FIXTURES/curl.log"
f="$FIXTURES/$(basename "$url")"
[ -f "$f" ] || exit 22
if [ -n "$out" ]; then cp "$f" "$out"; else cat "$f"; fi
`

type installEnv struct {
	t        *testing.T
	fixtures string
	bin      string
	dest     string
	osName   string
	arch     string
}

func newInstallEnv(t *testing.T, withSHA bool) *installEnv {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("install.sh supports Linux and macOS only")
	}
	arch := map[string]string{"amd64": "amd64", "arm64": "arm64"}[runtime.GOARCH]
	if arch == "" {
		t.Skipf("install.sh does not support %s", runtime.GOARCH)
	}
	e := &installEnv{
		t:        t,
		fixtures: t.TempDir(),
		bin:      t.TempDir(),
		dest:     filepath.Join(t.TempDir(), "bin"),
		osName:   runtime.GOOS,
		arch:     arch,
	}
	require.NoError(t, os.WriteFile(filepath.Join(e.bin, "curl"), []byte(fakeCurl), 0o755))
	// Only the tools install.sh needs, and never sudo.
	tools := []string{"sh", "uname", "tr", "grep", "head", "cut", "awk", "mktemp",
		"rm", "mkdir", "tar", "chmod", "mv", "cat", "cp", "basename", "gzip"}
	if withSHA {
		tools = append(tools, "sha256sum", "shasum")
	}
	for _, tool := range tools {
		p, err := exec.LookPath(tool)
		if err != nil {
			continue // e.g. only one of sha256sum and shasum exists
		}
		require.NoError(t, os.Symlink(p, filepath.Join(e.bin, tool)))
	}
	return e
}

func (e *installEnv) archiveName(name, version string) string {
	return name + "_" + strings.TrimPrefix(version, "v") + "_" + e.osName + "_" + e.arch + ".tar.gz"
}

// addArchive writes a tar.gz holding entry with content and returns its sha256.
func (e *installEnv) addArchive(file, entry, content string) string {
	e.t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(e.t, tw.WriteHeader(&tar.Header{Name: entry, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}))
	_, err := tw.Write([]byte(content))
	require.NoError(e.t, err)
	require.NoError(e.t, tw.Close())
	require.NoError(e.t, gz.Close())
	require.NoError(e.t, os.WriteFile(filepath.Join(e.fixtures, file), buf.Bytes(), 0o644))
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

func (e *installEnv) writeChecksums(lines ...string) {
	e.t.Helper()
	require.NoError(e.t, os.WriteFile(filepath.Join(e.fixtures, "checksums.txt"),
		[]byte(strings.Join(lines, "\n")+"\n"), 0o644))
}

// writeArchive writes a tar.gz of entries and returns its sha256.
func (e *installEnv) writeArchive(file string, entries ...archiveEntry) string {
	e.t.Helper()
	archive := makeTarGz(e.t, entries...)
	require.NoError(e.t, os.WriteFile(filepath.Join(e.fixtures, file), archive, 0o644))
	return sha256Hex(archive)
}

func (e *installEnv) run(args ...string) (string, error) {
	e.t.Helper()
	return e.runWithHome(true, args...)
}

// runWithHome runs install.sh with an empty HOME folder, or with HOME unset.
func (e *installEnv) runWithHome(withHome bool, args ...string) (string, error) {
	e.t.Helper()
	script, err := filepath.Abs("../../install.sh")
	require.NoError(e.t, err)
	cmd := exec.Command(filepath.Join(e.bin, "sh"), append([]string{script}, args...)...)
	cmd.Env = []string{"PATH=" + e.bin, "FIXTURES=" + e.fixtures}
	if withHome {
		cmd.Env = append(cmd.Env, "HOME="+e.t.TempDir())
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (e *installEnv) curlLog() string {
	b, err := os.ReadFile(filepath.Join(e.fixtures, "curl.log"))
	if os.IsNotExist(err) {
		return ""
	}
	require.NoError(e.t, err)
	return string(b)
}

func (e *installEnv) installed() bool {
	_, err := os.Stat(filepath.Join(e.dest, "tronctl"))
	return err == nil
}

func TestInstallScript_InstallsVerifiedArchive(t *testing.T) {
	for _, tc := range []struct {
		name, archive string
		args          []string
	}{
		{"tronctl", "tronctl", nil},
		{"tronctl-pq", "tronctl-pq", []string{"--pq"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newInstallEnv(t, true)
			file := e.archiveName(tc.archive, "v9.9.9")
			sum := e.addArchive(file, "tronctl", fakeBinary)
			other := e.archiveName("tronctl-other", "v9.9.9")
			e.writeChecksums(strings.Repeat("0", 64)+"  "+other, sum+"  "+file)

			out, err := e.run(append([]string{"--version", "v9.9.9", "--install-dir", e.dest}, tc.args...)...)
			require.NoError(t, err, out)
			require.Contains(t, out, "Checksum verified.")
			got, err := os.ReadFile(filepath.Join(e.dest, "tronctl"))
			require.NoError(t, err)
			require.Equal(t, fakeBinary, string(got))
		})
	}
}

func TestInstallScript_LatestVersion(t *testing.T) {
	e := newInstallEnv(t, true)
	require.NoError(t, os.WriteFile(filepath.Join(e.fixtures, "latest"),
		[]byte(`{"tag_name": "v9.9.8", "name": "v9.9.8"}`), 0o644))
	file := e.archiveName("tronctl", "v9.9.8")
	sum := e.addArchive(file, "tronctl", fakeBinary)
	e.writeChecksums(sum + "  " + file)

	out, err := e.run("--install-dir", e.dest)
	require.NoError(t, err, out)
	require.Contains(t, out, "tronctl v9.9.8 installed")
}

func TestInstallScript_FailsClosed(t *testing.T) {
	tests := []struct {
		name    string
		noSHA   bool
		setup   func(e *installEnv)
		args    []string
		wantErr string
	}{
		{
			name: "checksums.txt missing",
			setup: func(e *installEnv) {
				e.addArchive(e.archiveName("tronctl", "v9.9.9"), "tronctl", fakeBinary)
			},
			wantErr: "could not download checksums.txt",
		},
		{
			name: "archive not listed",
			setup: func(e *installEnv) {
				e.addArchive(e.archiveName("tronctl", "v9.9.9"), "tronctl", fakeBinary)
				e.writeChecksums(strings.Repeat("a", 64) + "  " + e.archiveName("tronctl", "v9.9.8"))
			},
			wantErr: "is not listed in checksums.txt",
		},
		{
			name: "name matched as a pattern is not enough",
			setup: func(e *installEnv) {
				file := e.archiveName("tronctl", "v9.9.9")
				sum := e.addArchive(file, "tronctl", fakeBinary)
				// "." in the real name would match any character in a regex.
				e.writeChecksums(sum + "  " + strings.ReplaceAll(file, ".", "x"))
			},
			wantErr: "is not listed in checksums.txt",
		},
		{
			name: "pq archive not published",
			setup: func(e *installEnv) {
				file := e.archiveName("tronctl", "v9.9.9")
				e.writeChecksums(e.addArchive(file, "tronctl", fakeBinary) + "  " + file)
			},
			args:    []string{"--pq"},
			wantErr: "tronctl-pq is published from v0.27.0",
		},
		{
			name: "checksum mismatch",
			setup: func(e *installEnv) {
				file := e.archiveName("tronctl", "v9.9.9")
				e.addArchive(file, "tronctl", fakeBinary)
				e.writeChecksums(strings.Repeat("b", 64) + "  " + file)
			},
			wantErr: "checksum mismatch",
		},
		{
			name: "malformed checksum",
			setup: func(e *installEnv) {
				file := e.archiveName("tronctl", "v9.9.9")
				sum := e.addArchive(file, "tronctl", fakeBinary)
				e.writeChecksums(sum[:63] + "  " + file)
			},
			wantErr: "malformed checksum",
		},
		{
			name:  "no sha256 tool",
			noSHA: true,
			setup: func(e *installEnv) {
				file := e.archiveName("tronctl", "v9.9.9")
				e.writeChecksums(e.addArchive(file, "tronctl", fakeBinary) + "  " + file)
			},
			wantErr: "no sha256 tool found",
		},
		{
			name: "archive listed twice",
			setup: func(e *installEnv) {
				file := e.archiveName("tronctl", "v9.9.9")
				sum := e.addArchive(file, "tronctl", fakeBinary)
				e.writeChecksums(sum+"  "+file, strings.Repeat("c", 64)+" *"+file)
			},
			wantErr: "is listed more than once in checksums.txt",
		},
		{
			name: "archive without tronctl",
			setup: func(e *installEnv) {
				file := e.archiveName("tronctl", "v9.9.9")
				e.writeChecksums(e.addArchive(file, "something-else", fakeBinary) + "  " + file)
			},
			wantErr: "tronctl not found in",
		},
		{
			name: "archive that is not a tar.gz",
			setup: func(e *installEnv) {
				file := e.archiveName("tronctl", "v9.9.9")
				body := []byte("not a gzip stream\n")
				require.NoError(e.t, os.WriteFile(filepath.Join(e.fixtures, file), body, 0o644))
				e.writeChecksums(sha256Hex(body) + "  " + file)
			},
			wantErr: "could not read",
		},
		{
			// The link target is a file the test owns, so only the symlink
			// check, not a failing chmod of a system file, can stop it.
			name: "tronctl is a symlink",
			setup: func(e *installEnv) {
				victim := filepath.Join(e.t.TempDir(), "victim")
				require.NoError(e.t, os.WriteFile(victim, []byte(fakeBinary), 0o644))
				file := e.archiveName("tronctl", "v9.9.9")
				sum := e.writeArchive(file, archiveEntry{name: "tronctl", typeflag: tar.TypeSymlink, linkname: victim})
				e.writeChecksums(sum + "  " + file)
			},
			wantErr: "is not a regular file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newInstallEnv(t, !tt.noSHA)
			tt.setup(e)
			out, err := e.run(append([]string{"--version", "v9.9.9", "--install-dir", e.dest}, tt.args...)...)
			require.Error(t, err, out)
			require.Contains(t, out, tt.wantErr)
			require.False(t, e.installed(), "nothing may be installed when verification fails")
		})
	}
}

func TestInstallScript_RejectsBadInput(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"path in version", []string{"--version", "v1.0.0/../../evil"}, "invalid version"},
		{"shell characters", []string{"--version", "v1.0.0;rm"}, "invalid version"},
		{"not a release tag", []string{"--version", "latest"}, "invalid version"},
		{"unknown option", []string{"--verison", "v1.0.0"}, "unknown option: --verison"},
		{"missing value", []string{"--version"}, "--version needs a value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newInstallEnv(t, true)
			out, err := e.run(append([]string{"--install-dir", e.dest}, tt.args...)...)
			require.Error(t, err, out)
			require.Contains(t, out, tt.wantErr)
			require.Empty(t, e.curlLog(), "nothing may be downloaded for bad input")
		})
	}
}

func TestInstallScript_DryRunDownloadsNothing(t *testing.T) {
	e := newInstallEnv(t, true)
	out, err := e.run("--version", "v9.9.9", "--pq", "--dry-run", "--install-dir", e.dest)
	require.NoError(t, err, out)
	require.Contains(t, out, e.archiveName("tronctl-pq", "v9.9.9"))
	require.Empty(t, e.curlLog())
	require.False(t, e.installed())
}

// Without --install-dir, an unwritable /usr/local/bin and no sudo, install.sh
// falls back to ~/.local/bin; with HOME unset it must say so, not fail on
// set -u.
func TestInstallScript_UnsetHome(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"dry run", []string{"--dry-run"}},
		{"install", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newInstallEnv(t, true)
			// install.sh's own check, run safely: only a dry run that picks
			// the fallback proves the real run would not write to the default
			// folder.
			probe, err := e.run("--version", "v9.9.9", "--dry-run")
			require.NoError(t, err, probe)
			if !strings.Contains(probe, "(fallback)") {
				t.Skip("the default install folder is writable here, so the fallback is not reached")
			}
			file := e.archiveName("tronctl", "v9.9.9")
			e.writeChecksums(e.addArchive(file, "tronctl", fakeBinary) + "  " + file)

			out, err := e.runWithHome(false, append([]string{"--version", "v9.9.9"}, tt.args...)...)

			require.Error(t, err, out)
			require.Contains(t, out, "HOME is not set")
			require.Contains(t, out, "--install-dir")
			require.NotContains(t, out, "parameter not set")
			require.NotContains(t, out, "unbound variable")
		})
	}
}
