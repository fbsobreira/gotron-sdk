package ledger

import (
	"fmt"
	"log"
	"os"
	"sync"

	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	nanos *NanoS //singleton
	once  sync.Once
)

func getLedger() *NanoS {
	once.Do(func() {
		var err error
		nanos, err = OpenNanoS()
		if err != nil {
			log.Fatalln("Couldn't open device:", err)
			os.Exit(-1)
		}
	})

	return nanos
}

// GetAddress ProcessAddressCommand list the address associated with Ledger Nano S
func GetAddress() string {
	n := getLedger()
	addr, err := n.GetAddress()
	if err != nil {
		log.Fatalln("Couldn't get address:", err)
		os.Exit(-1)
	}

	return addr
}

//ProcessAddressCommand list the address associated with Ledger Nano S
func ProcessAddressCommand() {
	n := getLedger()
	addr, err := n.GetAddress()
	if err != nil {
		log.Fatalln("Couldn't get address:", err)
		os.Exit(-1)
	}

	fmt.Printf("%-24s\t\t%23s\n", "NAME", "ADDRESS")
	fmt.Printf("%-48s\t%s\n", "Ledger Nano S", addr)
}

// SignTx signs the given transaction with the requested account.
func SignTx(tx []byte) ([]byte, error) {

	n := getLedger()
	sig, err := n.SignTxn(tx)
	if err != nil {
		log.Println("Couldn't sign transaction, error:", err)
		return nil, err
	}

	var hashBytes [32]byte
	hw := sha3.NewLegacyKeccak256()
	hw.Write(tx[:])
	hw.Sum(hashBytes[:0])

	pubkey, err := crypto.Ecrecover(hashBytes[:], sig[:])
	if err != nil {
		log.Println("Ecrecover failed :", err)
		return nil, err
	}

	if len(pubkey) == 0 || pubkey[0] != 4 {
		log.Println("invalid public key")
		return nil, err
	}

	//pubBytes := crypto.Keccak256(pubkey[1:65])[12:]
	//signerAddr, _ := address.PubkeyToAddress(pubBytes)

	// TODO:
	//return sig, nil
	return nil, nil
}
