package transaction_test

import (
	"crypto/rand"
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"google.golang.org/protobuf/types/known/anypb"
)

// Hand a PQ-signed transaction to another tool as JSON and read it back. The
// pq_auth_sig entries survive the round trip.
func ExampleToJSON_pqAuthSig() {
	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	defer key.Destroy()

	addr := key.Public().Address()
	param, err := anypb.New(&core.TransferContract{OwnerAddress: addr, ToAddress: addr, Amount: 1_000_000})
	if err != nil {
		log.Fatal(err)
	}
	tx := &core.Transaction{RawData: &core.TransactionRaw{
		RefBlockBytes: []byte{0x01, 0x02},
		RefBlockHash:  []byte{0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a},
		Expiration:    1_700_000_060_000,
		Timestamp:     1_700_000_000_000,
		Contract: []*core.Transaction_Contract{{
			Type:      core.Transaction_Contract_TransferContract,
			Parameter: param,
		}},
	}}
	if _, err := signer.SignTransactionPQ(tx, key); err != nil {
		log.Fatal(err)
	}

	data, err := transaction.ToJSON(tx)
	if err != nil {
		log.Fatal(err)
	}
	back, err := transaction.FromJSON(data)
	if err != nil {
		log.Fatal(err)
	}

	got := back.GetPqAuthSig()
	fmt.Println("pq_auth_sig entries:", len(got))
	fmt.Println("scheme:", got[0].GetScheme())
	fmt.Println("same signature:", string(got[0].GetSignature()) == string(tx.GetPqAuthSig()[0].GetSignature()))

	// Output:
	// pq_auth_sig entries: 1
	// scheme: ML_DSA_44
	// same signature: true
}
