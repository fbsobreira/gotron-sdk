//go:build falcon

package cmd

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKeysAddAndSignFNDSA creates an FN-DSA-512 key through keys add,
// resolves its signer and signs a transaction with it, on a Falcon build.
func TestKeysAddAndSignFNDSA(t *testing.T) {
	pqTestEnv(t, "pw")
	cmd := keysAddCmd()
	require.NoError(t, cmd.Flags().Set("scheme", "fn-dsa-512"))
	out, _, err := runKeysCmd(t, cmd, "fn-key")
	require.NoError(t, err)

	acct, err := store.PQAccountByName("fn-key")
	require.NoError(t, err)
	assert.Equal(t, core.PQScheme_FN_DSA_512, acct.Scheme)
	assert.False(t, store.DoesNamedAccountExist("fn-key"), "no ECDSA account may be created")
	assert.Contains(t, out, acct.Address)
	assert.Contains(t, out, "FN_DSA_512")

	var a tronAddress
	require.NoError(t, a.Set(acct.Address))
	s, err := resolveSigner(a)
	require.NoError(t, err)
	pqs, ok := s.(interface{ PQScheme() core.PQScheme })
	require.True(t, ok, "resolveSigner must return a post-quantum signer, got %T", s)
	assert.Equal(t, core.PQScheme_FN_DSA_512, pqs.PQScheme())
	assert.Equal(t, acct.Address, s.Address().String())

	tx := &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}}
	signed, err := s.Sign(tx)
	require.NoError(t, err)
	require.Len(t, signed.GetPqAuthSig(), 1)
	sig := signed.GetPqAuthSig()[0]
	assert.Equal(t, core.PQScheme_FN_DSA_512, sig.GetScheme())
	assert.NotEmpty(t, sig.GetSignature())
	assert.Empty(t, signed.GetSignature(), "a PQ-signed transaction carries no ECDSA signature")
}
