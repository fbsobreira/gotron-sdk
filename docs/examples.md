# Examples

This document provides practical examples for common use cases with the GoTRON SDK.

## Table of Contents

- [Examples](#examples)
  - [Table of Contents](#table-of-contents)
  - [Basic Operations](#basic-operations)
    - [Create and Fund Account](#create-and-fund-account)
    - [Monitor Account Balance](#monitor-account-balance)
  - [Token Operations](#token-operations)
    - [TRC20 Token Wrapper](#trc20-token-wrapper)
  - [Multi-Signature Workflows](#multi-signature-workflows)
    - [Understanding Permissions](#understanding-permissions)
    - [Setting Up Multi-Sig Permissions](#setting-up-multi-sig-permissions)
    - [Multi-Sig TRX Transfer](#multi-sig-trx-transfer)
    - [Validating Signatures with GetTransactionSignWeight](#validating-signatures-with-gettransactionsignweight)
    - [Signing Externally-Built Transactions](#signing-externally-built-transactions)
    - [Multi-Sig TRC20 Token Transfer](#multi-sig-trc20-token-transfer)
    - [Multi-Sig Resource Delegation](#multi-sig-resource-delegation)
    - [Common Pitfalls](#common-pitfalls)
  - [Advanced Use Cases](#advanced-use-cases)
    - [Resource Manager](#resource-manager)
  - [Best Practices Summary](#best-practices-summary)

## Basic Operations

### Create and Fund Account

```go
package main

import (
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

func main() {
	// Connect to network
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := c.Start(client.GRPCInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	// Create new account
	privateKey, err := keys.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	addr := address.BTCECPrivkeyToAddress(privateKey)
	fmt.Printf("New address: %s\n", address.Address(addr).String())
	fmt.Printf("Private key: %x\n", privateKey.Serialize())

	// Fund account (requires funded account)
	funderKey, _ := keys.GetPrivateKeyFromHex("your_private_key_here")
	funderAddr := address.BTCECPrivkeyToAddress(funderKey)

	fmt.Println("Funding account...", funderAddr.String())

	// Send 100 TRX
	tx, err := c.Transfer(funderAddr.String(), addr.String(), 100_000_000) // 100 TRX = 100 * 1e6 sun
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := transaction.SignTransaction(tx.Transaction, funderKey)
	if err != nil {
		log.Fatal(err)
	}

	result, err := c.Broadcast(signedTx)
	if err != nil {
		log.Fatal(err)
	}

	if !result.Result ||
		result.Code != api.Return_SUCCESS {
		log.Fatalf("Broadcast failed: (%d) %s", result.Code, result.Message)
	}

	fmt.Printf("Transaction ID: %x\n", tx.Txid)
}

```

### Monitor Account Balance

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
)

func monitorBalance(c *client.GrpcClient, addr string, interval time.Duration) {
	tronAddr, err := address.Base58ToAddress(addr)
	if err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("Monitoring balance for %s\n", addr)

	for range ticker.C {
		account, err := c.GetAccount(tronAddr.String())
		if err != nil {
			log.Printf("Error getting account: %v", err)
			continue
		}

		balance := float64(account.Balance) / 1e6
		fmt.Printf("[%s] Balance: %.6f TRX\n",
			time.Now().Format("15:04:05"), balance)
	}
}

func main() {
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := c.Start(client.GRPCInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	// Monitor balance every 30 seconds
	monitorBalance(c, "TX8h6Df74VpJsXF6sTDz1QJsq3Ec8dABc3", 10*time.Second)
}
```

## Token Operations

### TRC20 Token Wrapper

```go
package main

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

// Hex2Bytes converts a hex string to a byte slice unchecked.
func Hex2Bytes(value string) []byte {
	bytes, _ := hex.DecodeString(value)
	return bytes
}

type TRC20Token struct {
	client   *client.GrpcClient
	contract address.Address
	decimals uint8
}

func NewTRC20Token(client *client.GrpcClient, contractAddr string) (*TRC20Token, error) {
	addr, err := address.Base58ToAddress(contractAddr)
	if err != nil {
		return nil, err
	}

	token := &TRC20Token{
		client:   client,
		contract: addr,
	}

	// Get decimals
	decimals, err := token.Decimals()
	if err != nil {
		return nil, err
	}
	token.decimals = decimals

	return token, nil
}

func (t *TRC20Token) Name() (string, error) {
	method := "name()"

	result, err := t.client.TriggerConstantContract("", t.contract.String(), method, "")
	if err != nil {
		return "", err
	}

	// Parse string from result
	return parseString(result.ConstantResult[0]), nil
}

func (t *TRC20Token) Symbol() (string, error) {
	method := "symbol()"

	result, err := t.client.TriggerConstantContract("", t.contract.String(), method, "")
	if err != nil {
		return "", err
	}

	return parseString(result.ConstantResult[0]), nil
}

func (t *TRC20Token) Decimals() (uint8, error) {
	method := "decimals()"

	result, err := t.client.TriggerConstantContract("", t.contract.String(), method, "")
	if err != nil {
		return 0, err
	}

	decimals := new(big.Int).SetBytes(result.ConstantResult[0])
	return uint8(decimals.Uint64()), nil
}

func (t *TRC20Token) BalanceOf(account string) (*big.Int, error) {
	addr, err := address.Base58ToAddress(account)
	if err != nil {
		return nil, err
	}

	method := "balanceOf(address)"
	data := fmt.Sprintf(`[{"address": "%s"}]`, addr.String())

	result, err := t.client.TriggerConstantContract("", t.contract.String(), method, data)
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(result.ConstantResult[0]), nil
}

func (t *TRC20Token) Transfer(from string, to string, amount *big.Int, privateKey *btcec.PrivateKey) (string, error) {
	fromAddr, _ := address.Base58ToAddress(from)
	toAddr, _ := address.Base58ToAddress(to)

	method := "transfer(address,uint256)"
	data := fmt.Sprintf(`[{"address": "%s"}, {"uint256": "%d"}]`, toAddr.String(), amount)

	tx, err := t.client.TriggerContract(fromAddr.String(), t.contract.String(), method, data, 10000000, 0, "", 0)
	if err != nil {
		return "", err
	}

	signedTx, err := transaction.SignTransaction(tx.Transaction, privateKey)
	if err != nil {
		return "", err
	}

	result, err := t.client.Broadcast(signedTx)
	if err != nil {
		return "", err
	}

	if !result.Result ||
		result.Code != api.Return_SUCCESS {
		return "", fmt.Errorf("broadcast error: %s", result.Message)
	}

	return fmt.Sprintf("%x", tx.Txid), nil
}

// Helper function to parse string from bytes
func parseString(data []byte) string {
	if len(data) < 64 {
		return ""
	}

	// Skip offset and length
	data = data[64:]

	// Find actual string length
	length := 0
	for i, b := range data {
		if b == 0 {
			length = i
			break
		}
	}

	return string(data[:length])
}

func main() {
	exampleContract := "TVj7RNVHy6thbM7BWdSe9G6gXwKhjhdNZS"

	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := c.Start(client.GRPCInsecure())
	if err != nil {
		panic(err)
	}
	defer c.Stop()

	klvTRC20, err := NewTRC20Token(c, exampleContract)
	if err != nil {
		panic(err)
	}

	name, err := klvTRC20.Name()
	if err != nil {
		panic(err)
	}
	fmt.Println("Token Name:", name)

	symbol, err := klvTRC20.Symbol()
	if err != nil {
		panic(err)
	}
	fmt.Println("Token Symbol:", symbol)
	fmt.Println("Token Decimals:", klvTRC20.decimals)
}
```

## Multi-Signature Workflows

Multi-signature (multi-sig) accounts require multiple parties to approve a transaction
before it can be broadcast. The TRON network supports this through its permission system,
where each account can have owner, witness, and active permissions with configurable
thresholds and key weights.

### Understanding Permissions

TRON accounts have three permission tiers, each identified by a `PermissionId`:

| PermissionId | Type | Purpose |
|---|---|---|
| 0 | Owner | Full account control: transfer ownership, modify permissions |
| 1 | Witness | Super representative operations (block production) |
| 2+ | Active | Custom permissions for specific operations (transfers, contracts, etc.) |

Each permission has:
- **Threshold**: minimum total weight required to authorize a transaction
- **Keys**: a set of addresses, each with an assigned weight
- **Operations** (active only): a bitmask of allowed contract types

For example, a 2-of-3 multi-sig active permission has threshold=2 and three keys each with weight=1.

### Setting Up Multi-Sig Permissions

Use `UpdateAccountPermission` to convert a regular account into a multi-sig account.
The owner who currently controls the account must sign this transaction.

```go
package main

import (
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

func main() {
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	// The account to configure — the current owner must sign this transaction
	accountAddr := "TAccountToConfigureBase58..."

	// Owner permission: 1-of-1 (keep single owner for recovery)
	owner := map[string]interface{}{
		"threshold": int64(1),
		"keys": map[string]int64{
			"TOwnerAddressBase58...": 1,
		},
	}

	// Active permission: 2-of-3 multi-sig for transfers and contracts
	active := map[string]interface{}{
		"name":      "daily-operations",
		"threshold": int64(2),
		"keys": map[string]int64{
			"TSigner1Base58...": 1,
			"TSigner2Base58...": 1,
			"TSigner3Base58...": 1,
		},
		"operations": map[string]bool{
			"TransferContract":        true,
			"TriggerSmartContract":    true,
			"DelegateResourceContract": true,
		},
	}

	tx, err := c.UpdateAccountPermission(accountAddr, owner, nil, []map[string]interface{}{active})
	if err != nil {
		log.Fatal(err)
	}

	// Sign with the current owner key (before multi-sig is active)
	ownerKey, err := keys.GetPrivateKeyFromHex("owner-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := transaction.SignTransaction(tx.Transaction, ownerKey)
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

	fmt.Printf("Permission update tx: %x\n", tx.Txid)
	fmt.Println("Account is now multi-sig!")
}
```

> **Important:** After updating permissions, future transactions using the active permission
> must set `PermissionId = 2` and collect enough signatures to meet the threshold.
> Keep the owner permission secure — it can override all other permissions.

### Multi-Sig TRX Transfer

Build a TRX transfer from a multi-sig account, sign with multiple keys, and broadcast.

```go
package main

import (
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

func main() {
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	multiSigAddr := "TMultiSigAccountBase58..."
	recipientAddr := "TRecipientBase58..."

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
	key1, err := keys.GetPrivateKeyFromHex("signer1-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}
	key2, err := keys.GetPrivateKeyFromHex("signer2-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err = transaction.SignTransaction(signedTx, key2)
	if err != nil {
		log.Fatal(err)
	}

	// Step 4: Broadcast
	result, err := c.Broadcast(signedTx)
	if err != nil {
		log.Fatal(err)
	}
	if !result.Result || result.Code != api.Return_SUCCESS {
		log.Fatalf("broadcast failed: (%d) %s", result.Code, result.Message)
	}

	fmt.Printf("Multi-sig transfer tx: %x\n", tx.Txid)
}
```

### Validating Signatures with GetTransactionSignWeight

Before broadcasting, you can check whether enough signatures have been collected
using `GetTransactionSignWeight`. This is especially useful when signatures are
collected asynchronously from different parties.

```go
package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
)

func main() {
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	multiSigAddr := "TMultiSigAccountBase58..."
	recipientAddr := "TRecipientBase58..."

	// Build and set permission
	tx, err := c.Transfer(multiSigAddr, recipientAddr, 1_000_000)
	if err != nil {
		log.Fatal(err)
	}
	if err := tx.SetPermissionId(2); err != nil {
		log.Fatal(err)
	}

	// Sign with first key
	key1, err := keys.GetPrivateKeyFromHex("signer1-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		log.Fatal(err)
	}

	// Check current signature weight
	weight, err := c.GetTransactionSignWeight(signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Current weight: %d\n", weight.CurrentWeight)
	if weight.GetPermission() != nil {
		fmt.Printf("Required threshold: %d\n", weight.Permission.Threshold)
	}
	fmt.Printf("Approved signers: %d\n", len(weight.ApprovedList))
	for _, addr := range weight.ApprovedList {
		fmt.Printf("  - %s\n", hex.EncodeToString(addr))
	}

	if weight.GetPermission() != nil && weight.CurrentWeight < weight.Permission.Threshold {
		fmt.Println("Not enough signatures yet — collect more before broadcasting.")
	} else {
		fmt.Println("Threshold met — ready to broadcast!")
	}
}
```

### Signing Externally-Built Transactions

When a transaction is built by a frontend, API, or another service and shared as
`raw_data_hex`, use `FromRawDataHex` to reconstruct it, add your signature, and
broadcast.

```go
package main

import (
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

func main() {
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	// Received from another party (e.g., frontend, API, or co-signer)
	rawDataHex := "0a02..." // hex-encoded protobuf of transaction raw_data

	// Reconstruct the transaction from raw_data_hex
	tx, err := transaction.FromRawDataHex(rawDataHex)
	if err != nil {
		log.Fatal("failed to parse transaction: ", err)
	}

	// Sign with your key
	myKey, err := keys.GetPrivateKeyFromHex("my-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}
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
		fmt.Printf("Need more signatures. Share transaction JSON:\n%s\n", partialJSON)
		fmt.Printf("Signatures so far: %d\n", len(signedTx.Signature))
		return
	}

	// Broadcast when threshold is met
	result, err := c.Broadcast(signedTx)
	if err != nil {
		log.Fatal(err)
	}
	if !result.Result || result.Code != api.Return_SUCCESS {
		log.Fatalf("broadcast failed: (%d) %s", result.Code, result.Message)
	}

	fmt.Println("Transaction broadcast successfully!")
}
```

You can also reconstruct from a full JSON response (as returned by TRON HTTP APIs):

```go
// From JSON (includes raw_data_hex, txID validation, and signatures)
jsonData := []byte(`{"txID":"abc...","raw_data_hex":"0a02...","signature":["sig1hex..."]}`)
tx, err := transaction.FromJSON(jsonData)
if err != nil {
    log.Fatal(err)
}
```

### Multi-Sig TRC20 Token Transfer

TRC20 transfers use `TriggerContract` to call the token's `transfer(address,uint256)` method.
The multi-sig flow is the same: build, set permission, sign with multiple keys, broadcast.

```go
package main

import (
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

func main() {
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	multiSigAddr := "TMultiSigAccountBase58..."
	tokenContract := "TTokenContractBase58..."
	recipientAddr := "TRecipientBase58..."

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
	key1, err := keys.GetPrivateKeyFromHex("signer1-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}
	key2, err := keys.GetPrivateKeyFromHex("signer2-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err = transaction.SignTransaction(signedTx, key2)
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

	fmt.Printf("Multi-sig TRC20 transfer tx: %x\n", tx.Txid)
}
```

### Multi-Sig Resource Delegation

Delegate bandwidth or energy from a multi-sig account to another address.

```go
package main

import (
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

func main() {
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	if err := c.Start(client.GRPCInsecure()); err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	multiSigAddr := "TMultiSigAccountBase58..."
	delegateTo := "TDelegateToBase58..."

	// Delegate 100 TRX worth of energy
	tx, err := c.DelegateResource(
		multiSigAddr,
		delegateTo,
		core.ResourceCode_ENERGY,
		100_000_000, // 100 TRX in sun
		false,       // not locked
		0,           // no lock period
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set active permission for multi-sig
	if err := tx.SetPermissionId(2); err != nil {
		log.Fatal(err)
	}

	// Sign with two keys (2-of-3)
	key1, err := keys.GetPrivateKeyFromHex("signer1-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}
	key2, err := keys.GetPrivateKeyFromHex("signer2-private-key-hex")
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err = transaction.SignTransaction(signedTx, key2)
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

	fmt.Printf("Multi-sig delegation tx: %x\n", tx.Txid)
}
```

### Common Pitfalls

**1. Forgetting to set PermissionId before signing**

`SetPermissionId` modifies the transaction's `raw_data`, which changes the hash that gets
signed. If you sign first and set the permission ID after, the signatures will be invalid.

```go
// WRONG: sign before setting permission
signedTx, _ := transaction.SignTransaction(tx.Transaction, key1)
tx.SetPermissionId(2) // too late — signatures are now invalid

// CORRECT: set permission first, then sign
tx.SetPermissionId(2)
signedTx, _ := transaction.SignTransaction(tx.Transaction, key1)
```

**2. Transaction expiration (TAPOS errors)**

TRON transactions expire after ~60 seconds by default. If collecting signatures takes
longer than this, the transaction will fail with a TAPOS (Transaction as Proof of Stake)
error. Solutions:
- Collect signatures quickly using automated signing services
- Build the transaction closer to when all signers are available
- Re-build the transaction if it expires before all signatures are collected

**3. Using the wrong PermissionId**

- `PermissionId = 0`: Owner permission (default if not set)
- `PermissionId = 2`: First active permission (most common for multi-sig)
- `PermissionId = 3, 4, ...`: Additional active permissions (if configured)

If you get a "permission denied" error, verify the PermissionId matches the permission
that has the signing keys and the required operation type.

**4. Operation not allowed by active permission**

Active permissions have an operations bitmask that controls which transaction types are
allowed. If you configured the active permission to only allow `TransferContract`, trying
to use it for `TriggerSmartContract` will fail. Check your permission's operations when
setting up via `UpdateAccountPermission`.

**5. Incorrect raw_data_hex encoding**

When sharing transactions between systems, ensure the `raw_data_hex` is the protobuf
encoding of `TransactionRaw`, not the full `Transaction`. The `ToRawDataHex` and
`FromRawDataHex` functions handle this correctly. A `0x` prefix is accepted and
automatically stripped.

## Advanced Use Cases

### Resource Manager

```go
package main

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

type ResourceManager struct {
	client     *client.GrpcClient
	privateKey *ecdsa.PrivateKey
	address    address.Address
}

func NewResourceManager(nodeURL, privateKeyHex string) (*ResourceManager, error) {
	client := client.NewGrpcClient(nodeURL)
	err := client.Start()
	if err != nil {
		return nil, err
	}

	privateKey, err := keys.GetPrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return nil, err
	}

	addr := address.BTCECPubkeyToAddress(privateKey.PubKey())

	return &ResourceManager{
		client:     client,
		privateKey: privateKey.ToECDSA(),
		address:    addr,
	}, nil
}

func (rm *ResourceManager) GetResourceInfo() error {
	account, err := rm.client.GetAccountDetailed(rm.address.String())
	if err != nil {
		return err
	}

	fmt.Printf("Account: %s\n", rm.address.String())
	fmt.Printf("Balance: %.6f TRX\n", float64(account.Balance)/1e6)
	fmt.Printf("\nResources:\n")
	fmt.Printf("  Bandwidth:\n")
	fmt.Printf("    Free: %d/%d\n", account.BWTotal-account.BWUsed, account.BWTotal)
	fmt.Printf("    Used: %d/%d\n", account.BWUsed, account.BWTotal)
	fmt.Printf("  Energy:\n")
	fmt.Printf("    Free: %d/%d\n", account.EnergyTotal-account.EnergyUsed, account.EnergyTotal)
	fmt.Printf("    Used: %d/%d\n", account.EnergyUsed, account.EnergyTotal)

	for _, r := range account.FrozenResourcesV2 {
		fmt.Printf("  Frozen %s: %d (delegated: %s)\n", core.ResourceCode_name[int32(r.Type)], r.Amount, r.DelegateTo)
	}

	return nil
}

func (rm *ResourceManager) FreezeForEnergy(amount int64) error {
	tx, err := rm.client.FreezeBalance(
		rm.address.String(),
		"", // no delegate
		core.ResourceCode_ENERGY,
		amount,
	)
	if err != nil {
		return err
	}

	signedTx, _ := transaction.SignTransactionECDSA(tx.Transaction, rm.privateKey)
	result, err := rm.client.Broadcast(signedTx)
	if err != nil {
		return err
	}

	if !result.Result ||
		result.Code != api.Return_SUCCESS {
		return fmt.Errorf("failed to freeze balance: %s", result.Message)
	}

	fmt.Printf("Frozen %d sun for energy: %x\n", tx.Txid)
	return nil
}

func (rm *ResourceManager) DelegateEnergy(to string, amount int64) error {
	toAddr, err := address.Base58ToAddress(to)
	if err != nil {
		return err
	}

	tx, err := rm.client.DelegateResource(
		rm.address.String(),
		toAddr.String(),
		core.ResourceCode_ENERGY,
		amount,
		false, // not locked
		0,     // no lock period
	)
	if err != nil {
		return err
	}

	signedTx, _ := transaction.SignTransactionECDSA(tx.Transaction, rm.privateKey)
	result, err := rm.client.Broadcast(signedTx)
	if err != nil {
		return err
	}

	if !result.Result ||
		result.Code != api.Return_SUCCESS {
		return fmt.Errorf("failed to delegate energy: %s", result.Message)
	}

	fmt.Printf("Delegated %d energy to %s: %x\n", amount, to, tx.Txid)
	return nil
}
```

## Best Practices Summary

1. **Always validate addresses** before using them
2. **Handle errors appropriately** - don't ignore them
3. **Use proper resource limits** for smart contract calls
4. **Implement retry logic** for network operations
5. **Secure private keys** - never hardcode them
6. **Monitor resource usage** to avoid running out of bandwidth/energy
7. **Test on testnet first** before mainnet deployment
8. **Use connection pooling** for high-throughput applications
9. **Implement proper logging** for debugging
10. **Keep transactions atomic** - handle partial failures gracefully