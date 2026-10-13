package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	upgradeCheck   bool
	upgradeVersion string
)

const (
	checksumsAssetName = "checksums.txt"
	// releaseInfoTimeout bounds the GitHub API call that looks the release up.
	releaseInfoTimeout = 30 * time.Second
)

var (
	// releasesAPIBase is the GitHub releases API of this repository; tests
	// point it at an httptest server.
	releasesAPIBase = "https://api.github.com/repos/fbsobreira/gotron-sdk/releases"
	// upgradeHTTPClient serves the release lookup and the asset downloads. It
	// has no overall timeout, which would fail a valid download on a slow link:
	// the transport bounds connecting and waiting for headers, downloadAsset
	// aborts a body that stops arriving (upgradeIdleTimeout), and Ctrl-C
	// cancels the whole upgrade.
	upgradeHTTPClient = &http.Client{Transport: newUpgradeTransport()}
	// upgradeIdleTimeout aborts a download when no data arrives for this long.
	upgradeIdleTimeout = 60 * time.Second
	// Size caps for what upgrade reads off the wire or out of the archive.
	maxChecksumsSize int64 = 1 << 20
	maxArchiveSize   int64 = 256 << 20
	maxBinarySize    int64 = 256 << 20
	// upgradeVersionTimeout bounds the `version` run of the downloaded binary.
	upgradeVersionTimeout = 30 * time.Second
	// versionOutputRe finds the version in `tronctl version` output, e.g.
	// "TronCTL. tronctl version v0.26.0-abc1234 (goreleaser@gotron.sh ...)". It is
	// anchored to the start of a line so that nothing before the program name,
	// such as a folder called "version x" in a path, can be taken for it.
	versionOutputRe = regexp.MustCompile(`(?m)^TronCTL\. \S+ version (\S+)`)
	// commitSuffixPattern is the "-<commit>" that release builds append to the
	// tag: an abbreviated or full lowercase hex commit hash.
	commitSuffixPattern = `-[0-9a-f]{7,40}`
	commitSuffixRe      = regexp.MustCompile(commitSuffixPattern + `$`)
)

// ReleaseBuild reports that main set the version through release ldflags
// (goreleaser, the Makefile, scripts/tronctl-pq.sh). It is false for `go install`
// and plain `go build` binaries, whose version main reads from the build info.
var ReleaseBuild bool

