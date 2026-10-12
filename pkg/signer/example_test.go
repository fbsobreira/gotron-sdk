package signer_test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"google.golang.org/protobuf/types/known/anypb"
)

// newTransfer builds a minimal unsigned TRX transfer from the key's address to itself.
func newTransfer(key pqc.PrivateKey) *core.Transaction {
	addr := key.Public().Address()
	param, err := anypb.New(&core.TransferContract{OwnerAddress: addr, ToAddress: addr, Amount: 1_000_000})
	if err != nil {
		log.Fatal(err)
	}
	return &core.Transaction{RawData: &core.TransactionRaw{
		RefBlockBytes: []byte{0x01, 0x02},
		RefBlockHash:  []byte{0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a},
		Expiration:    1_700_000_060_000,
		Timestamp:     1_700_000_000_000,
		Contract: []*core.Transaction_Contract{{
			Type:      core.Transaction_Contract_TransferContract,
			Parameter: param,
		}},
	}}
}

// Sign a transaction offline with a post-quantum key. The signature goes in
// pq_auth_sig; the ECDSA signature list stays empty.
func ExampleNewPQSigner() {
	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	defer key.Destroy()

	s, err := signer.NewPQSigner(key)
	if err != nil {
		log.Fatal(err)
	}

	tx, err := s.Sign(newTransfer(key))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("ECDSA signatures:", len(tx.GetSignature()))
	fmt.Println("pq_auth_sig entries:", len(tx.GetPqAuthSig()))
	fmt.Println("scheme:", tx.GetPqAuthSig()[0].GetScheme())
	fmt.Println("signature bytes:", len(tx.GetPqAuthSig()[0].GetSignature()))

	// Output:
	// ECDSA signatures: 0
	// pq_auth_sig entries: 1
	// scheme: ML_DSA_44
	// signature bytes: 2420
}

// A transaction carries at most keystore.MaxTotalSignatures signatures, ECDSA
// and post-quantum together. Signing past that fails with ErrTooManySignatures.
func ExampleSignTransactionPQ_signatureLimit() {
	first, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	defer first.Destroy()

	tx := newTransfer(first)
	for i := 0; i < keystore.MaxTotalSignatures; i++ {
		key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := signer.SignTransactionPQ(tx, key); err != nil {
			log.Fatal(err)
		}
		key.Destroy()
	}

	// CheckSignatureRoom asks without signing.
	fmt.Println(errors.Is(keystore.CheckSignatureRoom(tx), keystore.ErrTooManySignatures))

	_, err = signer.SignTransactionPQ(tx, first)
	fmt.Println(errors.Is(err, signer.ErrTooManySignatures))

	// Output:
	// true
	// true
}
