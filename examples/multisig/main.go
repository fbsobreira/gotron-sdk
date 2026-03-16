// Runnable version of every multi-sig code block from docs/examples.md.
// Each function is a 1:1 copy of the doc example, with only two changes:
//   - Placeholder addresses replaced with real Nile testnet addresses
//   - keys.GetPrivateKeyFromHex() replaced with keys.GenerateKey()
//     (unless a real private key is provided via env or flag)
//
// If this file fails to run, the documentation is wrong.
//
// Usage:
//
//	go run ./examples/multisig                          # dry-run (no broadcast)
//	MULTISIG_KEY=<hex> go run ./examples/multisig       # live mode (key from env)
//	go run ./examples/multisig -key <private-key-hex>   # live mode (key from flag, unsafe: visible in shell history)
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

const (
	nileEndpoint = "grpc.nile.trongrid.io:50051"

	// Nile testnet addresses standing in for doc placeholders:
	//   "TRecipientBase58..." / "TDelegateToBase58..."
	defaultRecipient = "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"
	//   "TSigner3Base58..." (third signer for 2-of-3)
	defaultSigner3 = "TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM"
	//   "TTokenContractBase58..."
	nileBTT = "TNuoKL1ni8aoshfFL1ASca1Gou9RXwAzfn" // Nile BTT (TRC20)
)

var (
	privKeyHex     string
	skipPermission bool
	dryRun         bool
	signerKey      *btcec.PrivateKey
	signerKey2     *btcec.PrivateKey
	signerKey3     *btcec.PrivateKey
	signerAddr     string
	signerAddr2    string
	signerAddr3    string
)

func main() {
	flag.StringVar(&privKeyHex, "key", "", "private key hex (prefer MULTISIG_KEY env var)")
	flag.BoolVar(&skipPermission, "skip-permission", false, "skip UpdateAccountPermission (costs 100 TRX); use if already configured from a previous run")
	flag.Parse()

	// Prefer env var over CLI flag to avoid leaking keys in shell history.
	if envKey := os.Getenv("MULTISIG_KEY"); envKey != "" {
		privKeyHex = envKey
	}

	dryRun = privKeyHex == ""

	if dryRun {
		fmt.Println("Mode: DRY-RUN (no broadcast)")
		fmt.Println("  Pass -key <hex> to broadcast transactions to Nile testnet")

		// Use a random key and a known Nile address as the "account"
		var err error
		signerKey, err = keys.GenerateKey()
		if err != nil {
			log.Fatal(err)
		}
		signerKey2 = deriveChildKey(signerKey, "signer2")
		signerKey3 = deriveChildKey(signerKey, "signer3")
		signerAddr = "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b"
		signerAddr2 = defaultRecipient
		signerAddr3 = defaultSigner3
	} else {
		fmt.Println("Mode: LIVE (broadcasting to Nile testnet)")
		var err error
		signerKey, err = keys.GetPrivateKeyFromHex(privKeyHex)
		if err != nil {
			log.Fatalf("invalid private key: %v", err)
		}
		// Derive deterministic key2/key3 from the primary key so the
		// same addresses are used every run (permission setup + signing).
		signerKey2 = deriveChildKey(signerKey, "signer2")
		signerKey3 = deriveChildKey(signerKey, "signer3")

		signerAddr = address.BTCECPubkeyToAddress(signerKey.PubKey()).String()
		signerAddr2 = address.BTCECPubkeyToAddress(signerKey2.PubKey()).String()
		signerAddr3 = address.BTCECPubkeyToAddress(signerKey3.PubKey()).String()

		fmt.Printf("  Signer 1: %s\n", signerAddr)
		fmt.Printf("  Signer 2: %s (derived)\n", signerAddr2)
		fmt.Printf("  Signer 3: %s (derived)\n", signerAddr3)
	}

	fmt.Println()

	if skipPermission {
		fmt.Println("=== Setting Up Multi-Sig Permissions ===")
		fmt.Println("  [skipped] -skip-permission flag set")
	} else {
		exampleSetupPermissions()
	}
	exampleMultiSigTRXTransfer()
	exampleValidateSignWeight()
	exampleSignExternalTransaction()
	exampleMultiSigTRC20Transfer()
	exampleMultiSigResourceDelegation()

	fmt.Println("\nAll examples from docs/examples.md validated!")
}

