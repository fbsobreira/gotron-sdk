package cmd

import (
	"net"
	"os"

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

// setupKeystore sets the default keystore directory if provided.
func (r *Runtime) setupKeystore() {
	if len(r.DefaultKeystoreDir) > 0 {
		store.SetDefaultLocation(r.DefaultKeystoreDir)
	}
}
