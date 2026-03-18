package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

const installScriptURL = "https://raw.githubusercontent.com/fbsobreira/gotron-sdk/master/install.sh"

func init() {
	cmdUpgrade := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade tronctl to the latest version",
		Long: `Download and replace the current binary with the latest release from GitHub.

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

	fmt.Fprintf(os.Stderr, "Upgrading tronctl: %s → %s\n", currentVersion, targetVersion)

	// run install.sh to perform the actual upgrade
	if err := runInstallScript(targetVersion); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s Successfully upgraded tronctl: %s → %s\n",
		color.GreenString("✓"), currentVersion, targetVersion)
	return nil
}

func runInstallScript(version string) error {
	// check for required tools
	if _, err := exec.LookPath("curl"); err != nil {
		return fmt.Errorf("curl is required but not found in PATH")
	}
	shell, err := exec.LookPath("sh")
	if err != nil {
		return fmt.Errorf("sh is required but not found in PATH")
	}

	args := []string{"-s", "--", "--version", version}

	// pipe: curl -fsSL <url> | sh -s -- --version <version>
	curl := exec.Command("curl", "-fsSL", installScriptURL)
	sh := exec.Command(shell, args...)

	sh.Stdout = os.Stdout
	sh.Stderr = os.Stderr

	pipe, err := curl.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	sh.Stdin = pipe

	if err := curl.Start(); err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	if err := sh.Start(); err != nil {
		_ = curl.Process.Kill()
		_ = curl.Wait()
		return fmt.Errorf("failed to start installer: %w", err)
	}

	if err := curl.Wait(); err != nil {
		_ = sh.Wait()
		return fmt.Errorf("failed to download install script: %w", err)
	}
	if err := sh.Wait(); err != nil {
		return fmt.Errorf("install script failed: %w", err)
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

	// Homebrew: binary lives under Homebrew's Cellar directory
	if strings.Contains(execPath, "/Cellar/") || strings.Contains(execPath, "/homebrew/") {
		return "this binary was installed via Homebrew; upgrade with:\n  brew upgrade tronctl", true
	}

	// go install: check build info for a real module version.
	// goreleaser sets VersionWrapDump via ldflags with a "v" prefix (e.g. "v0.25.1-abc1234"),
	// while go install builds leave it empty — so an absent or non-v-prefixed value means
	// the binary was installed via go install.
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		if VersionWrapDump == "" || !strings.HasPrefix(strings.Split(VersionWrapDump, "-")[0], "v") {
			return "this binary was installed via `go install`; upgrade with:\n  go install github.com/fbsobreira/gotron-sdk/cmd/tronctl@latest", true
		}
	}

	return "", false
}

func getCurrentVersion() string {
	parts := strings.Split(VersionWrapDump, "-")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "unknown"
}

func fetchRelease(version string) (*GitHubRelease, error) {
	var url string
	if version != "" {
		// ensure version has v prefix
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		url = "https://api.github.com/repos/fbsobreira/gotron-sdk/releases/tags/" + version
	} else {
		url = versionLink
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // URL is constructed from constants
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
