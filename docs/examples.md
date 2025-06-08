# Examples

This document provides practical examples for common use cases with the GoTRON SDK.

## Table of Contents

- [Basic Operations](#basic-operations)
- [Token Operations](#token-operations)
- [Smart Contract Interactions](#smart-contract-interactions)
- [Advanced Use Cases](#advanced-use-cases)
- [Full Applications](#full-applications)

## Basic Operations

### Create and Fund Account

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
    "github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
)

func main() {
    // Connect to network
    c := client.NewGrpcClient("grpc.trongrid.io:50051")
    err := c.Start()
    if err != nil {
        log.Fatal(err)
    }
    defer c.Stop()
    
    // Create new account
    privateKey, err := keys.GenerateKey()
    if err != nil {
        log.Fatal(err)
    }
    
    addr := address.PubkeyToAddress(privateKey.PublicKey)
    fmt.Printf("New address: %s\n", address.Address(addr).String())
    fmt.Printf("Private key: %x\n", privateKey.D.Bytes())
    
    // Fund account (requires funded account)
    funderKey, _ := keys.GetPrivateKeyByHexString("your-private-key")
    funderAddr := address.PubkeyToAddress(funderKey.PublicKey)
    
    // Send 100 TRX
    tx, err := c.Transfer(funderAddr, addr, 100000000) // 100 TRX = 100 * 1e6 sun
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
    
    fmt.Printf("Transaction ID: %x\n", result.Txid)
}
```

### Monitor Account Balance

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
)

func monitorBalance(c *client.GrpcClient, addr string, interval time.Duration) {
    tronAddr, err := address.Base58ToAddress(addr)
    if err != nil {
        log.Fatal(err)
    }
    
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    fmt.Printf("Monitoring balance for %s\n", addr)
    
    for {
        select {
        case <-ticker.C:
            account, err := c.GetAccount(tronAddr)
            if err != nil {
                log.Printf("Error getting account: %v", err)
                continue
            }
            
            balance := float64(account.Balance) / 1e6
            fmt.Printf("[%s] Balance: %.6f TRX\n", 
                time.Now().Format("15:04:05"), balance)
        }
    }
}

func main() {
    c := client.NewGrpcClient("grpc.trongrid.io:50051")
    err := c.Start()
    if err != nil {
        log.Fatal(err)
    }
    defer c.Stop()
    
    // Monitor balance every 30 seconds
    monitorBalance(c, "TRX6Q82wMqWNbCCpKPfRvVmfSm5N2TwrJw", 30*time.Second)
}
```

### Multi-Signature Transaction

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
    "github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
)

func multiSigExample() error {
    c := client.NewGrpcClient("grpc.trongrid.io:50051")
    err := c.Start()
    if err != nil {
        return err
    }
    defer c.Stop()
    
    // Multi-sig account address
    multiSigAddr, _ := address.Base58ToAddress("TMultiSigAddress...")
    toAddr, _ := address.Base58ToAddress("TRecipientAddress...")
    
    // Create transaction
    tx, err := c.Transfer(multiSigAddr, toAddr, 1000000) // 1 TRX
    if err != nil {
        return err
    }
    
    // Sign with multiple keys
    key1, _ := keys.GetPrivateKeyByHexString("key1-hex")
    key2, _ := keys.GetPrivateKeyByHexString("key2-hex")
    
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
    
    fmt.Printf("Multi-sig transaction: %x\n", result.Txid)
    return nil
}
```

## Token Operations

### TRC20 Token Wrapper

```go
package main

import (
    "fmt"
    "math/big"
    
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/common"
    "github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
)

type TRC20Token struct {
    client   *client.GrpcClient
    contract common.Address
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
    data := common.Hex2Bytes("06fdde03") // name()
    
    result, err := t.client.TriggerConstantContract("", t.contract, data, 0)
    if err != nil {
        return "", err
    }
    
    // Parse string from result
    return parseString(result.ConstantResult[0]), nil
}

func (t *TRC20Token) Symbol() (string, error) {
    data := common.Hex2Bytes("95d89b41") // symbol()
    
    result, err := t.client.TriggerConstantContract("", t.contract, data, 0)
    if err != nil {
        return "", err
    }
    
    return parseString(result.ConstantResult[0]), nil
}

func (t *TRC20Token) Decimals() (uint8, error) {
    data := common.Hex2Bytes("313ce567") // decimals()
    
    result, err := t.client.TriggerConstantContract("", t.contract, data, 0)
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
    
    data := common.Hex2Bytes("70a08231") // balanceOf(address)
    data = append(data, common.LeftPadBytes(addr.Bytes()[1:], 32)...)
    
    result, err := t.client.TriggerConstantContract("", t.contract, data, 0)
    if err != nil {
        return nil, err
    }
    
    return new(big.Int).SetBytes(result.ConstantResult[0]), nil
}

func (t *TRC20Token) Transfer(from string, to string, amount *big.Int, privateKey *ecdsa.PrivateKey) (string, error) {
    fromAddr, _ := address.Base58ToAddress(from)
    toAddr, _ := address.Base58ToAddress(to)
    
    data := common.Hex2Bytes("a9059cbb") // transfer(address,uint256)
    data = append(data, common.LeftPadBytes(toAddr.Bytes()[1:], 32)...)
    data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
    
    tx, err := t.client.TriggerContract(fromAddr, t.contract, data, 10000000, 0, "")
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
    
    return fmt.Sprintf("%x", result.Txid), nil
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
```

### TRC10 Token Operations

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
    "github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
)

func issueTRC10Token() error {
    c := client.NewGrpcClient("grpc.trongrid.io:50051")
    err := c.Start()
    if err != nil {
        return err
    }
    defer c.Stop()
    
    // Issuer private key
    privateKey, _ := keys.GetPrivateKeyByHexString("your-private-key")
    issuerAddr := address.PubkeyToAddress(privateKey.PublicKey)
    
    // Token parameters
    name := "MyToken"
    abbr := "MTK"
    totalSupply := int64(1000000000) // 1 billion
    trxNum := int32(1)               // 1 TRX
    num := int32(1000)               // = 1000 tokens
    startTime := time.Now().Unix() * 1000
    endTime := startTime + (30 * 24 * 60 * 60 * 1000) // 30 days
    description := "My awesome token"
    url := "https://mytoken.com"
    freeAssetNetLimit := int64(0)
    publicFreeAssetNetLimit := int64(0)
    precision := int32(6) // 6 decimals
    
    // Frozen supply (optional)
    frozenSupply := map[int64]int64{
        90 * 24 * 60 * 60 * 1000: 100000000, // Freeze 100M for 90 days
    }
    
    // Create asset issue transaction
    tx, err := c.CreateAssetIssue(
        issuerAddr, name, abbr, totalSupply,
        trxNum, num, startTime, endTime,
        description, url, freeAssetNetLimit,
        publicFreeAssetNetLimit, precision, frozenSupply,
    )
    
    if err != nil {
        return err
    }
    
    // Sign and broadcast
    signedTx, _ := transaction.SignTransaction(tx.Transaction, privateKey)
    result, err := c.Broadcast(signedTx)
    if err != nil {
        return err
    }
    
    fmt.Printf("Token issued! Transaction: %x\n", result.Txid)
    return nil
}

func participateInICO() error {
    c := client.NewGrpcClient("grpc.trongrid.io:50051")
    err := c.Start()
    if err != nil {
        return err
    }
    defer c.Stop()
    
    // Participant
    privateKey, _ := keys.GetPrivateKeyByHexString("participant-private-key")
    participantAddr := address.PubkeyToAddress(privateKey.PublicKey)
    
    // Token issuer address
    issuerAddr, _ := address.Base58ToAddress("TTokenIssuerAddress...")
    
    // Participate with 1000 TRX
    amount := int64(1000000000) // 1000 TRX in sun
    tokenID := "1000001"        // Token ID
    
    tx, err := c.ParticipateAssetIssue(participantAddr, issuerAddr, tokenID, amount)
    if err != nil {
        return err
    }
    
    signedTx, _ := transaction.SignTransaction(tx.Transaction, privateKey)
    result, err := c.Broadcast(signedTx)
    if err != nil {
        return err
    }
    
    fmt.Printf("Participated in ICO! Transaction: %x\n", result.Txid)
    return nil
}
```

## Smart Contract Interactions

### Complete DApp Example - Simple Lottery

```go
package main

import (
    "encoding/hex"
    "fmt"
    "io/ioutil"
    "log"
    "math/big"
    "strings"
    
    "github.com/fbsobreira/gotron-sdk/pkg/abi"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
)

// Lottery contract ABI
const lotteryABI = `[
    {
        "inputs": [{"name": "_ticketPrice", "type": "uint256"}],
        "name": "constructor",
        "type": "constructor"
    },
    {
        "inputs": [],
        "name": "buyTicket",
        "outputs": [],
        "type": "function",
        "payable": true
    },
    {
        "inputs": [],
        "name": "drawWinner",
        "outputs": [],
        "type": "function"
    },
    {
        "inputs": [],
        "name": "getPlayers",
        "outputs": [{"name": "", "type": "address[]"}],
        "type": "function",
        "constant": true
    },
    {
        "inputs": [],
        "name": "getBalance",
        "outputs": [{"name": "", "type": "uint256"}],
        "type": "function",
        "constant": true
    }
]`

type Lottery struct {
    client       *client.GrpcClient
    contractAddr common.Address
    abi          abi.ABI
}

func NewLottery(client *client.GrpcClient, contractAddr string) (*Lottery, error) {
    addr, err := address.Base58ToAddress(contractAddr)
    if err != nil {
        return nil, err
    }
    
    contractABI, err := abi.JSON(strings.NewReader(lotteryABI))
    if err != nil {
        return nil, err
    }
    
    return &Lottery{
        client:       client,
        contractAddr: addr,
        abi:          contractABI,
    }, nil
}

func (l *Lottery) Deploy(owner string, ticketPrice *big.Int, privateKey *ecdsa.PrivateKey) (string, error) {
    ownerAddr, _ := address.Base58ToAddress(owner)
    
    // Read compiled bytecode
    bytecode, err := ioutil.ReadFile("Lottery.bin")
    if err != nil {
        return "", err
    }
    
    // Pack constructor
    constructorData, err := l.abi.Pack("", ticketPrice)
    if err != nil {
        return "", err
    }
    
    contractCode := append(bytecode, constructorData...)
    
    // Deploy contract
    tx, err := l.client.DeployContract(ownerAddr, "Lottery", contractCode, 10000000, 100, 100)
    if err != nil {
        return "", err
    }
    
    signedTx, _ := transaction.SignTransaction(tx.Transaction, privateKey)
    result, err := l.client.Broadcast(signedTx)
    if err != nil {
        return "", err
    }
    
    contractAddr := common.BytesToAddress(result.ContractAddress)
    return contractAddr.String(), nil
}

func (l *Lottery) BuyTicket(buyer string, value *big.Int, privateKey *ecdsa.PrivateKey) error {
    buyerAddr, _ := address.Base58ToAddress(buyer)
    
    data, err := l.abi.Pack("buyTicket")
    if err != nil {
        return err
    }
    
    // Call with TRX value
    tx, err := l.client.TriggerContract(
        buyerAddr, 
        l.contractAddr, 
        data, 
        10000000,           // fee limit
        value.Int64(),      // call value in sun
        "",                 // token ID (empty for TRX)
    )
    if err != nil {
        return err
    }
    
    signedTx, _ := transaction.SignTransaction(tx.Transaction, privateKey)
    _, err = l.client.Broadcast(signedTx)
    return err
}

func (l *Lottery) DrawWinner(owner string, privateKey *ecdsa.PrivateKey) error {
    ownerAddr, _ := address.Base58ToAddress(owner)
    
    data, err := l.abi.Pack("drawWinner")
    if err != nil {
        return err
    }
    
    tx, err := l.client.TriggerContract(ownerAddr, l.contractAddr, data, 10000000, 0, "")
    if err != nil {
        return err
    }
    
    signedTx, _ := transaction.SignTransaction(tx.Transaction, privateKey)
    _, err = l.client.Broadcast(signedTx)
    return err
}

func (l *Lottery) GetPlayers() ([]string, error) {
    data, err := l.abi.Pack("getPlayers")
    if err != nil {
        return nil, err
    }
    
    result, err := l.client.TriggerConstantContract("", l.contractAddr, data, 0)
    if err != nil {
        return nil, err
    }
    
    var addresses []common.Address
    err = l.abi.UnpackIntoInterface(&addresses, "getPlayers", result.ConstantResult[0])
    if err != nil {
        return nil, err
    }
    
    players := make([]string, len(addresses))
    for i, addr := range addresses {
        players[i] = address.Address(addr).String()
    }
    
    return players, nil
}

func (l *Lottery) GetBalance() (*big.Int, error) {
    data, err := l.abi.Pack("getBalance")
    if err != nil {
        return nil, err
    }
    
    result, err := l.client.TriggerConstantContract("", l.contractAddr, data, 0)
    if err != nil {
        return nil, err
    }
    
    balance := new(big.Int).SetBytes(result.ConstantResult[0])
    return balance, nil
}

// Example usage
func runLottery() {
    c := client.NewGrpcClient("grpc.trongrid.io:50051")
    c.Start()
    defer c.Stop()
    
    // Deploy lottery
    ownerKey, _ := keys.GetPrivateKeyByHexString("owner-private-key")
    ownerAddr := address.PubkeyToAddress(ownerKey.PublicKey).String()
    
    lottery := &Lottery{client: c}
    ticketPrice := big.NewInt(10000000) // 10 TRX
    
    contractAddr, err := lottery.Deploy(ownerAddr, ticketPrice, ownerKey)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Lottery deployed at: %s\n", contractAddr)
    
    // Initialize with deployed address
    lottery, _ = NewLottery(c, contractAddr)
    
    // Buy tickets
    player1Key, _ := keys.GetPrivateKeyByHexString("player1-key")
    player1Addr := address.PubkeyToAddress(player1Key.PublicKey).String()
    
    err = lottery.BuyTicket(player1Addr, ticketPrice, player1Key)
    if err != nil {
        log.Fatal(err)
    }
    
    // Check players
    players, _ := lottery.GetPlayers()
    fmt.Printf("Current players: %v\n", players)
    
    // Check balance
    balance, _ := lottery.GetBalance()
    fmt.Printf("Lottery balance: %d sun\n", balance)
    
    // Draw winner (only owner can do this)
    err = lottery.DrawWinner(ownerAddr, ownerKey)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Winner drawn!")
}
```

## Advanced Use Cases

### Automated Market Maker Bot

```go
package main

import (
    "fmt"
    "log"
    "math/big"
    "time"
    
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
)

type AMMBot struct {
    client      *client.GrpcClient
    privateKey  *ecdsa.PrivateKey
    address     string
    exchangeID  int64
    minProfit   float64
    maxTrade    int64
}

func NewAMMBot(nodeURL, privateKeyHex string, exchangeID int64) (*AMMBot, error) {
    client := client.NewGrpcClient(nodeURL)
    err := client.Start()
    if err != nil {
        return nil, err
    }
    
    privateKey, err := keys.GetPrivateKeyByHexString(privateKeyHex)
    if err != nil {
        return nil, err
    }
    
    addr := address.PubkeyToAddress(privateKey.PublicKey)
    
    return &AMMBot{
        client:     client,
        privateKey: privateKey,
        address:    address.Address(addr).String(),
        exchangeID: exchangeID,
        minProfit:  0.01, // 1% minimum profit
        maxTrade:   1000000000, // 1000 TRX max per trade
    }, nil
}

func (bot *AMMBot) Start() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    log.Printf("AMM Bot started for exchange %d", bot.exchangeID)
    
    for {
        select {
        case <-ticker.C:
            bot.checkArbitrage()
        }
    }
}

func (bot *AMMBot) checkArbitrage() {
    // Get exchange info
    exchange, err := bot.client.GetExchangeByID(bot.exchangeID)
    if err != nil {
        log.Printf("Error getting exchange: %v", err)
        return
    }
    
    // Calculate current rates
    token1Balance := exchange.FirstTokenBalance
    token2Balance := exchange.SecondTokenBalance
    
    // Simple constant product formula: x * y = k
    k := new(big.Int).Mul(
        big.NewInt(token1Balance),
        big.NewInt(token2Balance),
    )
    
    // Check if profitable trade exists
    // This is a simplified example - real implementation would need:
    // 1. External price oracle
    // 2. Slippage calculation
    // 3. Gas cost consideration
    
    log.Printf("Exchange %d - Token1: %d, Token2: %d", 
        bot.exchangeID, token1Balance, token2Balance)
}

func (bot *AMMBot) executeTrade(tokenIn string, amountIn int64, minAmountOut int64) error {
    addr, _ := address.Base58ToAddress(bot.address)
    
    tx, err := bot.client.TransactionWithExchange(
        addr,
        bot.exchangeID,
        tokenIn,
        amountIn,
        minAmountOut,
    )
    if err != nil {
        return err
    }
    
    signedTx, _ := transaction.SignTransaction(tx.Transaction, bot.privateKey)
    result, err := bot.client.Broadcast(signedTx)
    if err != nil {
        return err
    }
    
    log.Printf("Trade executed: %x", result.Txid)
    return nil
}
```

### Resource Manager

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
    "github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

type ResourceManager struct {
    client     *client.GrpcClient
    privateKey *ecdsa.PrivateKey
    address    common.Address
}

func NewResourceManager(nodeURL, privateKeyHex string) (*ResourceManager, error) {
    client := client.NewGrpcClient(nodeURL)
    err := client.Start()
    if err != nil {
        return nil, err
    }
    
    privateKey, err := keys.GetPrivateKeyByHexString(privateKeyHex)
    if err != nil {
        return nil, err
    }
    
    addr := address.PubkeyToAddress(privateKey.PublicKey)
    
    return &ResourceManager{
        client:     client,
        privateKey: privateKey,
        address:    addr,
    }, nil
}

func (rm *ResourceManager) GetResourceInfo() error {
    account, err := rm.client.GetAccount(rm.address)
    if err != nil {
        return err
    }
    
    resources, err := rm.client.GetAccountResource(rm.address)
    if err != nil {
        return err
    }
    
    fmt.Printf("Account: %s\n", address.Address(rm.address).String())
    fmt.Printf("Balance: %.6f TRX\n", float64(account.Balance)/1e6)
    fmt.Printf("\nResources:\n")
    fmt.Printf("  Bandwidth:\n")
    fmt.Printf("    Free: %d/%d\n", resources.FreeNetUsed, resources.FreeNetLimit)
    fmt.Printf("    Total: %d/%d\n", resources.NetUsed, resources.NetLimit)
    fmt.Printf("  Energy:\n")
    fmt.Printf("    Used: %d/%d\n", resources.EnergyUsed, resources.EnergyLimit)
    
    // Calculate frozen balance
    totalFrozen := int64(0)
    for _, frozen := range account.Frozen {
        totalFrozen += frozen.FrozenBalance
        fmt.Printf("\n  Frozen: %.6f TRX for %s\n", 
            float64(frozen.FrozenBalance)/1e6,
            frozen.Resource.String())
    }
    
    return nil
}

func (rm *ResourceManager) FreezeForEnergy(amount int64) error {
    tx, err := rm.client.FreezeBalance(
        rm.address,
        amount,
        3, // 3 days
        core.ResourceCode_ENERGY,
        "",
    )
    if err != nil {
        return err
    }
    
    signedTx, _ := transaction.SignTransaction(tx.Transaction, rm.privateKey)
    result, err := rm.client.Broadcast(signedTx)
    if err != nil {
        return err
    }
    
    fmt.Printf("Frozen %d sun for energy: %x\n", amount, result.Txid)
    return nil
}

func (rm *ResourceManager) DelegateEnergy(to string, amount int64) error {
    toAddr, err := address.Base58ToAddress(to)
    if err != nil {
        return err
    }
    
    tx, err := rm.client.DelegateResource(
        rm.address,
        toAddr,
        amount,
        core.ResourceCode_ENERGY,
        false, // not locked
    )
    if err != nil {
        return err
    }
    
    signedTx, _ := transaction.SignTransaction(tx.Transaction, rm.privateKey)
    result, err := rm.client.Broadcast(signedTx)
    if err != nil {
        return err
    }
    
    fmt.Printf("Delegated %d energy to %s: %x\n", amount, to, result.Txid)
    return nil
}

func (rm *ResourceManager) OptimizeResources() error {
    resources, err := rm.client.GetAccountResource(rm.address)
    if err != nil {
        return err
    }
    
    account, err := rm.client.GetAccount(rm.address)
    if err != nil {
        return err
    }
    
    // Calculate energy usage rate
    energyUsageRate := float64(resources.EnergyUsed) / float64(resources.EnergyLimit)
    
    // If energy usage is high and we have TRX, freeze more
    if energyUsageRate > 0.8 && account.Balance > 10000000000 { // > 10k TRX
        freezeAmount := int64(5000000000) // 5k TRX
        err := rm.FreezeForEnergy(freezeAmount)
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

## Full Applications

### Simple Wallet Application

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
    
    "github.com/fbsobreira/gotron-sdk/pkg/client"
    "github.com/fbsobreira/gotron-sdk/pkg/address"
    "github.com/fbsobreira/gotron-sdk/pkg/keys"
    "github.com/fbsobreira/gotron-sdk/pkg/keystore"
)

type Wallet struct {
    client   *client.GrpcClient
    keystore *keystore.KeyStore
    scanner  *bufio.Scanner
}

func NewWallet() (*Wallet, error) {
    client := client.NewGrpcClient("grpc.trongrid.io:50051")
    err := client.Start()
    if err != nil {
        return nil, err
    }
    
    ks := keystore.NewKeyStore("./wallet", keystore.StandardScryptN, keystore.StandardScryptP)
    
    return &Wallet{
        client:   client,
        keystore: ks,
        scanner:  bufio.NewScanner(os.Stdin),
    }, nil
}

func (w *Wallet) Run() {
    fmt.Println("Welcome to TRON Wallet!")
    fmt.Println("Type 'help' for available commands")
    
    for {
        fmt.Print("> ")
        w.scanner.Scan()
        command := strings.TrimSpace(w.scanner.Text())
        
        parts := strings.Split(command, " ")
        if len(parts) == 0 {
            continue
        }
        
        switch parts[0] {
        case "help":
            w.printHelp()
        case "create":
            w.createAccount()
        case "import":
            if len(parts) < 2 {
                fmt.Println("Usage: import <private-key>")
                continue
            }
            w.importAccount(parts[1])
        case "list":
            w.listAccounts()
        case "balance":
            if len(parts) < 2 {
                fmt.Println("Usage: balance <address>")
                continue
            }
            w.getBalance(parts[1])
        case "send":
            if len(parts) < 4 {
                fmt.Println("Usage: send <from> <to> <amount>")
                continue
            }
            amount, _ := strconv.ParseFloat(parts[3], 64)
            w.sendTRX(parts[1], parts[2], amount)
        case "exit":
            fmt.Println("Goodbye!")
            return
        default:
            fmt.Println("Unknown command. Type 'help' for available commands")
        }
    }
}

func (w *Wallet) printHelp() {
    fmt.Println(`
Available commands:
  create              - Create new account
  import <key>        - Import private key
  list                - List all accounts
  balance <address>   - Check account balance
  send <from> <to> <amount> - Send TRX
  exit                - Exit wallet
`)
}

func (w *Wallet) createAccount() {
    fmt.Print("Enter password: ")
    w.scanner.Scan()
    password := w.scanner.Text()
    
    account, err := w.keystore.NewAccount(password)
    if err != nil {
        fmt.Printf("Error creating account: %v\n", err)
        return
    }
    
    fmt.Printf("Created account: %s\n", account.Address.String())
}

func (w *Wallet) importAccount(privateKeyHex string) {
    privateKey, err := keys.GetPrivateKeyByHexString(privateKeyHex)
    if err != nil {
        fmt.Printf("Invalid private key: %v\n", err)
        return
    }
    
    fmt.Print("Enter password: ")
    w.scanner.Scan()
    password := w.scanner.Text()
    
    account, err := w.keystore.ImportECDSA(privateKey, password)
    if err != nil {
        fmt.Printf("Error importing account: %v\n", err)
        return
    }
    
    fmt.Printf("Imported account: %s\n", account.Address.String())
}

func (w *Wallet) listAccounts() {
    accounts := w.keystore.Accounts()
    if len(accounts) == 0 {
        fmt.Println("No accounts found")
        return
    }
    
    fmt.Println("Accounts:")
    for i, account := range accounts {
        balance := w.getAccountBalance(account.Address.String())
        fmt.Printf("%d. %s (%.6f TRX)\n", i+1, account.Address.String(), balance)
    }
}

func (w *Wallet) getBalance(addr string) {
    balance := w.getAccountBalance(addr)
    fmt.Printf("Balance: %.6f TRX\n", balance)
}

func (w *Wallet) getAccountBalance(addr string) float64 {
    tronAddr, err := address.Base58ToAddress(addr)
    if err != nil {
        return 0
    }
    
    account, err := w.client.GetAccount(tronAddr)
    if err != nil {
        return 0
    }
    
    return float64(account.Balance) / 1e6
}

func (w *Wallet) sendTRX(from, to string, amount float64) {
    fromAddr, err := address.Base58ToAddress(from)
    if err != nil {
        fmt.Printf("Invalid from address: %v\n", err)
        return
    }
    
    toAddr, err := address.Base58ToAddress(to)
    if err != nil {
        fmt.Printf("Invalid to address: %v\n", err)
        return
    }
    
    // Find account in keystore
    var account *keystore.Account
    for _, acc := range w.keystore.Accounts() {
        if acc.Address == fromAddr {
            account = &acc
            break
        }
    }
    
    if account == nil {
        fmt.Println("Account not found in keystore")
        return
    }
    
    fmt.Print("Enter password: ")
    w.scanner.Scan()
    password := w.scanner.Text()
    
    err = w.keystore.Unlock(*account, password)
    if err != nil {
        fmt.Printf("Failed to unlock account: %v\n", err)
        return
    }
    defer w.keystore.Lock(account.Address)
    
    // Create and sign transaction
    amountSun := int64(amount * 1e6)
    tx, err := w.client.Transfer(fromAddr, toAddr, amountSun)
    if err != nil {
        fmt.Printf("Failed to create transaction: %v\n", err)
        return
    }
    
    signedTx, err := w.keystore.SignTx(*account, tx.Transaction, nil)
    if err != nil {
        fmt.Printf("Failed to sign transaction: %v\n", err)
        return
    }
    
    result, err := w.client.Broadcast(signedTx)
    if err != nil {
        fmt.Printf("Failed to broadcast transaction: %v\n", err)
        return
    }
    
    fmt.Printf("Transaction sent: %x\n", result.Txid)
}

func main() {
    wallet, err := NewWallet()
    if err != nil {
        log.Fatal(err)
    }
    defer wallet.client.Stop()
    
    wallet.Run()
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