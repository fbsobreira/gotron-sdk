// Minimal example showing how to rent and return energy via JustLend DAO's
// energy rental contract using the GoTRON SDK.
//
// The contract exposes two methods:
//
//	rentResource(address receiver, uint256 amount, uint32 resourceType)
//	returnResource(address receiver, uint256 amount, uint32 resourceType)
//
// Resource types: 0 = Bandwidth, 1 = Energy.
//
// For more detailed examples see https://github.com/fbsobreira/gotron-examples
//
// Usage:
//
//	go run ./examples/justlend                            # dry-run (simulate only)
//	JUSTLEND_KEY=<hex> go run ./examples/justlend         # live mode (key from env)
//	go run ./examples/justlend -key <private-key-hex>     # live mode (key from flag, unsafe: visible in shell history)
//	go run ./examples/justlend -contract <address>        # override contract address
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

const (
	mainnetEndpoint = "grpc.trongrid.io:50051"

	// JustLend energy rental contract on mainnet.
	defaultContract = "TU2MJ5Veik1LRAgjeSzEdvmDYx7mefJZvd"

	// A known mainnet address used as the receiver in simulations.
	defaultReceiver = "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"
)

// Resource type constants as defined by the JustLend contract.
const (
	ResourceBandwidth = 0
	ResourceEnergy    = 1
)

var (
	privKeyHex      string
	contractAddress string
	receiverAddress string
	dryRun          bool
)

func main() {
	flag.StringVar(&privKeyHex, "key", "", "private key hex (prefer JUSTLEND_KEY env var)")
	flag.StringVar(&contractAddress, "contract", defaultContract, "JustLend energy rental contract address")
	flag.StringVar(&receiverAddress, "receiver", defaultReceiver, "receiver address for rent/return")
	flag.Parse()

	if envKey := os.Getenv("JUSTLEND_KEY"); envKey != "" {
		privKeyHex = envKey
	}

	dryRun = privKeyHex == ""

	if dryRun {
		fmt.Println("Mode: DRY-RUN (simulate only, no broadcast)")
		fmt.Println("  Pass -key <hex> or set JUSTLEND_KEY to execute transactions")
	} else {
		fmt.Println("Mode: LIVE (broadcasting to mainnet)")
	}
	fmt.Println()

	c := client.NewGrpcClient(mainnetEndpoint)
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	// 1. Simulate rentResource (read-only)
	exampleSimulateRent(c)

	// 2. Simulate returnResource (read-only)
	exampleSimulateReturn(c)

	// 3. Execute rentResource (state-changing, requires key)
	if !dryRun {
		exampleExecuteRent(c)
	}

	// 4. Execute returnResource (state-changing, requires key)
	if !dryRun {
		exampleExecuteReturn(c)
	}

	fmt.Println("\nAll JustLend examples completed!")
}

// exampleSimulateRent calls rentResource via TriggerConstantContract to
// simulate renting energy without broadcasting a transaction.
func exampleSimulateRent(c *client.GrpcClient) {
	fmt.Println("=== Simulate rentResource ===")

	method := "rentResource(address,uint256,uint32)"
	params := fmt.Sprintf(`[{"address": "%s"}, {"uint256": "1000000"}, {"uint32": "%d"}]`,
		receiverAddress, ResourceEnergy)

	tx, err := c.TriggerConstantContract("", contractAddress, method, params)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	printConstantResult(tx, "rentResource")
	fmt.Println("  OK")
}

// exampleSimulateReturn calls returnResource via TriggerConstantContract to
// simulate returning rented energy without broadcasting a transaction.
func exampleSimulateReturn(c *client.GrpcClient) {
	fmt.Println("=== Simulate returnResource ===")

	method := "returnResource(address,uint256,uint32)"
	params := fmt.Sprintf(`[{"address": "%s"}, {"uint256": "1000000"}, {"uint32": "%d"}]`,
		receiverAddress, ResourceEnergy)

	tx, err := c.TriggerConstantContract("", contractAddress, method, params)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	printConstantResult(tx, "returnResource")
	fmt.Println("  OK")
}

// exampleExecuteRent executes rentResource via TriggerContract, signs it,
// and broadcasts the transaction.
func exampleExecuteRent(c *client.GrpcClient) {
	fmt.Println("=== Execute rentResource ===")

	signerKey, err := keys.GetPrivateKeyFromHex(privKeyHex)
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}
	signerAddr := address.BTCECPubkeyToAddress(signerKey.PubKey()).String()

	method := "rentResource(address,uint256,uint32)"
	params := fmt.Sprintf(`[{"address": "%s"}, {"uint256": "1000000"}, {"uint32": "%d"}]`,
		receiverAddress, ResourceEnergy)

	tx, err := c.TriggerContract(
		signerAddr,
		contractAddress,
		method,
		params,
		50_000_000, // fee limit: 50 TRX
		0, "", 0,
	)
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := transaction.SignTransaction(tx.Transaction, signerKey)
	if err != nil {
		log.Fatal(err)
	}

	result, err := c.Broadcast(signedTx)
	if err != nil {
		log.Fatal(err)
	}
	if !result.Result || result.Code != api.Return_SUCCESS {
		log.Fatalf("broadcast failed: (%d) %s", result.Code, result.Message)
	}

	fmt.Printf("  TX: %x\n", tx.Txid)
	fmt.Println("  OK")
}

// exampleExecuteReturn executes returnResource via TriggerContract, signs it,
// and broadcasts the transaction.
func exampleExecuteReturn(c *client.GrpcClient) {
	fmt.Println("=== Execute returnResource ===")

	signerKey, err := keys.GetPrivateKeyFromHex(privKeyHex)
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}
	signerAddr := address.BTCECPubkeyToAddress(signerKey.PubKey()).String()

	method := "returnResource(address,uint256,uint32)"
	params := fmt.Sprintf(`[{"address": "%s"}, {"uint256": "1000000"}, {"uint32": "%d"}]`,
		receiverAddress, ResourceEnergy)

	tx, err := c.TriggerContract(
		signerAddr,
		contractAddress,
		method,
		params,
		50_000_000, // fee limit: 50 TRX
		0, "", 0,
	)
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := transaction.SignTransaction(tx.Transaction, signerKey)
	if err != nil {
		log.Fatal(err)
	}

	result, err := c.Broadcast(signedTx)
	if err != nil {
		log.Fatal(err)
	}
	if !result.Result || result.Code != api.Return_SUCCESS {
		log.Fatalf("broadcast failed: (%d) %s", result.Code, result.Message)
	}

	fmt.Printf("  TX: %x\n", tx.Txid)
	fmt.Println("  OK")
}

// printConstantResult decodes and prints the result from a TriggerConstantContract call.
func printConstantResult(tx *api.TransactionExtention, method string) {
	if tx.GetResult().GetCode() != 0 {
		if results := tx.GetConstantResult(); len(results) > 0 {
			msg, _ := abi.DecodeRevertReason(results[0])
			fmt.Printf("  Reverted: %s\n", msg)
		} else {
			fmt.Printf("  Reverted: (no revert data)\n")
		}
		return
	}

	results := tx.GetConstantResult()
	if len(results) == 0 {
		fmt.Println("  No result returned")
		return
	}

	fmt.Printf("  %s result: %s\n", method, hex.EncodeToString(results[0]))

	// Try to interpret as uint256 (common return for these methods).
	if len(results[0]) >= 32 {
		v := new(big.Int).SetBytes(results[0][:32])
		fmt.Printf("  Decoded (uint256): %s\n", v.String())
	}
}
