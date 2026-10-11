package signer

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// txWithSignatures returns a transaction carrying ecdsa ECDSA signatures and
// pq pq_auth_sig entries (placeholder bytes; they are only counted).
func txWithSignatures(ecdsa, pq int) *core.Transaction {
	tx := newPQTestTx()
	for range ecdsa {
		tx.Signature = append(tx.Signature, make([]byte, 65))
	}
	for range pq {
		tx.PqAuthSig = append(tx.PqAuthSig, &core.PQAuthSig{PublicKey: []byte{1}, Signature: []byte{2}})
	}
	return tx
}

// TestECDSASigners_SignatureCap checks that every ECDSA signer counts
// signature and pq_auth_sig entries together and refuses to add one beyond
// MaxTotalSignatures, leaving the transaction unchanged, while one below the
// limit still signs.
func TestECDSASigners_SignatureCap(t *testing.T) {
	const pass = "cap-pass"
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	pkSigner, err := NewPrivateKeySigner(key)
	require.NoError(t, err)
	ks, acct := newTestKeystoreAndAccount(t, pass)
	require.NoError(t, ks.Unlock(acct, pass))

	signers := []struct {
		name string
		s    Signer
		// device is true when signing below the limit needs hardware.
		device bool
	}{
		{"private key", pkSigner, false},
		{"keystore", NewKeystoreSigner(ks, acct), false},
		{"keystore passphrase", NewKeystorePassphraseSigner(ks, acct, pass), false},
		{"ledger", NewLedgerSignerWithAddress(address.Address(acct.Address)), true},
	}
	full := []struct {
		name      string
		ecdsa, pq int
	}{
		{"ecdsa only", MaxTotalSignatures, 0},
		{"mixed", MaxTotalSignatures - 1, 1},
		{"pq only", 0, MaxTotalSignatures},
	}
	for _, sg := range signers {
		for _, f := range full {
			t.Run(sg.name+"/"+f.name, func(t *testing.T) {
				tx := txWithSignatures(f.ecdsa, f.pq)
				before := proto.Clone(tx)

				_, err := sg.s.Sign(tx)
				require.ErrorIs(t, err, ErrTooManySignatures)
				assert.True(t, proto.Equal(before, tx), "transaction must be left unchanged")
			})
		}
		if sg.device {
			continue
		}
		t.Run(sg.name+"/below limit", func(t *testing.T) {
			tx := txWithSignatures(MaxTotalSignatures-2, 1)
			_, err := sg.s.Sign(tx)
			require.NoError(t, err)
			assert.Len(t, tx.GetSignature(), MaxTotalSignatures-1)
		})
	}
}

// TestSignatureCapIsShared checks that the keystore check returns the same
// error the signers document, so errors.Is matches either name.
func TestSignatureCapIsShared(t *testing.T) {
	assert.ErrorIs(t, keystore.CheckSignatureRoom(txWithSignatures(MaxTotalSignatures, 0)), ErrTooManySignatures)
	assert.NoError(t, keystore.CheckSignatureRoom(txWithSignatures(MaxTotalSignatures-1, 0)))
	assert.NoError(t, keystore.CheckSignatureRoom(nil))
}
