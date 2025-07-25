# SDK Usage Guide

This guide provides comprehensive examples and patterns for using the GoTRON SDK in your Go applications.

## Table of Contents

- [SDK Usage Guide](#sdk-usage-guide)
  - [Table of Contents](#table-of-contents)
  - [Getting Started](#getting-started)
    - [Installation](#installation)
    - [Basic Import](#basic-import)
  - [Client Connection](#client-connection)
    - [Creating a Client](#creating-a-client)
    - [Client with Options](#client-with-options)
    - [Multiple Network Support](#multiple-network-support)
  - [Account Management](#account-management)
    - [Get Account Information](#get-account-information)
    - [Create New Account](#create-new-account)
    - [Import Account](#import-account)
  - [Transactions](#transactions)
    - [Send TRX](#send-trx)
    - [Send with Memo](#send-with-memo)
  - [Key Management](#key-management)
    - [Using Keystore](#using-keystore)
    - [HD Wallet](#hd-wallet)
  - [Best Practices](#best-practices)
    - [1. Connection Management](#1-connection-management)
    - [2. Transaction Builder Pattern](#2-transaction-builder-pattern)
  - [Resources](#resources)

## Getting Started

### Installation

Add GoTRON SDK to your project:

```bash
go get github.com/fbsobreira/gotron-sdk
```

### Basic Import

```go
import (
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/keystore"
    "github.com/fbsobreira/gotron-sdk/pkg/common"
)
```

## Client Connection

### Creating a Client

```go
package main

import (
    "log"
    "github.com/fbsobreira/gotron-sdk/pkg/client"
)

func main() {
    // Create client with default options
    c := client.NewGrpcClient("grpc.trongrid.io:50051")
    
    // Start the connection
    err := c.Start(client.GRPCInsecure())
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer c.Stop()
    
    // Client is ready to use
}
```

### Client with Options

```go
// Create client with custom options
c := client.NewGrpcClientWithTimeout("grpc.trongrid.io:50051", 30)

// With TLS
opts := make([]grpc.DialOption, 0)
opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
c.Conn, err = grpc.Dial(c.Address, opts...)

// With API Key
ctx := metadata.AppendToOutgoingContext(context.Background(), "TRON-PRO-API-KEY", "your-api-key")
```

### Multiple Network Support

```go
type Network struct {
    Name   string
    URL    string
    APIKey string
}

var networks = map[string]Network{
    "mainnet": {
        Name: "Mainnet",
        URL:  "grpc.trongrid.io:50051",
    },
    "shasta": {
        Name: "Shasta Testnet",
        URL:  "grpc.shasta.trongrid.io:50051",
    },
    "nile": {
        Name: "Nile Testnet",
        URL:  "grpc.nile.trongrid.io:50051",
    },
}

// Connect to specific network
network := networks["mainnet"]
client := client.NewGrpcClient(network.URL)
```

## Account Management

### Get Account Information

```go
func getAccountInfo(c *client.GrpcClient, addr string) error {
	// Convert address
	tronAddr, err := address.Base58ToAddress(addr)
	if err != nil {
		return err
	}

	// Get account
	account, err := c.GetAccount(tronAddr.String())
	if err != nil {
		return err
	}

	// Display information
	fmt.Printf("Address: %s\n", addr)
	fmt.Printf("Balance: %d sun (%f TRX)\n",
		account.Balance,
		float64(account.Balance)/1e6)
	fmt.Printf("Created: %v\n", account.CreateTime)

	// Check resources
	res, err := c.GetAccountResource(tronAddr.String())
	if err == nil {
		fmt.Printf("Bandwidth: %d/%d\n",
			res.FreeNetUsed,
			res.FreeNetLimit)
		fmt.Printf("Energy: %d/%d\n",
			res.EnergyUsed,
			res.EnergyLimit)
	}

	return nil
}
```

### Create New Account

```go
func createAccount() (*ecdsa.PrivateKey, string, error) {
	// Generate new private key
	privateKey, err := keys.GenerateKey()
	if err != nil {
		return nil, "", err
	}

	// Get address
	addr := address.PubkeyToAddress(privateKey.ToECDSA().PublicKey)
	addrBase58 := address.Address(addr).String()

	fmt.Printf("New Address: %s\n", addrBase58)
	fmt.Printf("Private Key: %x\n", privateKey.Serialize())

	return privateKey.ToECDSA(), addrBase58, nil
}
```

### Import Account

```go
func importAccount(privateKeyHex string) (string, error) {
    // Import from hex private key
    privateKey, err := keys.GetPrivateKeyFromHex(privateKeyHex)
    if err != nil {
        return "", err
    }
    
    // Get address
    addr := address.PubkeyToAddress(privateKey.ToECDSA().PublicKey)
    return address.Address(addr).String(), nil
}
```

## Transactions

### Send TRX

```go
import (
    "github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
)


func sendTRX(c *client.GrpcClient, from, to string, amount int64, privateKey *ecdsa.PrivateKey) (string, error) {
	// Convert addresses
	fromAddr, _ := address.Base58ToAddress(from)
	toAddr, _ := address.Base58ToAddress(to)

	// Create transaction
	tx, err := c.Transfer(fromAddr.String(), toAddr.String(), amount)
	if err != nil {
		return "", err
	}

	// Sign transaction
	signedTx, err := transaction.SignTransactionECDSA(tx.Transaction, privateKey)
	if err != nil {
		return "", err
	}

	// Broadcast
	result, err := c.Broadcast(signedTx)
	if err != nil {
		return "", err
	}

	if !result.Result ||
		result.Code != api.Return_SUCCESS {
		return "", fmt.Errorf("transaction failed: (%d) %s", result.Code, result.Message)
	}

	return hex.EncodeToString(tx.Txid), nil
}
```

### Send with Memo

```go
func sendWithMemo(c *client.GrpcClient, from, to string, amount int64, memo string, pk *ecdsa.PrivateKey) (string, error) {
	fromAddr, _ := address.Base58ToAddress(from)
	toAddr, _ := address.Base58ToAddress(to)

	// Create transaction with memo
	tx, err := c.Transfer(fromAddr.String(), toAddr.String(), amount)
	if err != nil {
		return "", err
	}

	err = tx.SetData(memo)
	if err != nil {
		return "", err
	}

	// Sign and broadcast
	signedTx, err := transaction.SignTransactionECDSA(tx.Transaction, pk)
	if err != nil {
		return "", err
	}

	result, err := c.Broadcast(signedTx)
	if err != nil {
		return "", err
	}

	if !result.Result ||
		result.Code != api.Return_SUCCESS {
		return "", fmt.Errorf("transaction failed: (%d) %s", result.Code, result.Message)
	}

	return hex.EncodeToString(tx.Txid), nil

}
```

## Key Management

### Using Keystore

```go
import (
    "github.com/fbsobreira/gotron-sdk/pkg/keystore"
)

func keystoreExample() error {
	// Create keystore
	ks := keystore.NewKeyStore("./keystore", keystore.StandardScryptN, keystore.StandardScryptP)

	// Create new account
	account, err := ks.NewAccount("password")
	if err != nil {
		return err
	}

	fmt.Printf("Created account: %s\n", account.Address.String())

	// List accounts
	accounts := ks.Accounts()
	for _, acc := range accounts {
		fmt.Printf("Account: %s\n", acc.Address.String())
	}

	// Unlock account for signing
	err = ks.Unlock(account, "password")
	if err != nil {
		return err
	}

	// Sign transaction
	tx := &core.Transaction{} // Your transaction

	_, err = ks.SignTx(account, tx)
	if err != nil {
		return err
	}

	fmt.Printf("Signature: %x\n", tx.Signature[0])

	return err
}
```

### HD Wallet

```go
import (
    "github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
)

func hdWalletExample() error {
	// Generate mnemonic
	mnemonic := mnemonic.Generate()

	fmt.Printf("Mnemonic: %s\n", mnemonic)

	// Create HD wallet
	for i := 0; i < 5; i++ {
		private, _ := keys.FromMnemonicSeedAndPassphrase(mnemonic, "", i)
		fmt.Printf("--------------\n")
		fmt.Printf("HD Path for index %d: 44'/195'/0'/0/%d\n", i, i)
		fmt.Printf("Private Key: %x\n", private.Serialize())
		// Convert to Tron address
		addr := address.PubkeyToAddress(private.ToECDSA().PublicKey)
		fmt.Printf("Address: %s\n", addr.String())
	}

	return nil
}
```

## Best Practices

### 1. Connection Management

```go
type TronClient struct {
    client *client.GrpcClient
    mu     sync.Mutex
}

func NewTronClient(node string) (*TronClient, error) {
    c := client.NewGrpcClient(node)
    if err := c.Start(client.GRPCInsecure()); err != nil {
        return nil, err
    }
    
    return &TronClient{client: c}, nil
}

func (tc *TronClient) Close() {
    tc.mu.Lock()
    defer tc.mu.Unlock()
    
    if tc.client != nil {
        tc.client.Stop()
        tc.client = nil
    }
}
```

### 2. Transaction Builder Pattern

```go
func NewTransactionBuilder(c *client.GrpcClient) *TransactionBuilder {
	return &TransactionBuilder{
		client:   c,
		feeLimit: 1000000, // Default 1 TRX
	}
}

func (tb *TransactionBuilder) From(addr string) *TransactionBuilder {
	tb.from = addr
	return tb
}

func (tb *TransactionBuilder) To(addr string) *TransactionBuilder {
	tb.to = addr
	return tb
}

func (tb *TransactionBuilder) Amount(amount int64) *TransactionBuilder {
	tb.amount = amount
	return tb
}

func (tb *TransactionBuilder) Build() (*api.TransactionExtention, error) {
	fromAddr, _ := address.Base58ToAddress(tb.from)
	toAddr, _ := address.Base58ToAddress(tb.to)

	tx, err := tb.client.Transfer(fromAddr.String(), toAddr.String(), tb.amount)
	if err != nil {
		return nil, err
	}

	if tb.memo != "" {
		tx.SetData(tb.memo)
	}

	return tx, nil
}

// Usage
tx, err := NewTransactionBuilder(client).
    From("TRX...").
    To("TLy...").
    Amount(1000000).
    Build()
```

## Resources

- [TRON Documentation](https://developers.tron.network/)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [Go gRPC](https://grpc.io/docs/languages/go/)
- [Ethereum ABI](https://solidity.readthedocs.io/en/latest/abi-spec.html)