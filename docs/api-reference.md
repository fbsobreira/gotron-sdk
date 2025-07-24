# API Reference

Complete API reference for the GoTRON SDK packages.

## Table of Contents

- [Client Package](#client-package)
- [Address Package](#address-package)
- [Transaction Package](#transaction-package)
- [Keystore Package](#keystore-package)
- [ABI Package](#abi-package)
- [Common Package](#common-package)
- [Proto Package](#proto-package)

## Client Package

### `github.com/fbsobreira/gotron-sdk/pkg/client`

The client package provides the main interface for interacting with TRON nodes.

#### GrpcClient

```go
type GrpcClient struct {
    Address string
    Conn    *grpc.ClientConn
    Client  api.WalletClient
}
```

##### Constructor

```go
func NewGrpcClient(address string) *GrpcClient
func NewGrpcClientWithTimeout(address string, timeout int) *GrpcClient
```

##### Connection Methods

```go
func (g *GrpcClient) Start(opts ...grpc.DialOption) error
func (g *GrpcClient) Stop()
func (g *GrpcClient) SetAPIKey(apiKey string) error
```

##### Account Methods

```go
// Get account information
func (g *GrpcClient) GetAccount(addr string) (*core.Account, error)

// Get account resources
func (g *GrpcClient) GetAccountResource(addr string) (*api.AccountResourceMessage, error)

// Get account net usage
func (g *GrpcClient) GetAccountNet(addr string) (*api.AccountNetMessage, error)
```

##### Transaction Methods

```go
// Create transfer transaction
func (g *GrpcClient) Transfer(from, toAddress string, amount int64) (*api.TransactionExtention, error)

// Broadcast transaction
func (g *GrpcClient) Broadcast(tx *core.Transaction) (*api.Return, error)

// Get transaction by ID
func (g *GrpcClient) GetTransactionByID(id string) (*core.Transaction, error)

// Get transaction info by ID
func (g *GrpcClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error)
```

##### Block Methods

```go
// Get current block
func (g *GrpcClient) GetNowBlock() (*api.BlockExtention, error)

// Get block by number
func (g *GrpcClient) GetBlockByNum(num int64) (*api.BlockExtention, error)

// Get block by ID
func (g *GrpcClient) GetBlockByID(id string) (*core.Block, error)

// Get block by latest number
func (g *GrpcClient) GetBlockByLatestNum(num int64) (*api.BlockListExtention, error)

// Get block by limit next
func (g *GrpcClient) GetBlockByLimitNext(start, end int64) (*api.BlockListExtention, error)
```

##### Smart Contract Methods

```go
// Deploy contract
func (g *GrpcClient) DeployContract(
    from, contractName string,
    abi *core.SmartContract_ABI,
    codeStr string,
    feeLimit, curPercent, oeLimit int64,
) (*api.TransactionExtention, error)

// Trigger smart contract
func (g *GrpcClient) TriggerContract(
    from, contractAddress, method, jsonString string,
    feeLimit, tAmount int64,
    tTokenID string,
    tTokenAmount int64,
) (*api.TransactionExtention, error)

// Trigger constant contract (call)
func (g *GrpcClient) TriggerConstantContract(
    from, contractAddress, method, jsonString string,
) (*api.TransactionExtention, error)

// Get contract ABI
func (g *GrpcClient) GetContractABI(contractAddress string) (*core.SmartContract_ABI, error)
```

##### Resource Management

```go
// Freeze balance V2
func (g *GrpcClient) FreezeBalanceV2(
    from string,
    resource core.ResourceCode,
    frozenBalance int64,
) (*api.TransactionExtention, error)

// Unfreeze balance V2
func (g *GrpcClient) UnfreezeBalanceV2(
    from string,
    resource core.ResourceCode,
    unfreezeBalance int64,
) (*api.TransactionExtention, error)

// Delegate resource
func (g *GrpcClient) DelegateResource(
    from, to string,
    resource core.ResourceCode,
    delegateBalance int64,
    lock bool,
    lockPeriod int64,
) (*api.TransactionExtention, error)

// Undelegate resource
func (g *GrpcClient) UnDelegateResource(
    owner, receiver string,
    resource core.ResourceCode,
    delegateBalance int64,
) (*api.TransactionExtention, error)

// Freeze balance V1 (deprecated, use V2)
func (g *GrpcClient) FreezeBalance(
    from, delegateTo string,
    resource core.ResourceCode,
    frozenBalance int64,
) (*api.TransactionExtention, error)

// Unfreeze balance V1 (deprecated, use V2)
func (g *GrpcClient) UnfreezeBalance(
    from, delegateTo string,
    resource core.ResourceCode,
) (*api.TransactionExtention, error)
```

##### Witness Methods

```go
// List witnesses
func (g *GrpcClient) ListWitnesses() (*api.WitnessList, error)

// Create witness
func (g *GrpcClient) CreateWitness(from, urlStr string) (*api.TransactionExtention, error)

// Update witness
func (g *GrpcClient) UpdateWitness(from, urlStr string) (*api.TransactionExtention, error)

// Vote witness
func (g *GrpcClient) VoteWitnessAccount(
    from string,
    witnessMap map[string]int64,
) (*api.TransactionExtention, error)

// Get witness brokerage
func (g *GrpcClient) GetWitnessBrokerage(witness string) (float64, error)

// Update brokerage
func (g *GrpcClient) UpdateBrokerage(from string, commission int32) (*api.TransactionExtention, error)
```

##### TRC10 Token Methods

```go
// Create asset issue
func (g *GrpcClient) AssetIssue(
    from, name, description, abbr, urlStr string,
    precision int32,
    totalSupply, startTime, endTime, FreeAssetNetLimit, PublicFreeAssetNetLimit int64,
    trxNum, icoNum, voteScore int32,
    frozenSupply map[string]string,
) (*api.TransactionExtention, error)

// Transfer asset
func (g *GrpcClient) TransferAsset(
    from, toAddress, assetName string,
    amount int64,
) (*api.TransactionExtention, error)

// Participate asset issue
func (g *GrpcClient) ParticipateAssetIssue(
    from, issuerAddress, tokenID string,
    amount int64,
) (*api.TransactionExtention, error)

// Get asset issue by account
func (g *GrpcClient) GetAssetIssueByAccount(address string) (*api.AssetIssueList, error)

// Get asset issue by ID
func (g *GrpcClient) GetAssetIssueByID(tokenID string) (*core.AssetIssueContract, error)

// Get asset issue list
func (g *GrpcClient) GetAssetIssueList(page int64, limit ...int) (*api.AssetIssueList, error)
```

##### Proposal Methods

```go
// List proposals
func (g *GrpcClient) ProposalsList() (*api.ProposalList, error)

// Create proposal
func (g *GrpcClient) ProposalCreate(from string, parameters map[int64]int64) (*api.TransactionExtention, error)

// Approve proposal
func (g *GrpcClient) ProposalApprove(from string, id int64, confirm bool) (*api.TransactionExtention, error)

// Withdraw proposal
func (g *GrpcClient) ProposalWithdraw(from string, id int64) (*api.TransactionExtention, error)
```

##### Exchange Methods

```go
// List exchanges
func (g *GrpcClient) ExchangeList(page int64, limit ...int) (*api.ExchangeList, error)

// Get exchange by ID
func (g *GrpcClient) ExchangeByID(id int64) (*core.Exchange, error)

// Create exchange
func (g *GrpcClient) ExchangeCreate(
    from string,
    tokenID, tokenQuant int64,
    secondTokenID, secondTokenQuant int64,
) (*api.TransactionExtention, error)

// Inject exchange
func (g *GrpcClient) ExchangeInject(
    from string,
    exchangeID int64,
    tokenID, tokenQuant int64,
) (*api.TransactionExtention, error)

// Withdraw exchange
func (g *GrpcClient) ExchangeWithdraw(
    from string,
    exchangeID int64,
    tokenID, tokenQuant int64,
) (*api.TransactionExtention, error)

// Trade with exchange
func (g *GrpcClient) ExchangeTrade(
    from string,
    exchangeID int64,
    tokenID, tokenQuant, expected int64,
) (*api.TransactionExtention, error)
```

##### TRC20 Token Methods

```go
// General TRC20 contract call
func (g *GrpcClient) TRC20Call(
    from, contractAddress, data string,
    constant bool,
    feeLimit int64,
) (*api.TransactionExtention, error)

// Get token name
func (g *GrpcClient) TRC20GetName(contractAddress string) (string, error)

// Get token symbol
func (g *GrpcClient) TRC20GetSymbol(contractAddress string) (string, error)

// Get token decimals
func (g *GrpcClient) TRC20GetDecimals(contractAddress string) (*big.Int, error)

// Parse numeric property from TRC20 response
func (g *GrpcClient) ParseTRC20NumericProperty(data string) (*big.Int, error)

// Parse string property from TRC20 response
func (g *GrpcClient) ParseTRC20StringProperty(data string) (string, error)

// Get token balance for address
func (g *GrpcClient) TRC20ContractBalance(addr, contractAddress string) (*big.Int, error)

// Send TRC20 tokens
func (g *GrpcClient) TRC20Send(
    from, to, contract string,
    amount *big.Int,
    feeLimit int64,
) (*api.TransactionExtention, error)

// Transfer TRC20 tokens on behalf of owner
func (g *GrpcClient) TRC20TransferFrom(
    owner, from, to, contract string,
    amount *big.Int,
    feeLimit int64,
) (*api.TransactionExtention, error)

// Approve TRC20 token spending
func (g *GrpcClient) TRC20Approve(
    from, to, contract string,
    amount *big.Int,
    feeLimit int64,
) (*api.TransactionExtention, error)
```

##### Network Information Methods

```go
// List all nodes
func (g *GrpcClient) ListNodes() (*api.NodeList, error)

// Get next maintenance time
func (g *GrpcClient) GetNextMaintenanceTime() (*api.NumberMessage, error)

// Get total transaction count
func (g *GrpcClient) TotalTransaction() (*api.NumberMessage, error)

// Get node information
func (g *GrpcClient) GetNodeInfo() (*core.NodeInfo, error)

// Get energy prices
func (g *GrpcClient) GetEnergyPrices() (*api.PricesResponseMessage, error)

// Get bandwidth prices
func (g *GrpcClient) GetBandwidthPrices() (*api.PricesResponseMessage, error)

// Get memo fee
func (g *GrpcClient) GetMemoFee() (*api.PricesResponseMessage, error)
```

##### Additional Account Methods

```go
// Get rewards information
func (g *GrpcClient) GetRewardsInfo(addr string) (int64, error)

// Create new account
func (g *GrpcClient) CreateAccount(from, addr string) (*api.TransactionExtention, error)

// Get detailed account information
func (g *GrpcClient) GetAccountDetailed(addr string) (*account.Account, error)

// Withdraw balance (claim rewards)
func (g *GrpcClient) WithdrawBalance(from string) (*api.TransactionExtention, error)

// Update account permissions
func (g *GrpcClient) UpdateAccountPermission(
    from string,
    owner, witness map[string]interface{},
    actives []map[string]interface{},
) (*api.TransactionExtention, error)
```

##### Additional Asset Methods

```go
// Get asset issue by name
func (g *GrpcClient) GetAssetIssueByName(name string) (*core.AssetIssueContract, error)

// Update asset issue
func (g *GrpcClient) UpdateAssetIssue(
    from, description, urlStr string,
    newLimit, newPublicLimit int64,
) (*api.TransactionExtention, error)

// Unfreeze asset
func (g *GrpcClient) UnfreezeAsset(from string) (*api.TransactionExtention, error)
```

##### Additional Resource Methods

```go
// Get delegated resources
func (g *GrpcClient) GetDelegatedResources(address string) ([]*api.DelegatedResourceList, error)

// Get delegated resources V2
func (g *GrpcClient) GetDelegatedResourcesV2(address string) ([]*api.DelegatedResourceList, error)

// Get maximum delegatable size
func (g *GrpcClient) GetCanDelegatedMaxSize(address string, resource int32) (*api.CanDelegatedMaxSizeResponseMessage, error)

// Get available unfreeze count
func (g *GrpcClient) GetAvailableUnfreezeCount(from string) (*api.GetAvailableUnfreezeCountResponseMessage, error)

// Get withdrawable unfreeze amount
func (g *GrpcClient) GetCanWithdrawUnfreezeAmount(from string, timestamp int64) (*api.CanWithdrawUnfreezeAmountResponseMessage, error)

// Withdraw expired unfreeze
func (g *GrpcClient) WithdrawExpireUnfreeze(from string, timestamp int64) (*api.TransactionExtention, error)
```

##### Additional Contract Methods

```go
// Update contract energy limit
func (g *GrpcClient) UpdateEnergyLimitContract(
    from, contractAddress string,
    value int64,
) (*api.TransactionExtention, error)

// Update contract settings
func (g *GrpcClient) UpdateSettingContract(
    from, contractAddress string,
    value int64,
) (*api.TransactionExtention, error)

// Estimate energy for contract call
func (g *GrpcClient) EstimateEnergy(
    from, contractAddress, method, jsonString string,
    tAmount int64,
    tTokenID string,
    tTokenAmount int64,
) (*api.EstimateEnergyMessage, error)

// Update transaction hash
func (g *GrpcClient) UpdateHash(tx *api.TransactionExtention) error
```

##### Additional Block Methods

```go
// Get block information by number
func (g *GrpcClient) GetBlockInfoByNum(num int64) (*api.TransactionInfoList, error)
```

##### Transaction Analysis Methods

```go
// Get transaction sign weight
func (g *GrpcClient) GetTransactionSignWeight(tx *core.Transaction) (*api.TransactionSignWeight, error)
```

##### Client Management Methods

```go
// Set client timeout
func (g *GrpcClient) SetTimeout(timeout time.Duration)

// Reconnect to a different node
func (g *GrpcClient) Reconnect(url string) error
```

## Address Package

### `github.com/fbsobreira/gotron-sdk/pkg/address`

The address package handles TRON address encoding and validation.

#### Types

```go
type Address [AddressLength]byte
```

#### Constants

```go
const (
    AddressLength       = 21
    AddressLengthBase58 = 34
    TronBytePrefix      = byte(0x41)
)
```

#### Functions

```go
// Convert public key to address
func PubkeyToAddress(p ecdsa.PublicKey) Address

// Convert BTCEC public key to address
func BTCECPubkeyToAddress(p *btcec.PublicKey) Address

// Convert BTCEC private key to address
func BTCECPrivkeyToAddress(p *btcec.PrivateKey) Address

// Convert base58 string to address
func Base58ToAddress(s string) (Address, error)

// Convert base64 string to address
func Base64ToAddress(s string) (Address, error)

// Convert hex string to address
func HexToAddress(s string) Address

// Convert big.Int to address
func BigToAddress(b *big.Int) Address
```

#### Methods

```go
// Convert to string (base58)
func (a Address) String() string

// Convert to hex
func (a Address) Hex() string

// Convert to bytes
func (a Address) Bytes() []byte

// Check if valid TRON address
func (a Address) IsValid() bool

// Database driver value interface
func (a Address) Value() (driver.Value, error)
```

## Transaction Package

### `github.com/fbsobreira/gotron-sdk/pkg/client/transaction`

The transaction package handles transaction signing and management.

#### Controller

```go
type Controller struct {
    executionError error
    resultError    error
    client         *client.GrpcClient
    tx             *core.Transaction
    sender         sender
    Behavior       behavior
    Result         *api.Return
    Receipt        *core.TransactionInfo
}
```

##### Methods

```go
// Create controller with options
func NewController(
    client *client.GrpcClient,
    senderKs *keystore.KeyStore,
    senderAcct *keystore.Account,
    tx *core.Transaction,
    options ...func(*Controller),
) *Controller

// Execute transaction (sign and broadcast)
func (C *Controller) ExecuteTransaction() error

// Get transaction hash
func (C *Controller) TransactionHash() (string, error)

// Get raw data bytes from transaction
func (C *Controller) GetRawData() ([]byte, error)

// Get result error
func (C *Controller) GetResultError() error
```

#### Transaction Signing Functions

```go
// Sign transaction with BTCEC private key
func SignTransaction(tx *core.Transaction, signer *btcec.PrivateKey) (*core.Transaction, error)

// Sign transaction with ECDSA private key
func SignTransactionECDSA(tx *core.Transaction, signer *ecdsa.PrivateKey) (*core.Transaction, error)
```

## Keystore Package

### `github.com/fbsobreira/gotron-sdk/pkg/keystore`

The keystore package provides secure key storage and management.

#### KeyStore

```go
type KeyStore struct {
    storage  Storage
    cache    *accountCache
    scrypt   scryptParams
    isLocked bool
    mu       sync.RWMutex
}
```

##### Constructor

```go
func NewKeyStore(keydir string, scryptN, scryptP int) *KeyStore
```

##### Methods

```go
// Create new account
func (ks *KeyStore) NewAccount(passphrase string) (Account, error)

// Import ECDSA private key
func (ks *KeyStore) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (Account, error)

// Import preexisting keyfile
func (ks *KeyStore) Import(keyJSON []byte, passphrase, newPassphrase string) (Account, error)

// Export account
func (ks *KeyStore) Export(a Account, passphrase, newPassphrase string) ([]byte, error)

// Delete account
func (ks *KeyStore) Delete(a Account, passphrase string) error

// Update account
func (ks *KeyStore) Update(a Account, passphrase, newPassphrase string) error

// List all accounts
func (ks *KeyStore) Accounts() []Account

// Check if account exists
func (ks *KeyStore) HasAddress(addr address.Address) bool

// Unlock account
func (ks *KeyStore) Unlock(a Account, passphrase string) error

// Lock account
func (ks *KeyStore) Lock(addr address.Address) error

// Sign transaction
func (ks *KeyStore) SignTx(a Account, tx *core.Transaction) (*core.Transaction, error)

// Sign transaction with passphrase
func (ks *KeyStore) SignTxWithPassphrase(a Account, passphrase string, tx *core.Transaction) (*core.Transaction, error)

// Sign hash
func (ks *KeyStore) SignHash(a Account, hash []byte) ([]byte, error)

// Sign hash with passphrase
func (ks *KeyStore) SignHashWithPassphrase(a Account, passphrase string, hash []byte) ([]byte, error)

// Get wallets
func (ks *KeyStore) Wallets() []Wallet

// Subscribe to wallet events
func (ks *KeyStore) Subscribe(sink chan<- WalletEvent) event.Subscription

// Get decrypted key
func (ks *KeyStore) GetDecryptedKey(a Account, auth string) (Account, *Key, error)

// Export account to JSON
func (ks *KeyStore) Export(a Account, passphrase, newPassphrase string) (keyJSON []byte, err error)
```

#### Account

```go
type Account struct {
    Address address.Address
    URL     URL
}
```

## ABI Package

### `github.com/fbsobreira/gotron-sdk/pkg/abi`

The ABI package handles smart contract ABI encoding and decoding.

#### ABI

```go
type ABI struct {
    Constructor Method
    Methods     map[string]Method
    Events      map[string]Event
}
```

##### Functions

```go
// Parse ABI from JSON
func JSON(reader io.Reader) (ABI, error)

// Pack method call
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error)

// Unpack method return
func (abi ABI) Unpack(v interface{}, name string, data []byte) error

// Unpack event
func (abi ABI) UnpackEvent(v interface{}, name string, data []byte) error
```

#### Method

```go
type Method struct {
    Name    string
    Const   bool
    Inputs  Arguments
    Outputs Arguments
}
```

#### Event

```go
type Event struct {
    Name      string
    Anonymous bool
    Inputs    Arguments
}
```

## Common Package

### `github.com/fbsobreira/gotron-sdk/pkg/common`

The common package provides utility functions and types.

#### Functions

```go
// Bytes to hex string
func BytesToHex(bytes []byte) string

// Hex string to bytes
func HexToBytes(hex string) []byte

// Left pad bytes
func LeftPadBytes(slice []byte, l int) []byte

// Right pad bytes
func RightPadBytes(slice []byte, l int) []byte

// Convert to hex
func ToHex(b []byte) string

// From hex
func FromHex(s string) []byte

// Has hex prefix
func HasHexPrefix(str string) bool

// Is hex
func IsHex(str string) bool

// Big int to bytes
func BigToBytes(num *big.Int) []byte

// Bytes to big int
func BytesToBig(bytes []byte) *big.Int
```

#### Hash Functions

```go
// Keccak256 hash
func Keccak256(data ...[]byte) []byte

// SHA256 hash
func SHA256(data ...[]byte) []byte

// Hash message for signing
func HashMessage(message []byte) []byte
```

#### Encoding Functions

```go
// Encode to base58
func EncodeBase58(input []byte) string

// Decode from base58
func DecodeBase58(input string) ([]byte, error)

// Encode check
func EncodeCheck(input []byte) string

// Decode check
func DecodeCheck(input string) ([]byte, error)
```

## Proto Package

### `github.com/fbsobreira/gotron-sdk/pkg/proto`

The proto package contains generated Protocol Buffer definitions for TRON.

#### Core Types

- `Account`
- `Transaction`
- `Block`
- `SmartContract`
- `AssetIssueContract`
- `TransferContract`
- `TransferAssetContract`
- `VoteWitnessContract`
- `WitnessCreateContract`
- `WitnessUpdateContract`
- `FreezeBalanceContract`
- `UnfreezeBalanceContract`
- `ProposalCreateContract`
- `ProposalApproveContract`
- `ProposalDeleteContract`
- `ExchangeCreateContract`
- `ExchangeInjectContract`
- `ExchangeWithdrawContract`
- `ExchangeTransactionContract`

#### API Types

- `TransactionExtention`
- `Return`
- `NodeInfo`
- `AccountResourceMessage`
- `AccountNetMessage`
- `WitnessList`
- `AssetIssueList`
- `ProposalList`
- `ExchangeList`
- `BlockList`

## Error Types

### Common Errors

```go
var (
    ErrInvalidAddress     = errors.New("invalid address")
    ErrInvalidPrivateKey  = errors.New("invalid private key")
    ErrInvalidTransaction = errors.New("invalid transaction")
    ErrInsufficientFunds  = errors.New("insufficient funds")
    ErrContractExecution  = errors.New("contract execution failed")
)
```

### Transaction Errors

```go
var (
    ErrBadTransactionParam = errors.New("bad transaction parameter")
    ErrTransactionExpired  = errors.New("transaction expired")
    ErrTransactionFailed   = errors.New("transaction failed")
)
```

### Keystore Errors

```go
var (
    ErrLocked          = errors.New("account locked")
    ErrNoMatch         = errors.New("no key for given address or file")
    ErrDecrypt         = errors.New("could not decrypt key with given passphrase")
    ErrAccountAlreadyExists = errors.New("account already exists")
)
```

## Constants

### Resource Types

```go
const (
    ResourceBandwidth core.ResourceCode = 0
    ResourceEnergy    core.ResourceCode = 1
)
```

### Contract Types

```go
const (
    AccountCreateContract       = "AccountCreateContract"
    TransferContract           = "TransferContract"
    TransferAssetContract      = "TransferAssetContract"
    VoteAssetContract         = "VoteAssetContract"
    VoteWitnessContract       = "VoteWitnessContract"
    WitnessCreateContract     = "WitnessCreateContract"
    AssetIssueContract        = "AssetIssueContract"
    WitnessUpdateContract     = "WitnessUpdateContract"
    ParticipateAssetIssueContract = "ParticipateAssetIssueContract"
    AccountUpdateContract     = "AccountUpdateContract"
    FreezeBalanceContract     = "FreezeBalanceContract"
    UnfreezeBalanceContract   = "UnfreezeBalanceContract"
    WithdrawBalanceContract   = "WithdrawBalanceContract"
    UnfreezeAssetContract     = "UnfreezeAssetContract"
    UpdateAssetContract       = "UpdateAssetContract"
    ProposalCreateContract    = "ProposalCreateContract"
    ProposalApproveContract   = "ProposalApproveContract"
    ProposalDeleteContract    = "ProposalDeleteContract"
    SetAccountIdContract      = "SetAccountIdContract"
    CustomContract           = "CustomContract"
    CreateSmartContract      = "CreateSmartContract"
    TriggerSmartContract     = "TriggerSmartContract"
    ExchangeCreateContract   = "ExchangeCreateContract"
    ExchangeInjectContract   = "ExchangeInjectContract"
    ExchangeWithdrawContract = "ExchangeWithdrawContract"
    ExchangeTransactionContract = "ExchangeTransactionContract"
    UpdateEnergyLimitContract = "UpdateEnergyLimitContract"
    AccountPermissionUpdateContract = "AccountPermissionUpdateContract"
)
```