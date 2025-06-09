# SDK Usage Guide

This guide provides comprehensive examples and patterns for using the GoTRON SDK in your Go applications.

## Table of Contents

- [Getting Started](#getting-started)
- [Client Connection](#client-connection)
- [Account Management](#account-management)
- [Transactions](#transactions)
- [Smart Contracts](#smart-contracts)
- [TRC20 Tokens](#trc20-tokens)
- [Key Management](#key-management)
- [Advanced Topics](#advanced-topics)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

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
    err := c.Start()
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
import (
    "fmt"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
)

func getAccountInfo(c *client.GrpcClient, addr string) error {
    // Convert address
    tronAddr, err := address.Base58ToAddress(addr)
    if err != nil {
        return err
    }
    
    // Get account
    account, err := c.GetAccount(tronAddr)
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
    res, err := c.GetAccountResource(tronAddr)
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
import (
    "crypto/ecdsa"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
)

func createAccount() (*ecdsa.PrivateKey, string, error) {
    // Generate new private key
    privateKey, err := keys.GenerateKey()
    if err != nil {
        return nil, "", err
    }
    
    // Get address
    addr := address.PubkeyToAddress(privateKey.PublicKey)
    addrBase58 := address.Address(addr).String()
    
    fmt.Printf("New Address: %s\n", addrBase58)
    fmt.Printf("Private Key: %x\n", privateKey.D.Bytes())
    
    return privateKey, addrBase58, nil
}
```

### Import Account

```go
func importAccount(privateKeyHex string) (string, error) {
    // Import from hex private key
    privateKey, err := keys.GetPrivateKeyByHexString(privateKeyHex)
    if err != nil {
        return "", err
    }
    
    // Get address
    addr := address.PubkeyToAddress(privateKey.PublicKey)
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
    tx, err := c.Transfer(fromAddr, toAddr, amount)
    if err != nil {
        return "", err
    }
    
    // Sign transaction
    signedTx, err := transaction.SignTransaction(tx.Transaction, privateKey)
    if err != nil {
        return "", err
    }
    
    // Broadcast
    result, err := c.Broadcast(signedTx)
    if err != nil {
        return "", err
    }
    
    if result.Code != 0 {
        return "", fmt.Errorf("broadcast failed: %s", result.Message)
    }
    
    return hex.EncodeToString(result.Txid), nil
}
```

### Send with Memo

```go
func sendWithMemo(c *client.GrpcClient, from, to string, amount int64, memo string, pk *ecdsa.PrivateKey) error {
    fromAddr, _ := address.Base58ToAddress(from)
    toAddr, _ := address.Base58ToAddress(to)
    
    // Create transaction with memo
    tx, err := c.TransferWithMemo(fromAddr, toAddr, amount, memo)
    if err != nil {
        return err
    }
    
    // Sign and broadcast
    signedTx, _ := transaction.SignTransaction(tx.Transaction, pk)
    result, err := c.Broadcast(signedTx)
    
    return err
}
```

### Batch Transactions

```go
type Transfer struct {
    To     string
    Amount int64
}

func batchTransfer(c *client.GrpcClient, from string, transfers []Transfer, pk *ecdsa.PrivateKey) error {
    fromAddr, _ := address.Base58ToAddress(from)
    
    for _, transfer := range transfers {
        toAddr, _ := address.Base58ToAddress(transfer.To)
        
        // Create transaction
        tx, err := c.Transfer(fromAddr, toAddr, transfer.Amount)
        if err != nil {
            log.Printf("Failed to create tx to %s: %v", transfer.To, err)
            continue
        }
        
        // Sign and broadcast
        signedTx, _ := transaction.SignTransaction(tx.Transaction, pk)
        result, _ := c.Broadcast(signedTx)
        
        log.Printf("Sent %d to %s: %x", transfer.Amount, transfer.To, result.Txid)
        
        // Wait between transactions
        time.Sleep(1 * time.Second)
    }
    
    return nil
}
```

## Smart Contracts

### Deploy Contract

```go
import (
    "io/ioutil"
    "github.com/fbsobreira/gotron-sdk/pkg/abi"
)

func deployContract(c *client.GrpcClient, owner string, pk *ecdsa.PrivateKey) (string, error) {
    ownerAddr, _ := address.Base58ToAddress(owner)
    
    // Read contract files
    bytecode, err := ioutil.ReadFile("Token.bin")
    if err != nil {
        return "", err
    }
    
    abiData, err := ioutil.ReadFile("Token.abi")
    if err != nil {
        return "", err
    }
    
    // Parse ABI
    contractABI, err := abi.JSON(strings.NewReader(string(abiData)))
    if err != nil {
        return "", err
    }
    
    // Prepare constructor parameters
    params := []interface{}{"My Token", "MTK", uint8(18), uint64(1000000)}
    constructorData, err := contractABI.Pack("", params...)
    if err != nil {
        return "", err
    }
    
    // Combine bytecode and constructor data
    contractCode := append(bytecode, constructorData...)
    
    // Deploy
    tx, err := c.DeployContract(ownerAddr, "MyToken", contractCode, 10000000, 100, 100)
    if err != nil {
        return "", err
    }
    
    // Sign and broadcast
    signedTx, _ := transaction.SignTransaction(tx.Transaction, pk)
    result, err := c.Broadcast(signedTx)
    if err != nil {
        return "", err
    }
    
    // Get contract address from result
    contractAddr := common.BytesToAddress(result.ContractAddress)
    return contractAddr.String(), nil
}
```

### Call Contract (Read)

```go
func callContract(c *client.GrpcClient, contractAddr, method string, params ...interface{}) ([]interface{}, error) {
    contract, _ := address.Base58ToAddress(contractAddr)
    
    // Load ABI
    abiData, _ := ioutil.ReadFile("Token.abi")
    contractABI, _ := abi.JSON(strings.NewReader(string(abiData)))
    
    // Pack method call
    data, err := contractABI.Pack(method, params...)
    if err != nil {
        return nil, err
    }
    
    // Make constant call
    result, err := c.TriggerConstantContract("", contract, data, 0)
    if err != nil {
        return nil, err
    }
    
    // Unpack result
    var output []interface{}
    err = contractABI.UnpackIntoInterface(&output, method, result.ConstantResult[0])
    
    return output, err
}

// Example: Get token balance
balance, _ := callContract(client, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", "balanceOf", userAddress)
```

### Trigger Contract (Write)

```go
func triggerContract(c *client.GrpcClient, from, contractAddr, method string, feeLimit int64, pk *ecdsa.PrivateKey, params ...interface{}) (string, error) {
    fromAddr, _ := address.Base58ToAddress(from)
    contract, _ := address.Base58ToAddress(contractAddr)
    
    // Load ABI
    abiData, _ := ioutil.ReadFile("Token.abi")
    contractABI, _ := abi.JSON(strings.NewReader(string(abiData)))
    
    // Pack method call
    data, err := contractABI.Pack(method, params...)
    if err != nil {
        return "", err
    }
    
    // Create transaction
    tx, err := c.TriggerContract(fromAddr, contract, data, feeLimit, 0, "")
    if err != nil {
        return "", err
    }
    
    // Sign and broadcast
    signedTx, _ := transaction.SignTransaction(tx.Transaction, pk)
    result, err := c.Broadcast(signedTx)
    if err != nil {
        return "", err
    }
    
    return hex.EncodeToString(result.Txid), nil
}

// Example: Transfer tokens
txID, _ := triggerContract(client, myAddress, tokenContract, "transfer", 10000000, privateKey, recipientAddress, big.NewInt(1000))
```

## TRC20 Tokens

### TRC20 Transfer

```go
func transferTRC20(c *client.GrpcClient, from, tokenContract, to string, amount *big.Int, pk *ecdsa.PrivateKey) error {
    fromAddr, _ := address.Base58ToAddress(from)
    contractAddr, _ := address.Base58ToAddress(tokenContract)
    toAddr, _ := address.Base58ToAddress(to)
    
    // Create transfer data
    transferMethod := "a9059cbb" // transfer(address,uint256)
    paddedTo := common.LeftPadBytes(toAddr.Bytes()[1:], 32)
    paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
    
    data := append(common.Hex2Bytes(transferMethod), paddedTo...)
    data = append(data, paddedAmount...)
    
    // Trigger contract
    tx, err := c.TriggerContract(fromAddr, contractAddr, data, 10000000, 0, "")
    if err != nil {
        return err
    }
    
    // Sign and broadcast
    signedTx, _ := transaction.SignTransaction(tx.Transaction, pk)
    _, err = c.Broadcast(signedTx)
    
    return err
}
```

### Get TRC20 Balance

```go
func getTRC20Balance(c *client.GrpcClient, tokenContract, account string) (*big.Int, error) {
    contractAddr, _ := address.Base58ToAddress(tokenContract)
    accountAddr, _ := address.Base58ToAddress(account)
    
    // Create balanceOf call
    balanceOfMethod := "70a08231" // balanceOf(address)
    paddedAddr := common.LeftPadBytes(accountAddr.Bytes()[1:], 32)
    data := append(common.Hex2Bytes(balanceOfMethod), paddedAddr...)
    
    // Make call
    result, err := c.TriggerConstantContract("", contractAddr, data, 0)
    if err != nil {
        return nil, err
    }
    
    // Parse result
    balance := new(big.Int).SetBytes(result.ConstantResult[0])
    return balance, nil
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
    signedTx, err := ks.SignTx(account, tx, nil)
    
    return err
}
```

### HD Wallet

```go
import (
    "github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
    "github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
)

func hdWalletExample() error {
    // Generate mnemonic
    mnemonic, err := mnemonic.GenerateMnemonic(128)
    if err != nil {
        return err
    }
    
    fmt.Printf("Mnemonic: %s\n", mnemonic)
    
    // Create HD wallet
    seed := mnemonic.NewSeedFromMnemonic(mnemonic, "")
    wallet, err := hd.NewFromSeed(seed)
    if err != nil {
        return err
    }
    
    // Derive accounts
    for i := 0; i < 5; i++ {
        path := fmt.Sprintf("m/44'/195'/0'/0/%d", i)
        account, err := wallet.Derive(path)
        if err != nil {
            continue
        }
        
        addr := address.PubkeyToAddress(account.PublicKey)
        fmt.Printf("Account %d: %s\n", i, address.Address(addr).String())
    }
    
    return nil
}
```

## Advanced Topics

### Resource Management

```go
func freezeForResources(c *client.GrpcClient, owner string, amount int64, resource string, pk *ecdsa.PrivateKey) error {
    ownerAddr, _ := address.Base58ToAddress(owner)
    
    var resourceType core.ResourceCode
    switch resource {
    case "BANDWIDTH":
        resourceType = core.ResourceCode_BANDWIDTH
    case "ENERGY":
        resourceType = core.ResourceCode_ENERGY
    default:
        return fmt.Errorf("invalid resource type")
    }
    
    // Freeze balance
    tx, err := c.FreezeBalance(ownerAddr, amount, 3, resourceType, "")
    if err != nil {
        return err
    }
    
    // Sign and broadcast
    signedTx, _ := transaction.SignTransaction(tx.Transaction, pk)
    _, err = c.Broadcast(signedTx)
    
    return err
}
```

### Multi-Signature

```go
func multiSigTransfer(c *client.GrpcClient, from, to string, amount int64, signers []*ecdsa.PrivateKey) error {
    fromAddr, _ := address.Base58ToAddress(from)
    toAddr, _ := address.Base58ToAddress(to)
    
    // Create transaction
    tx, err := c.Transfer(fromAddr, toAddr, amount)
    if err != nil {
        return err
    }
    
    // Sign with multiple keys
    signedTx := tx.Transaction
    for _, signer := range signers {
        signedTx, err = transaction.SignTransaction(signedTx, signer)
        if err != nil {
            return err
        }
    }
    
    // Broadcast
    _, err = c.Broadcast(signedTx)
    return err
}
```

### Event Monitoring

```go
func monitorTransactionEvents(c *client.GrpcClient, txID string) error {
    // Get transaction info which includes events
    txInfo, err := c.GetTransactionInfoByID(txID)
    if err != nil {
        return err
    }
    
    // Check transaction status
    if txInfo.Result != core.TransactionInfo_SUCCESS {
        return fmt.Errorf("transaction failed: %s", txInfo.Result.String())
    }
    
    // Process contract events
    for _, event := range txInfo.Log {
        fmt.Printf("Event Topics:\n")
        for i, topic := range event.Topics {
            fmt.Printf("  Topic[%d]: %x\n", i, topic)
        }
        fmt.Printf("Event Data: %x\n", event.Data)
        fmt.Printf("Contract Address: %x\n", event.Address)
        
        // Parse event data based on ABI
        // Example: Transfer event
        // topic[0] = keccak256("Transfer(address,address,uint256)")
        // topic[1] = from address
        // topic[2] = to address
        // data = amount
    }
    
    // Process internal transactions
    for _, internal := range txInfo.InternalTransactions {
        fmt.Printf("Internal TX: %x\n", internal.Hash)
        fmt.Printf("  From: %x\n", internal.CallerAddress)
        fmt.Printf("  To: %x\n", internal.TransferToAddress)
        fmt.Printf("  Amount: %d\n", internal.CallValueInfo)
    }
    
    return nil
}

// Monitor new blocks for events
func monitorBlockEvents(c *client.GrpcClient, startBlock int64) error {
    currentBlock := startBlock
    
    for {
        // Get block
        block, err := c.GetBlockByNum(currentBlock)
        if err != nil {
            // Block might not exist yet
            time.Sleep(3 * time.Second)
            continue
        }
        
        // Process transactions in block
        for _, tx := range block.Transactions {
            txID := hex.EncodeToString(tx.Txid)
            
            // Get transaction info
            txInfo, err := c.GetTransactionInfoByID(txID)
            if err != nil {
                continue
            }
            
            // Check if transaction has events
            if len(txInfo.Log) > 0 {
                fmt.Printf("Transaction %s has %d events\n", txID, len(txInfo.Log))
                monitorTransactionEvents(c, txID)
            }
        }
        
        currentBlock++
        time.Sleep(3 * time.Second) // TRON block time
    }
}
```

## Error Handling

### Comprehensive Error Handling

```go
func safeTransfer(c *client.GrpcClient, from, to string, amount int64, pk *ecdsa.PrivateKey) error {
    // Validate inputs
    if amount <= 0 {
        return fmt.Errorf("invalid amount: %d", amount)
    }
    
    fromAddr, err := address.Base58ToAddress(from)
    if err != nil {
        return fmt.Errorf("invalid from address: %w", err)
    }
    
    toAddr, err := address.Base58ToAddress(to)
    if err != nil {
        return fmt.Errorf("invalid to address: %w", err)
    }
    
    // Check balance
    account, err := c.GetAccount(fromAddr)
    if err != nil {
        return fmt.Errorf("failed to get account: %w", err)
    }
    
    if account.Balance < amount {
        return fmt.Errorf("insufficient balance: have %d, want %d", account.Balance, amount)
    }
    
    // Create transaction
    tx, err := c.Transfer(fromAddr, toAddr, amount)
    if err != nil {
        return fmt.Errorf("failed to create transaction: %w", err)
    }
    
    // Sign transaction
    signedTx, err := transaction.SignTransaction(tx.Transaction, pk)
    if err != nil {
        return fmt.Errorf("failed to sign transaction: %w", err)
    }
    
    // Broadcast with retry
    var result *api.Return
    for i := 0; i < 3; i++ {
        result, err = c.Broadcast(signedTx)
        if err == nil {
            break
        }
        time.Sleep(time.Second * time.Duration(i+1))
    }
    
    if err != nil {
        return fmt.Errorf("failed to broadcast after retries: %w", err)
    }
    
    if result.Code != 0 {
        return fmt.Errorf("transaction failed: %s", result.Message)
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
    if err := c.Start(); err != nil {
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
type TransactionBuilder struct {
    client   *client.GrpcClient
    from     string
    to       string
    amount   int64
    memo     string
    feeLimit int64
}

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

func (tb *TransactionBuilder) Build() (*core.Transaction, error) {
    fromAddr, _ := address.Base58ToAddress(tb.from)
    toAddr, _ := address.Base58ToAddress(tb.to)
    
    if tb.memo != "" {
        return tb.client.TransferWithMemo(fromAddr, toAddr, tb.amount, tb.memo)
    }
    
    return tb.client.Transfer(fromAddr, toAddr, tb.amount)
}

// Usage
tx, err := NewTransactionBuilder(client).
    From("TRX...").
    To("TLy...").
    Amount(1000000).
    Build()
```

### 3. Rate Limiting

```go
import "golang.org/x/time/rate"

type RateLimitedClient struct {
    client  *client.GrpcClient
    limiter *rate.Limiter
}

func NewRateLimitedClient(node string, rps int) *RateLimitedClient {
    return &RateLimitedClient{
        client:  client.NewGrpcClient(node),
        limiter: rate.NewLimiter(rate.Limit(rps), 1),
    }
}

func (rlc *RateLimitedClient) Transfer(from, to string, amount int64) error {
    // Wait for rate limit
    if err := rlc.limiter.Wait(context.Background()); err != nil {
        return err
    }
    
    // Proceed with transfer
    // ...
}
```

### 4. Configuration Management

```go
type Config struct {
    Node     string
    Network  string
    APIKey   string
    Timeout  time.Duration
    MaxRetry int
}

func LoadConfig(path string) (*Config, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    
    // Set defaults
    if config.Timeout == 0 {
        config.Timeout = 60 * time.Second
    }
    if config.MaxRetry == 0 {
        config.MaxRetry = 3
    }
    
    return &config, nil
}
```

## Testing

### Unit Testing Example

```go
func TestTransfer(t *testing.T) {
    // Setup
    client := client.NewGrpcClient("grpc.shasta.trongrid.io:50051")
    err := client.Start()
    require.NoError(t, err)
    defer client.Stop()
    
    // Test data
    from := "TTestFromAddress"
    to := "TTestToAddress"
    amount := int64(1000000)
    
    // Create mock transaction
    tx, err := client.Transfer(from, to, amount)
    assert.NoError(t, err)
    assert.NotNil(t, tx)
    
    // Verify transaction details
    assert.Equal(t, amount, tx.RawData.Contract[0].Parameter.Value["amount"])
}
```

## Resources

- [TRON Documentation](https://developers.tron.network/)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [Go gRPC](https://grpc.io/docs/languages/go/)
- [Ethereum ABI](https://solidity.readthedocs.io/en/latest/abi-spec.html)