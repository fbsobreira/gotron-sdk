package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Runtime holds shared state for CLI command execution.
type Runtime struct {
	Conn               *client.GrpcClient
	SignerAddress      tronAddress
	Passphrase         string
	UseLedgerWallet    bool
	Verbose            bool
	DryRun             bool
	NoWait             bool
	NoPrettyOutput     bool
	Timeout            uint32
	Node               string
	APIKey             string
	WithTLS            bool
	GivenFilePath      string
	DefaultKeystoreDir string
}

// rt is the global runtime instance used by all subcommands.
var rt = &Runtime{}

// loadDotEnv loads a .env file from the current directory if present.
// Missing file is silently ignored — .env is optional.
func (r *Runtime) loadDotEnv() {
	_ = godotenv.Load() // .env in cwd; no error if missing
}

// setupVerbose enables verbose logging if requested.
func (r *Runtime) setupVerbose() {
	if r.Verbose {
		c.EnableAllVerbose()
	}
}

// setupNetwork initializes the gRPC connection and network options.
func (r *Runtime) setupNetwork() error {
	if _, _, err := net.SplitHostPort(r.Node); err != nil {
		r.Node = net.JoinHostPort(r.Node, "50051")
	}
	r.Conn = client.NewGrpcClient(r.Node)

	// load grpc options
	opts := make([]grpc.DialOption, 0)
	if r.WithTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	} else {
		opts = append(opts, client.GRPCInsecure())
	}

	// set API key
	if err := r.Conn.SetAPIKey(r.APIKey); err != nil {
		return err
	}

	return r.Conn.Start(opts...)
}

// setupSigner resolves the signer address from the signer flag.
func (r *Runtime) setupSigner(signer string) error {
	if len(signer) > 0 {
		var err error
		if r.SignerAddress, err = findAddress(signer); err != nil {
			return err
		}
	}
	return nil
}

// applyEnvOverrides reads environment variables and applies them when
// the corresponding flag was not explicitly set. Flags always win.
func (r *Runtime) applyEnvOverrides(flagNode, flagSigner string, flagTLS bool) string {
	if key := os.Getenv("TRONGRID_APIKEY"); len(key) > 0 && len(r.APIKey) == 0 {
		r.APIKey = key
	}
	if dir := os.Getenv("TRONCTL_KS_DIR"); len(dir) > 0 && len(r.DefaultKeystoreDir) == 0 {
		r.DefaultKeystoreDir = dir
	}
	if envNode := os.Getenv("TRONCTL_NODE"); len(envNode) > 0 && r.Node == flagNode {
		r.Node = envNode
	}
	if v := os.Getenv("TRONCTL_TLS"); !flagTLS && (v == "true" || v == "1") {
		r.WithTLS = true
	}
	signer := flagSigner
	if envSigner := os.Getenv("TRONCTL_SIGNER"); len(envSigner) > 0 && len(signer) == 0 {
		signer = envSigner
	}
	return signer
}

// setupKeystore points the store at the --ks-dir / TRONCTL_KS_DIR value, if
// one was given. The value is the config root that holds the account-keys
// and pq-keys folders, not a keystore folder itself; see resolveKsDir for how
// it becomes a path. Warnings go to w, best effort:
//   - a value whose last element is account-keys or pq-keys is almost always
//     meant as a key folder, so the warning names the folders actually read
//     and suggests the parent;
//   - a relative value used to be resolved against the home directory, so
//     while only that old location holds keys the warning names both paths.
func (r *Runtime) setupKeystore(w io.Writer) error {
	if len(r.DefaultKeystoreDir) == 0 {
		return nil
	}
	root, err := resolveKsDir(r.DefaultKeystoreDir)
	if err != nil {
		return err
	}
	if b := filepath.Base(root); b == c.DefaultConfigAccountAliasesDirName || b == c.DefaultConfigPQKeysDirName {
		_, _ = fmt.Fprintf(w, "warning: --ks-dir/TRONCTL_KS_DIR %s looks like a key folder, but it is used as the config root: "+
			"keys are read from %s and %s; pass the parent, %s, to use the folder itself\n",
			root, filepath.Join(root, c.DefaultConfigAccountAliasesDirName),
			filepath.Join(root, c.DefaultConfigPQKeysDirName), filepath.Dir(root))
	}
	if old, ok := legacyKsDir(r.DefaultKeystoreDir, root); ok {
		_, _ = fmt.Fprintf(w, "warning: --ks-dir/TRONCTL_KS_DIR %q now means %s (relative to the current directory), "+
			"but keys exist under %s; use an absolute path or ~/%s to keep using them\n",
			r.DefaultKeystoreDir, root, old, r.DefaultKeystoreDir)
	}
	store.SetDefaultLocation(root)
	return nil
}

// legacyKsDir reports the home-relative path a relative --ks-dir value used
// to mean, when that old path holds keys and root holds none.
func legacyKsDir(dir, root string) (string, bool) {
	if filepath.IsAbs(dir) || hasHomePrefix(dir, os.PathSeparator) || hasKeyFolders(root) {
		return "", false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	old := filepath.Join(home, dir)
	if old == root || !hasKeyFolders(old) {
		return "", false
	}
	return old, true
}

// hasKeyFolders reports whether the account-keys or pq-keys folder under
// root has any entry.
func hasKeyFolders(root string) bool {
	for _, sub := range []string{c.DefaultConfigAccountAliasesDirName, c.DefaultConfigPQKeysDirName} {
		if entries, err := os.ReadDir(filepath.Join(root, sub)); err == nil && len(entries) > 0 {
			return true
		}
	}
	return false
}

// hasHomePrefix reports whether dir is ~ or starts with ~ and a separator:
// / on any system, and also \ when sep is \ (Windows), since elsewhere \ is
// an ordinary filename character. ~user forms are not recognised.
func hasHomePrefix(dir string, sep byte) bool {
	if dir == "~" || strings.HasPrefix(dir, "~/") {
		return true
	}
	return sep == '\\' && strings.HasPrefix(dir, `~\`)
}

// resolveKsDir turns a --ks-dir / TRONCTL_KS_DIR value into an absolute
// path. A leading ~ is the home directory (see hasHomePrefix), and any other
// relative path is resolved against the current directory, the same
// directory a .env file is loaded from. (The store itself would resolve a
// relative path against the home directory.)
func resolveKsDir(dir string) (string, error) {
	if hasHomePrefix(dir, os.PathSeparator) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve --ks-dir %q: %w", dir, err)
		}
		return filepath.Join(home, dir[1:]), nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve --ks-dir %q: %w", dir, err)
	}
	return abs, nil
}