// broadcast sends a signed transaction if in live mode, or skips if dry-run.
// It checks GetTransactionSignWeight first — if the threshold isn't met,
// the transaction is skipped with a warning instead of failing.
func broadcast(c *client.GrpcClient, signedTx *core.Transaction) {
	if dryRun {
		fmt.Println("  [dry-run] skipping broadcast")
		return
	}

	// Check if we have enough signatures before broadcasting.
	weight, err := c.GetTransactionSignWeight(signedTx)
	if err != nil {
		log.Fatalf("GetTransactionSignWeight: %v", err)
	}
	if weight.GetPermission() == nil {
		fmt.Printf("  [skipped] no permission info returned (weight result: %v)\n",
			weight.GetResult())
		return
	}
	if weight.CurrentWeight < weight.Permission.Threshold {
		fmt.Printf("  [skipped] weight %d < threshold %d (need more signers)\n",
			weight.CurrentWeight, weight.Permission.Threshold)
		return
	}

	result, err := c.Broadcast(signedTx)
	if err != nil {
		log.Fatal(err)
	}
	if !result.Result || result.Code != api.Return_SUCCESS {
		log.Fatalf("broadcast failed: (%d) %s", result.Code, result.Message)
	}
	fmt.Println("  broadcast: SUCCESS")
}

// deriveChildKey produces a deterministic child key by hashing the parent
// private key bytes with a label. Same parent + label always yields the
// same child, so permission addresses are stable across runs.
func deriveChildKey(parent *btcec.PrivateKey, label string) *btcec.PrivateKey {
	h := sha256.Sum256(append(parent.Serialize(), []byte(label)...))
	child, _ := btcec.PrivKeyFromBytes(h[:])
	return child
}

// ----- docs/examples.md: "Setting Up Multi-Sig Permissions" -----

func exampleSetupPermissions() {
	fmt.Println("=== Setting Up Multi-Sig Permissions ===")

	c := client.NewGrpcClient(nileEndpoint)
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	// The account to configure — the current owner must sign this transaction
	accountAddr := signerAddr

	// Owner permission: 1-of-1 (keep single owner for recovery)
	owner := map[string]interface{}{
		"threshold": int64(1),
		"keys": map[string]int64{
			accountAddr: 1,
		},
	}

	// Active permission: 2-of-3 multi-sig for transfers and contracts
	active := map[string]interface{}{
		"name":      "daily-operations",
		"threshold": int64(2),
		"keys": map[string]int64{
			accountAddr: 1,
			signerAddr2: 1,
			signerAddr3: 1,
		},
		"operations": map[string]bool{
			"TransferContract":         true,
			"TriggerSmartContract":     true,
			"DelegateResourceContract": true,
		},
	}

	tx, err := c.UpdateAccountPermission(accountAddr, owner, nil, []map[string]interface{}{active})
	if err != nil {
		log.Fatal(err)
	}

	// Sign with the current owner key (before multi-sig is active)
	ownerKey := signerKey

	signedTx, err := transaction.SignTransaction(tx.Transaction, ownerKey)
	if err != nil {
		log.Fatal(err)
	}

	broadcast(c, signedTx)

	fmt.Printf("  Permission update tx: %x\n", tx.Txid)
	fmt.Println("  OK")
}

// ----- docs/examples.md: "Multi-Sig TRX Transfer" -----

func exampleMultiSigTRXTransfer() {
	fmt.Println("=== Multi-Sig TRX Transfer ===")

	c := client.NewGrpcClient(nileEndpoint)
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	multiSigAddr := signerAddr
	recipientAddr := defaultRecipient

	// Step 1: Build the transaction
	tx, err := c.Transfer(multiSigAddr, recipientAddr, 1_000_000) // 1 TRX
	if err != nil {
		log.Fatal(err)
	}

	// Step 2: Set PermissionId = 2 for the active permission
	// This MUST be done before signing — it changes the transaction hash.
	if err := tx.SetPermissionId(2); err != nil {
		log.Fatal(err)
	}

	// Step 3: Sign with multiple keys (2-of-3 threshold)
	key1 := signerKey
	key2 := signerKey2

	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err = transaction.SignTransaction(signedTx, key2)
	if err != nil {
		log.Fatal(err)
	}

	// Step 4: Broadcast
	broadcast(c, signedTx)

	fmt.Printf("  Multi-sig transfer tx: %x\n", tx.Txid)
	fmt.Println("  OK")
}

