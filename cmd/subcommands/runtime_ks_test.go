package cmd

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runtimeKSTestKey is a throwaway secp256k1 key used only by these tests.
const runtimeKSTestKey = "e9a6e2a4e8e050b8616870520abc3c61f0368a4aaee3c3f0e742cf29e4cc8501" // gitleaks:allow

// TestPersistentPreRunResolvesSignerNameInKsDir checks that -s <account-name>
// is looked up in the --ks-dir keystore. The keystore location must be applied
// before the signer is resolved, or names that exist only in the custom
// directory fail with "Invalid account name".
func TestPersistentPreRunResolvesSignerNameInKsDir(t *testing.T) {
	ksDir := pqTestEnv(t, "pw")

	// ECDSA account in the --ks-dir keystore.
	sk, err := keys.GetPrivateKeyFromHex(runtimeKSTestKey)
	require.NoError(t, err)
	ks := store.FromAccountName("ks-ecdsa")
	ecdsaAcct, err := ks.ImportECDSA(sk.ToECDSA(), "pw")
	require.NoError(t, err)
	ks.Close()

	// Post-quantum account in the same keystore.
	pqAcct := savePQAccount(t, "ks-pq", "pw")

	// Point the store somewhere else, as the default location would be
	// before the --ks-dir flag is applied.
	store.SetDefaultLocation(t.TempDir())
	require.False(t, store.NameExists("ks-ecdsa"))
	require.False(t, store.NameExists("ks-pq"))

	t.Setenv("TRONCTL_KS_DIR", "")
	t.Setenv("TRONCTL_SIGNER", "")

	prevRT := *rt
	prevConn, prevSigner, prevSignerAddr := conn, signer, signerAddress
	prevKsDir, prevNode := defaultKeystoreDir, node
	t.Cleanup(func() {
		if rt.Conn != nil {
			rt.Conn.Stop()
		}
		*rt = prevRT
		conn, signer, signerAddress = prevConn, prevSigner, prevSignerAddr
		defaultKeystoreDir, node = prevKsDir, prevNode
	})

	tests := []struct {
		name     string
		signer   string
		wantAddr string
	}{
		{name: "ECDSA account name", signer: "ks-ecdsa", wantAddr: ecdsaAcct.Address.String()},
		{name: "post-quantum account name", signer: "ks-pq", wantAddr: pqAcct.Address},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rt.Conn != nil {
				rt.Conn.Stop()
			}
			*rt = Runtime{}
			// Each case starts with the store pointed away from --ks-dir.
			store.SetDefaultLocation(t.TempDir())
			// grpc.NewClient is lazy: nothing is dialled here.
			node = "127.0.0.1:50051"
			defaultKeystoreDir = ksDir
			signer = tt.signer

			require.NoError(t, RootCmd.PersistentPreRunE(RootCmd, nil))
			assert.Equal(t, tt.wantAddr, rt.SignerAddress.String())
			assert.Equal(t, tt.wantAddr, signerAddress.String())
		})
	}
}
