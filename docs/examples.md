# Examples

This document provides practical examples for common use cases with the GoTRON SDK.

## Table of Contents

- [Examples](#examples)
  - [Table of Contents](#table-of-contents)
  - [Basic Operations](#basic-operations)
    - [Create and Fund Account](#create-and-fund-account)
    - [Monitor Account Balance](#monitor-account-balance)
    - [Multi-Signature Transaction](#multi-signature-transaction)
  - [Token Operations](#token-operations)
    - [TRC20 Token Wrapper](#trc20-token-wrapper)
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

### Multi-Signature Transaction

```go
package main

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

func multiSigExample() error {
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := c.Start(client.GRPCInsecure())
	if err != nil {
		return err
	}
	defer c.Stop()

	// Multi-sig account address
	multiSigAddr, _ := address.Base58ToAddress("TMultiSigAddress...")
	toAddr, _ := address.Base58ToAddress("TRecipientAddress...")

	// Create transaction
	tx, err := c.Transfer(multiSigAddr.String(), toAddr.String(), 1_000_000) // 1 TRX
	if err != nil {
		return err
	}

	// Sign with multiple keys
	key1, _ := keys.GetPrivateKeyFromHex("key1-hex")
	key2, _ := keys.GetPrivateKeyFromHex("key2-hex")

	// First signature
	signedTx, err := transaction.SignTransaction(tx.Transaction, key1)
	if err != nil {
		return err
	}

	// Second signature
	signedTx, err = transaction.SignTransaction(signedTx, key2)
	if err != nil {
		return err
	}

	// Broadcast when enough signatures collected
	result, err := c.Broadcast(signedTx)
	if err != nil {
		return err
	}

	if !result.Result ||
		result.Code != api.Return_SUCCESS {
		return fmt.Errorf("broadcast failed: (%d) %s", result.Code, result.Message)
	}

	fmt.Printf("Multi-sig transaction: %x\n", tx.Txid)
	return nil
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
	data := fmt.Sprintf(`[{"address": "%s"}, {"uint256": %d}]`, toAddr.String(), amount)

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