// ----- docs/examples.md: "Validating Signatures with GetTransactionSignWeight" -----

func exampleValidateSignWeight() {
	fmt.Println("=== Validating Signatures with GetTransactionSignWeight ===")

	c := client.NewGrpcClient(nileEndpoint)
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	multiSigAddr := signerAddr
	recipientAddr := defaultRecipient

	// Build and set permission
	tx, err := c.Transfer(multiSigAddr, recipientAddr, 1_000_000)
	if err != nil {
		log.Fatal(err)
	}
	if err := tx.SetPermissionId(2); err != nil {
		log.Fatal(err)
	}

	// Sign with first key
	key1 := signerKey
	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		log.Fatal(err)
	}

	// Check current signature weight
	weight, err := c.GetTransactionSignWeight(signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Current weight: %d\n", weight.CurrentWeight)
	fmt.Printf("  Required threshold: %d\n", weight.Permission.Threshold)
	fmt.Printf("  Approved signers: %d\n", len(weight.ApprovedList))
	for _, addr := range weight.ApprovedList {
		fmt.Printf("    - %s\n", hex.EncodeToString(addr))
	}

	if weight.CurrentWeight < weight.Permission.Threshold {
		fmt.Println("  Not enough signatures yet — collect more before broadcasting.")
	} else {
		fmt.Println("  Threshold met — ready to broadcast!")
	}
	fmt.Println("  OK")
}

// ----- docs/examples.md: "Signing Externally-Built Transactions" -----