func init() {
	cmdUpgrade := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade tronctl to the latest version",
		Long: `Download the latest release from GitHub and replace the current binary in place.

The release archive for this platform is checked against the release's
checksums.txt (SHA-256) before anything is installed, and the new binary must
run and report the expected version before it replaces the current one. Any
failure leaves the current binary untouched. A tronctl-pq build (FN-DSA-512)
upgrades only to a tronctl-pq archive, never to the default build.

By default, upgrades to the latest stable release. Use --check to only
check for updates without installing, or --version to target a specific release.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil // skip gRPC connection setup
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade()
		},
	}

	cmdUpgrade.Flags().BoolVar(&upgradeCheck, "check", false, "Only check if an update is available")
	cmdUpgrade.Flags().StringVar(&upgradeVersion, "version", "", "Upgrade/downgrade to a specific version tag")

	RootCmd.AddCommand(cmdUpgrade)
}

func runUpgrade() error {
	// detect install method and warn if self-upgrade is not appropriate
	if msg, ok := detectManagedInstall(); ok {
		return fmt.Errorf("%s", msg)
	}

	currentVersion := getCurrentVersion()

	// fetch target release
	release, err := fetchRelease(upgradeVersion)
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}

	targetVersion := release.TagName

	if upgradeCheck {
		return printUpdateStatus(currentVersion, targetVersion)
	}

	if currentVersion == targetVersion {
		fmt.Fprintf(os.Stderr, "%s tronctl is already up to date (%s)\n",
			color.GreenString("✓"), currentVersion)
		return nil
	}

	target, err := upgradeTargetPath()
	if err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Ctrl-C, SIGTERM or a closed terminal cancels the download or the version check, so the temp file is
	// still cleaned up.
	ctx, stop := signal.NotifyContext(context.Background(), upgradeSignals()...)
	defer stop()
	opts := upgradeOptions{
		client:  upgradeHTTPClient,
		out:     os.Stderr,
		current: currentVersion,
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
		pq:      pqBuild,
	}
	if err := upgradeTo(ctx, release, target, opts); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s Successfully upgraded tronctl: %s → %s\n",
		color.GreenString("✓"), currentVersion, targetVersion)
	return nil
}

// upgradeTargetPath returns the real path of the running binary.
func upgradeTargetPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot locate the running binary: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve the running binary: %w", err)
	}
	return execPath, nil
}

// upgradeOptions carries what upgradeTo needs from its environment, so tests
// can swap in an httptest client and another platform's asset.
type upgradeOptions struct {
	client  *http.Client
	out     io.Writer // progress messages
	current string    // the version being replaced, for the progress message
	goos    string    // selects the archive and the binary name inside it
	goarch  string
	pq      bool // install a tronctl-pq archive
}

// upgradeTo installs release over target: it downloads the archive for
// opts.goos/opts.goarch and checksums.txt, verifies the archive, extracts the
// binary, checks that it runs and reports release.TagName, and then replaces
// target. Any failure leaves target untouched and no temp file behind.
func upgradeTo(ctx context.Context, release *GitHubRelease, target string, opts upgradeOptions) error {
	tag := release.TagName
	if tag == "" {
		return errors.New("the release has no tag name")
	}
	if runtime.GOOS == "windows" {
		// Best-effort: drop the backup the previous upgrade had to leave behind.
		// replaceExecutableWithBackup retries this and reports a failure.
		_ = os.Remove(target + ".old")
	}

	archiveName := upgradeArchiveName(tag, opts.goos, opts.goarch, opts.pq)
	archiveAsset, ok := findReleaseAsset(release, archiveName)
	if !ok {
		if opts.pq {
			// Never fall back to the default archive: it cannot sign with FN-DSA-512.
			return fmt.Errorf("release %s has no tronctl-pq archive for %s/%s (%s). This tronctl is a "+
				"tronctl-pq build with FN-DSA-512 signing and will not be replaced by the default build, "+
				"which cannot sign with FN-DSA-512. tronctl-pq is published from v0.27.0 for linux/amd64, "+
				"linux/arm64, darwin/amd64, darwin/arm64 and windows/amd64; elsewhere, build from source "+
				"with -tags falcon", tag, opts.goos, opts.goarch, archiveName)
		}
		return fmt.Errorf("release %s has no %s for %s/%s", tag, archiveName, opts.goos, opts.goarch)
	}
	checksumsAsset, ok := findReleaseAsset(release, checksumsAssetName)
	if !ok {
		return fmt.Errorf("release %s has no %s; refusing to install an unverified binary", tag, checksumsAssetName)
	}
	// Progress output is best-effort: a broken stderr must not abort an upgrade.
	_, _ = fmt.Fprintf(opts.out, "Upgrading tronctl: %s → %s\n", opts.current, tag)

	checksums, err := downloadAsset(ctx, opts.client, checksumsAsset.URL, maxChecksumsSize)
	if err != nil {
		return fmt.Errorf("download %s: %w", checksumsAssetName, err)
	}
	want, err := checksumFor(checksums, archiveName)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(opts.out, "Downloading %s\n", archiveName) // best-effort progress output
	archive, err := downloadAsset(ctx, opts.client, archiveAsset.URL, maxArchiveSize)
	if err != nil {
		return fmt.Errorf("download %s: %w", archiveName, err)
	}
	got := sha256.Sum256(archive)
	if subtle.ConstantTimeCompare(got[:], want) != 1 {
		return fmt.Errorf("checksum mismatch for %s: refusing to install it", archiveName)
	}
	_, _ = fmt.Fprintln(opts.out, "Checksum verified.") // best-effort progress output

	bin, err := extractBinary(archive, opts.goos)
	if err != nil {
		return fmt.Errorf("%s: %w", archiveName, err)
	}
	return installBinary(ctx, target, bin, tag)
}

// upgradeArchiveName is the release asset name for a platform, e.g.
// tronctl_0.27.0_linux_amd64.tar.gz or tronctl-pq_0.27.0_windows_amd64.zip.
func upgradeArchiveName(tag, goos, goarch string, pq bool) string {
	prefix := "tronctl"
	if pq {
		prefix = "tronctl-pq"
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s_%s_%s_%s%s", prefix, strings.TrimPrefix(tag, "v"), goos, goarch, ext)
}

func upgradeBinaryName(goos string) string {
	if goos == "windows" {
		return "tronctl.exe"
	}
	return "tronctl"
}

func findReleaseAsset(release *GitHubRelease, name string) (GitHubReleaseAssets, bool) {
	for _, a := range release.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return GitHubReleaseAssets{}, false
}

// httpsOnlyRedirects refuses to follow a redirect away from https.
func httpsOnlyRedirects(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" {
		return fmt.Errorf("refusing redirect to non-https URL %s", req.URL.Redacted())
	}
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

// downloadAsset GETs an https URL and returns its body, which must be HTTP 200
// and at most limit bytes.
// newUpgradeTransport is http.DefaultTransport with bounded connection setup
// and response-header waits, so a dead server fails fast without limiting how
// long a body that keeps arriving may take.
func newUpgradeTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	t.TLSHandshakeTimeout = 30 * time.Second
	t.ResponseHeaderTimeout = 30 * time.Second
	return t
}

// errDownloadStalled is the cancel cause when no data arrives for
// upgradeIdleTimeout.
var errDownloadStalled = errors.New("download stalled")

// idleTimeoutReader pushes back a stall timer each time data arrives.
type idleTimeoutReader struct {
	r     io.Reader
	timer *time.Timer
	idle  time.Duration
}

func (r *idleTimeoutReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.timer.Reset(r.idle)
	}
	return n, err
}

func downloadAsset(ctx context.Context, client *http.Client, rawURL string, limit int64) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid download URL: %w", err)
	}
	if u.Scheme != "https" || u.Host == "" {
		return nil, fmt.Errorf("refusing non-https download URL %q", u.Redacted())
	}
	// A stall timer, pushed back by every chunk of data, bounds how long the
	// download may go without progress instead of how long it may take.
	idle := upgradeIdleTimeout
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stall := time.AfterFunc(idle, func() { cancel(errDownloadStalled) })
	defer stall.Stop()
	stalled := func(err error) error {
		if errors.Is(context.Cause(ctx), errDownloadStalled) {
			return fmt.Errorf("%w: no data received for %s from %s", errDownloadStalled, idle, u.Redacted())
		}
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	httpsClient := *client
	httpsClient.CheckRedirect = httpsOnlyRedirects
	resp, err := httpsClient.Do(req)
	if err != nil {
		return nil, stalled(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, u.Redacted())
	}
	data, err := readCapped(&idleTimeoutReader{r: resp.Body, timer: stall, idle: idle}, limit)
	if err != nil {
		return nil, stalled(err)
	}
	// A response can still complete after the stall timer cancelled the
	// request: do not trust what arrived. Ask the context, not the timer: a
	// chunk read after the cancel restarts the timer, so Stop alone cannot
	// tell that it fired. A cancel that had not landed yet did not affect the
	// response.
	stall.Stop()
	if errors.Is(context.Cause(ctx), errDownloadStalled) {
		return nil, stalled(nil)
	}
	return data, nil
}

// readCapped reads all of r, failing (instead of truncating) past limit bytes.
func readCapped(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("larger than %d bytes", limit)
	}
	return data, nil
}

// checksumFor returns the SHA-256 that checksums.txt lists for name. Lines are
// "<hex>  <name>" or "<hex> *<name>"; name must appear exactly once.
func checksumFor(checksums []byte, name string) ([]byte, error) {
	var want []byte
	found := false
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		if found {
			return nil, fmt.Errorf("%s is listed more than once in %s", name, checksumsAssetName)
		}
		found = true
		sum, err := hex.DecodeString(fields[0])
		if err != nil || len(sum) != sha256.Size {
			return nil, fmt.Errorf("malformed checksum for %s in %s", name, checksumsAssetName)
		}
		want = sum
	}
	if !found {
		return nil, fmt.Errorf("%s is not listed in %s; refusing to install an unverified binary", name, checksumsAssetName)
	}
	return want, nil
}

// archiveEntryIsBinary reports whether an archive entry is the tronctl binary.
// Entries that could escape the archive root, and any path that ends in the
// binary name but is not exactly it, fail the whole archive.
func archiveEntryIsBinary(name, binName string) (bool, error) {
	if name == "" || strings.Contains(name, `\`) || strings.Contains(name, "..") ||
		path.IsAbs(name) || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return false, fmt.Errorf("unsafe archive entry %q", name)
	}
	if name == binName {
		return true, nil
	}
	if path.Base(name) == binName {
		return false, fmt.Errorf("unsafe archive entry %q: %s must be at the archive root", name, binName)
	}
	return false, nil
}

// extractBinary returns the root tronctl entry of a release archive: a .zip
// for windows, a .tar.gz otherwise.
func extractBinary(archive []byte, goos string) ([]byte, error) {
	binName := upgradeBinaryName(goos)
	if goos == "windows" {
		return extractFromZip(archive, binName)
	}
	return extractFromTarGz(archive, binName)
}

func extractFromTarGz(archive []byte, binName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	var bin []byte
	found := false
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, tar.ErrInsecurePath) && hdr != nil {
			err = nil // archiveEntryIsBinary judges the name below
		}
		if err != nil {
			return nil, fmt.Errorf("read archive: %w", err)
		}
		isBin, err := archiveEntryIsBinary(hdr.Name, binName)
		if err != nil {
			return nil, err
		}
		if !isBin {
			continue
		}
		if found {
			return nil, fmt.Errorf("archive contains %s more than once", binName)
		}
		found = true
		if hdr.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("archive entry %s is not a regular file", binName)
		}
		if bin, err = readCapped(tr, maxBinarySize); err != nil {
			return nil, fmt.Errorf("archive entry %s: %w", binName, err)
		}
	}
	if !found {
		return nil, fmt.Errorf("archive does not contain %s", binName)
	}
	return bin, nil
}

func extractFromZip(archive []byte, binName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if errors.Is(err, zip.ErrInsecurePath) {
		err = nil // zr is usable; archiveEntryIsBinary judges each name below
	}
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}
	var bin []byte
	found := false
	for _, f := range zr.File {
		isBin, err := archiveEntryIsBinary(f.Name, binName)
		if err != nil {
			return nil, err
		}
		if !isBin {
			continue
		}
		if found {
			return nil, fmt.Errorf("archive contains %s more than once", binName)
		}
		found = true
		if !f.Mode().IsRegular() {
			return nil, fmt.Errorf("archive entry %s is not a regular file", binName)
		}
		if bin, err = readZipFile(f); err != nil {
			return nil, fmt.Errorf("archive entry %s: %w", binName, err)
		}
	}
	if !found {
		return nil, fmt.Errorf("archive does not contain %s", binName)
	}
	return bin, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	data, err := readCapped(rc, maxBinarySize)
	if cerr := rc.Close(); err == nil && cerr != nil {
		return nil, cerr
	}
	return data, err
}

// installBinary writes bin next to target, checks that it runs and reports
// tag, and then swaps it in. On any failure the temp file is removed and
// target is left as it was.
func installBinary(ctx context.Context, target string, bin []byte, tag string) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("cannot read the current binary: %w", err)
	}
	dir := filepath.Dir(target)
	// Same directory as target: the rename stays on one filesystem (atomic),
	// and the version check does not depend on an exec-capable /tmp.
	// Keep target's extension so Windows can run the temp file.
	tmp, err := os.CreateTemp(dir, ".tronctl-upgrade-*"+filepath.Ext(target))
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return notWritableError(dir, err)
		}
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	fail := func(err error) error {
		_ = os.Remove(tmpPath) // best-effort cleanup; target is untouched
		return err
	}
	if _, err := tmp.Write(bin); err != nil {
		_ = tmp.Close() // best-effort: the write error is the one to report
		return fail(fmt.Errorf("write new binary: %w", err))
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close() // best-effort: the sync error is the one to report
		return fail(fmt.Errorf("sync new binary: %w", err))
	}
	if err := tmp.Close(); err != nil {
		return fail(fmt.Errorf("close new binary: %w", err))
	}
	// Keep the current binary's mode (a 0o750 install stays 0o750); only make
	// sure its owner can run it.
	perm := info.Mode().Perm()
	if runtime.GOOS != "windows" {
		perm |= 0o100
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fail(fmt.Errorf("chmod new binary: %w", err))
	}
	if err := checkNewBinary(ctx, tmpPath, tag); err != nil {
		return fail(err)
	}
	if err := replaceExecutable(target, tmpPath); err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return fail(notWritableError(dir, err))
		}
		return fail(fmt.Errorf("replace %s: %w", target, err))
	}
	return nil
}

func notWritableError(dir string, err error) error {
	return fmt.Errorf("the folder %s is not writable (%w): rerun `tronctl upgrade` with sudo "+
		"(from an administrator shell on Windows), or reinstall with install.sh", dir, err)
}

// checkNewBinary runs `<bin> version` and requires exit 0 and a reported
// version of tag (optionally followed by "-<commit>").
func checkNewBinary(ctx context.Context, bin, tag string) error {
	ctx, cancel := context.WithTimeout(ctx, upgradeVersionTimeout)
	defer cancel()
	// tronctl reads .env from the working directory and its config and
	// keystore from the home directory, and `version` still resolves the
	// signer, so run it in an empty directory with that as its home.
	scratch, err := os.MkdirTemp("", "tronctl-upgrade-check-*")
	if err != nil {
		return fmt.Errorf("create scratch directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratch) }() // best-effort cleanup of the scratch directory
	cmd := exec.CommandContext(ctx, bin, "version")
	// main prints path.Base(os.Args[0]), which does not split a Windows path on
	// `\`, so the temp file's full path (and any "version " in a folder name)
	// would land in the output. cmd.Path still names the real file.
	cmd.Args[0] = "tronctl"
	cmd.Dir = scratch
	cmd.Env = versionCheckEnv(scratch)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if len(output) > 512 {
		output = output[:512] + "..."
	}
	if err != nil {
		return fmt.Errorf("the downloaded binary failed to run `version`: %w (output: %q)", err, output)
	}
	if !upgradeVersionMatches(output, tag) {
		reported := "nothing"
		if m := versionOutputRe.FindStringSubmatch(output); m != nil {
			reported = m[1]
		}
		return fmt.Errorf("the downloaded binary reports version %s, expected %s", reported, tag)
	}
	return nil
}

// versionCheckEnv is the current environment without tronctl's own settings,
// with home pointed at an empty directory.
func versionCheckEnv(home string) []string {
	var env []string
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		switch k := strings.ToUpper(key); {
		case strings.HasPrefix(k, "TRONCTL_"), k == "TRONGRID_APIKEY", k == "GOTRON_SDK_DEBUG",
			k == "HOME", k == "USERPROFILE", k == "XDG_CONFIG_HOME":
			continue
		}
		env = append(env, kv)
	}
	return append(env, "HOME="+home, "USERPROFILE="+home)
}

// upgradeVersionMatches reports whether `tronctl version` output names tag,
// exactly or followed by "-<commit>" (7 to 40 lowercase hex digits):
// v1.0.0-rc1 matches v1.0.0-rc1-abc1234, while v1.0.0 does not match
// v1.0.0-rc1-abc1234 or v1.0.0-abc1234-dirty.
func upgradeVersionMatches(output, tag string) bool {
	if tag == "" {
		return false
	}
	m := versionOutputRe.FindStringSubmatch(output)
	if m == nil {
		return false
	}
	want := regexp.MustCompile(`^` + regexp.QuoteMeta(tag) + `(?:` + commitSuffixPattern + `)?$`)
	return want.MatchString(m[1])
}

// replaceExecutable moves newPath over target. Windows cannot rename over a
// running executable, so there the current binary is moved aside first.
func replaceExecutable(target, newPath string) error {
	if runtime.GOOS == "windows" {
		return replaceExecutableWithBackup(target, newPath)
	}
	return os.Rename(newPath, target)
}

// replaceExecutableWithBackup renames target to target.old and newPath to
// target, restoring target if the second rename fails. target.old stays (a
// running executable cannot be deleted on Windows); the next upgrade removes it.
func replaceExecutableWithBackup(target, newPath string) error {
	old := target + ".old"
	if err := os.Remove(old); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove the previous backup %s: %w", old, err)
	}
	if err := os.Rename(target, old); err != nil {
		return fmt.Errorf("move the current binary aside: %w", err)
	}
	if err := os.Rename(newPath, target); err != nil {
		if rerr := os.Rename(old, target); rerr != nil {
			return fmt.Errorf("install the new binary: %w; restoring the previous binary also failed (%v), "+
				"rename %s back to %s", err, rerr, old, target)
		}
		return fmt.Errorf("install the new binary: %w", err)
	}
	return nil
}

// detectManagedInstall checks if the binary was installed via a package manager
// or go install, returning an appropriate upgrade message if so.
func detectManagedInstall() (string, bool) {
	execPath, err := os.Executable()
	if err != nil {
		return "", false
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", false
	}
	mainVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		mainVersion = info.Main.Version
	}
	return managedInstallHint(execPath, ReleaseBuild, mainVersion)
}

// managedInstallHint decides whether the binary at execPath belongs to a
// package manager or to `go install`, and if so returns how to upgrade it.
//
// main fills its version from the build info when no ldflags set it, so a
// `go install ...@v0.26.0` binary reports v0.26.0 just like a release does;
// only releaseBuild (ldflags set the version) tells the two apart. mainVersion
// is the build info's Main.Version: a module version for `go install`, and
// "(devel)" or empty for a build without one.
func managedInstallHint(execPath string, releaseBuild bool, mainVersion string) (string, bool) {
	// Homebrew: binary lives under Homebrew's Cellar directory
	if strings.Contains(execPath, "/Cellar/") || strings.Contains(execPath, "/homebrew/") {
		return "this binary was installed via Homebrew; upgrade with:\n  brew upgrade tronctl", true
	}
	// Refused: the build info carries a real module version (go install, or a
	// VCS-stamped go build of a checkout) and no ldflags set the version.
	// Replacing such a binary in place would overwrite something the user
	// built. Builds with ldflags (goreleaser, scripts/tronctl-pq.sh, make) and
	// "(devel)" builds without VCS information are not refused.
	if !releaseBuild && mainVersion != "" && mainVersion != "(devel)" {
		return "this binary was not built from a release (it came from `go install` or `go build`); upgrade with:\n" +
			"  go install github.com/fbsobreira/gotron-sdk/cmd/tronctl@latest\n" +
			"or download a release from https://github.com/fbsobreira/gotron-sdk/releases", true
	}
	return "", false
}

// getCurrentVersion is the release tag this binary was built from:
// VersionWrapDump without the trailing "-<commit>", so a prerelease such as
// v1.0.0-rc1-abc1234 stays v1.0.0-rc1.
func getCurrentVersion() string {
	if v := commitSuffixRe.ReplaceAllString(VersionWrapDump, ""); v != "" {
		return v
	}
	return "unknown"
}

func fetchRelease(version string) (*GitHubRelease, error) {
	endpoint := releasesAPIBase + "/latest"
	if version != "" {
		// ensure version has v prefix
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		endpoint = releasesAPIBase + "/tags/" + url.PathEscape(version)
	}

	ctx, cancel := context.WithTimeout(context.Background(), releaseInfoTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	// Same redirect policy as the asset downloads: never follow one off https.
	client := *upgradeHTTPClient
	client.CheckRedirect = httpsOnlyRedirects
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		if version != "" {
			return nil, fmt.Errorf("release %s not found (HTTP %d)", version, resp.StatusCode)
		}
		return nil, fmt.Errorf("could not fetch latest release (HTTP %d)", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}
	return &release, nil
}

func printUpdateStatus(current, latest string) error {
	if current == latest {
		fmt.Fprintf(os.Stderr, "%s tronctl is up to date (%s)\n",
			color.GreenString("✓"), current)
	} else {
		fmt.Fprintf(os.Stderr, "%s Update available: %s → %s\n",
			color.YellowString("!"), current, latest)
		fmt.Fprintf(os.Stderr, "  Run `tronctl upgrade` to install\n")
	}
	return nil
}