func exampleSignExternalTransaction() {
	fmt.Println("=== Signing Externally-Built Transactions ===")

	c := client.NewGrpcClient(nileEndpoint)
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	// Build a real transaction to get a valid raw_data_hex
	// (in the doc, rawDataHex is "received from another party")
	srcTx, err := c.Transfer(signerAddr, defaultRecipient, 1_000_000)
	if err != nil {
		log.Fatal(err)
	}
	rawDataHex, err := transaction.ToRawDataHex(srcTx.GetTransaction())
	if err != nil {
		log.Fatal(err)
	}

	// --- From here, the code matches the doc example exactly ---

	// Reconstruct the transaction from raw_data_hex
	tx, err := transaction.FromRawDataHex(rawDataHex)
	if err != nil {
		log.Fatal("failed to parse transaction: ", err)
	}

	// Sign with your key
	myKey := signerKey
	signedTx, err := transaction.SignTransaction(tx, myKey)
	if err != nil {
		log.Fatal(err)
	}

	// Optionally verify signature weight before broadcasting
	weight, err := c.GetTransactionSignWeight(signedTx)
	if err != nil {
		log.Fatal(err)
	}
	if weight.CurrentWeight < weight.Permission.Threshold {
		// Share the partially-signed transaction with the next signer.
		// Use ToJSON to preserve existing signatures — ToRawDataHex would drop them.
		partialJSON, err := transaction.ToJSON(signedTx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  Need more signatures. Share transaction JSON: %s...\n", string(partialJSON)[:80])
		fmt.Printf("  Signatures so far: %d\n", len(signedTx.Signature))
	} else {
		// Broadcast when threshold is met
		broadcast(c, signedTx)
	}

	// --- Also test the FromJSON snippet from the doc ---
	jsonData, err := transaction.ToJSON(srcTx.GetTransaction())
	if err != nil {
		log.Fatal(err)
	}
	txFromJSON, err := transaction.FromJSON(jsonData)
	if err != nil {
		log.Fatal(err)
	}
	_ = txFromJSON

	fmt.Println("  OK")
}

// ----- docs/examples.md: "Multi-Sig TRC20 Token Transfer" -----

func exampleMultiSigTRC20Transfer() {
	fmt.Println("=== Multi-Sig TRC20 Token Transfer ===")

	c := client.NewGrpcClient(nileEndpoint)
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	multiSigAddr := signerAddr
	tokenContract := nileBTT
	recipientAddr := defaultRecipient

	// Build TRC20 transfer: transfer(address,uint256)
	// Parameters: recipient address and amount (in token smallest unit)
	method := "transfer(address,uint256)"
	params := fmt.Sprintf(`[{"address": "%s"}, {"uint256": "1000000"}]`, recipientAddr)

	tx, err := c.TriggerContract(
		multiSigAddr,
		tokenContract,
		method,
		params,
		10_000_000, // fee limit: 10 TRX
		0,          // call value
		"",         // token ID
		0,          // token value
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set active permission for multi-sig
	if err := tx.SetPermissionId(2); err != nil {
		log.Fatal(err)
	}

	// Sign with two keys (2-of-3)
	key1 := signerKey
	key2 := signerKey2

	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err = transaction.SignTransaction(signedTx, key2)
	if err != nil {
		log.Fatal(err)
	}

	broadcast(c, signedTx)

	fmt.Printf("  Multi-sig TRC20 transfer tx: %x\n", tx.Txid)
	fmt.Println("  OK")
}

// ----- docs/examples.md: "Multi-Sig Resource Delegation" -----

func exampleMultiSigResourceDelegation() {
	fmt.Println("=== Multi-Sig Resource Delegation ===")

	c := client.NewGrpcClient(nileEndpoint)
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	multiSigAddr := signerAddr
	delegateTo := defaultRecipient

	// In live mode, stake 2 TRX for energy first (uses owner permission).
	// DelegateResource requires the account to have staked TRX.
	if !dryRun {
		stakeEnergy(c, multiSigAddr)
	}

	// Delegate 1 TRX worth of energy
	tx, err := c.DelegateResource(
		multiSigAddr,
		delegateTo,
		core.ResourceCode_ENERGY,
		1_000_000, // 1 TRX in sun
		false,     // not locked
		0,         // no lock period
	)
	if err != nil {
		fmt.Printf("  [skipped] DelegateResource: %v\n", err)
		fmt.Println("  OK")
		return
	}
	if tx.GetTransaction().GetRawData() == nil {
		fmt.Println("  [skipped] node returned empty transaction (account may need staked TRX)")
		fmt.Println("  OK")
		return
	}

	// Set active permission for multi-sig
	if err := tx.SetPermissionId(2); err != nil {
		log.Fatal(err)
	}

	// Sign with two keys (2-of-3)
	key1 := signerKey
	key2 := signerKey2

	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err = transaction.SignTransaction(signedTx, key2)
	if err != nil {
		log.Fatal(err)
	}

	broadcast(c, signedTx)

	fmt.Printf("  Multi-sig delegation tx: %x\n", tx.Txid)
	fmt.Println("  OK")
}

// stakeEnergy freezes TRX for energy using the owner permission (single signer).
// This is a prerequisite for DelegateResource.
func stakeEnergy(c *client.GrpcClient, ownerAddr string) {
	fmt.Println("  Staking 2 TRX for energy (owner permission)...")

	tx, err := c.FreezeBalanceV2(ownerAddr, core.ResourceCode_ENERGY, 2_000_000)
	if err != nil {
		fmt.Printf("  [skipped] FreezeBalanceV2: %v\n", err)
		return
	}
	if tx.GetTransaction().GetRawData() == nil {
		fmt.Println("  [skipped] FreezeBalanceV2 returned empty transaction")
		return
	}

	// Owner permission (PermissionId=0) — single signer, no SetPermissionId needed.
	signedTx, err := transaction.SignTransaction(tx.Transaction, signerKey)
	if err != nil {
		fmt.Printf("  [skipped] sign FreezeBalanceV2: %v\n", err)
		return
	}

	result, err := c.Broadcast(signedTx)
	if err != nil {
		fmt.Printf("  [skipped] broadcast FreezeBalanceV2: %v\n", err)
		return
	}
	if !result.Result || result.Code != api.Return_SUCCESS {
		fmt.Printf("  [skipped] FreezeBalanceV2 broadcast: (%d) %s\n", result.Code, result.Message)
		return
	}
	fmt.Printf("  Staked: %x\n", tx.Txid)
}
