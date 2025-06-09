# API Reference

Complete API reference for the Gotron SDK packages.

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
func (c *GrpcClient) Start() error
func (c *GrpcClient) Stop()
func (c *GrpcClient) SetAPIKey(apiKey string)
```

##### Account Methods

```go
// Get account information
func (c *GrpcClient) GetAccount(address common.Address) (*core.Account, error)

// Get account resources
func (c *GrpcClient) GetAccountResource(address common.Address) (*api.AccountResourceMessage, error)

// Get account net usage
func (c *GrpcClient) GetAccountNet(address common.Address) (*api.AccountNetMessage, error)
```

##### Transaction Methods

```go
// Create transfer transaction
func (c *GrpcClient) Transfer(from, to common.Address, amount int64) (*api.TransactionExtention, error)

// Create transfer with memo
func (c *GrpcClient) TransferWithMemo(from, to common.Address, amount int64, memo string) (*api.TransactionExtention, error)

// Broadcast transaction
func (c *GrpcClient) Broadcast(transaction *core.Transaction) (*api.Return, error)

// Get transaction by ID
func (c *GrpcClient) GetTransactionByID(id string) (*core.Transaction, error)

// Get transaction info by ID
func (c *GrpcClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error)
```

##### Block Methods

```go
// Get current block
func (c *GrpcClient) GetNowBlock() (*core.Block, error)

// Get block by number
func (c *GrpcClient) GetBlockByNum(num int64) (*core.Block, error)

// Get block by ID
func (c *GrpcClient) GetBlockByID(id string) (*core.Block, error)

// Get block by latest number
func (c *GrpcClient) GetBlockByLatestNum(num int64) (*api.BlockList, error)

// Get block by limit next
func (c *GrpcClient) GetBlockByLimitNext(start, end int64) (*api.BlockList, error)
```

##### Smart Contract Methods

```go
// Deploy contract
func (c *GrpcClient) DeployContract(
    owner common.Address,
    name string,
    bytecode []byte,
    feeLimit int64,
    consumeUserResourcePercent int64,
    originEnergyLimit int64,
) (*api.TransactionExtention, error)

// Trigger smart contract
func (c *GrpcClient) TriggerContract(
    owner, contract common.Address,
    data []byte,
    feeLimit int64,
    callValue int64,
    tokenID string,
) (*api.TransactionExtention, error)

// Trigger constant contract (call)
func (c *GrpcClient) TriggerConstantContract(
    owner, contract common.Address,
    data []byte,
    callValue int64,
) (*api.TransactionExtention, error)

// Get contract
func (c *GrpcClient) GetContract(address common.Address) (*core.SmartContract, error)

// Get contract info
func (c *GrpcClient) GetContractInfo(address common.Address) (*core.SmartContractDataWrapper, error)
```

##### Resource Management

```go
// Freeze balance
func (c *GrpcClient) FreezeBalance(
    owner common.Address,
    amount int64,
    duration int64,
    resource core.ResourceCode,
    receiver string,
) (*api.TransactionExtention, error)

// Unfreeze balance
func (c *GrpcClient) UnfreezeBalance(
    owner common.Address,
    resource core.ResourceCode,
    receiver string,
) (*api.TransactionExtention, error)

// Delegate resource
func (c *GrpcClient) DelegateResource(
    owner common.Address,
    receiver common.Address,
    balance int64,
    resource core.ResourceCode,
    lock bool,
) (*api.TransactionExtention, error)

// Undelegate resource
func (c *GrpcClient) UnDelegateResource(
    owner common.Address,
    receiver common.Address,
    balance int64,
    resource core.ResourceCode,
) (*api.TransactionExtention, error)
```

##### Witness Methods

```go
// List witnesses
func (c *GrpcClient) ListWitnesses() (*api.WitnessList, error)

// Create witness
func (c *GrpcClient) CreateWitness(owner common.Address, url string) (*api.TransactionExtention, error)

// Update witness
func (c *GrpcClient) UpdateWitness(owner common.Address, url string) (*api.TransactionExtention, error)

// Vote witness
func (c *GrpcClient) VoteWitnessAccount(
    owner common.Address,
    votes map[string]int64,
) (*api.TransactionExtention, error)

// Get brokerage info
func (c *GrpcClient) GetBrokerageInfo(address common.Address) (*api.NumberMessage, error)

// Update brokerage
func (c *GrpcClient) UpdateBrokerage(owner common.Address, brokerage int32) (*api.TransactionExtention, error)
```

##### TRC10 Token Methods

```go
// Create asset issue
func (c *GrpcClient) CreateAssetIssue(
    owner common.Address,
    name string,
    abbr string,
    totalSupply int64,
    trxNum int32,
    num int32,
    startTime int64,
    endTime int64,
    description string,
    url string,
    freeAssetNetLimit int64,
    publicFreeAssetNetLimit int64,
    precision int32,
    fronzenSupply map[int64]int64,
) (*api.TransactionExtention, error)

// Transfer asset
func (c *GrpcClient) TransferAsset(
    from, to common.Address,
    assetID string,
    amount int64,
) (*api.TransactionExtention, error)

// Participate asset issue
func (c *GrpcClient) ParticipateAssetIssue(
    from, to common.Address,
    assetID string,
    amount int64,
) (*api.TransactionExtention, error)

// Get asset issue by account
func (c *GrpcClient) GetAssetIssueByAccount(address common.Address) (*api.AssetIssueList, error)

// Get asset issue by ID
func (c *GrpcClient) GetAssetIssueByID(id string) (*core.AssetIssueContract, error)

// Get asset issue list
func (c *GrpcClient) GetAssetIssueList() (*api.AssetIssueList, error)

// Get paginated asset issue list
func (c *GrpcClient) GetPaginatedAssetIssueList(offset, limit int64) (*api.AssetIssueList, error)
```

##### Proposal Methods

```go
// List proposals
func (c *GrpcClient) ListProposals() (*api.ProposalList, error)

// Get proposal by ID
func (c *GrpcClient) GetProposalByID(id int64) (*core.Proposal, error)

// Create proposal
func (c *GrpcClient) CreateProposal(owner common.Address, parameters map[int64]int64) (*api.TransactionExtention, error)

// Approve proposal
func (c *GrpcClient) ApproveProposal(owner common.Address, id int64, approval bool) (*api.TransactionExtention, error)

// Delete proposal
func (c *GrpcClient) DeleteProposal(owner common.Address, id int64) (*api.TransactionExtention, error)
```

##### Exchange Methods

```go
// List exchanges
func (c *GrpcClient) ListExchanges() (*api.ExchangeList, error)

// Get exchange by ID
func (c *GrpcClient) GetExchangeByID(id int64) (*core.Exchange, error)

// Create exchange
func (c *GrpcClient) CreateExchange(
    owner common.Address,
    firstTokenID string,
    firstTokenBalance int64,
    secondTokenID string,
    secondTokenBalance int64,
) (*api.TransactionExtention, error)

// Inject exchange
func (c *GrpcClient) InjectExchange(
    owner common.Address,
    exchangeID int64,
    tokenID string,
    quant int64,
) (*api.TransactionExtention, error)

// Withdraw exchange
func (c *GrpcClient) WithdrawExchange(
    owner common.Address,
    exchangeID int64,
    tokenID string,
    quant int64,
) (*api.TransactionExtention, error)

// Transaction with exchange
func (c *GrpcClient) TransactionWithExchange(
    owner common.Address,
    exchangeID int64,
    tokenID string,
    quant int64,
    expected int64,
) (*api.TransactionExtention, error)
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

// Convert base58 string to address
func Base58ToAddress(s string) (Address, error)

// Convert hex string to address
func HexToAddress(s string) (Address, error)

// Validate address
func IsValid(addr string) bool

// Get address from private key
func GetAddressFromPrivateKey(privateKey string) (string, error)
```

#### Methods

```go
// Convert to string (base58)
func (a Address) String() string

// Convert to hex
func (a Address) Hex() string

// Convert to bytes
func (a Address) Bytes() []byte

// Check if zero address
func (a Address) IsZero() bool
```

## Transaction Package

### `github.com/fbsobreira/gotron-sdk/pkg/client/transaction`

The transaction package handles transaction signing and management.

#### Controller

```go
type Controller struct {
    client          *client.GrpcClient
    tx              *core.Transaction
    rawData         *core.TransactionRaw
    privateKey      *ecdsa.PrivateKey
    signatureList   [][]byte
}
```

##### Methods

```go
// Create controller
func NewController(client *client.GrpcClient, tx *core.Transaction, privateKey *ecdsa.PrivateKey) *Controller

// Sign transaction
func (c *Controller) Sign() error

// Add signature
func (c *Controller) AddSignature(privateKey *ecdsa.PrivateKey) error

// Build transaction
func (c *Controller) Build() (*core.Transaction, error)

// Broadcast transaction
func (c *Controller) Broadcast() (*api.Return, error)
```

#### Signer Functions

```go
// Sign transaction
func SignTransaction(transaction *core.Transaction, privateKey *ecdsa.PrivateKey) (*core.Transaction, error)

// Sign message
func SignMessage(message []byte, privateKey *ecdsa.PrivateKey) ([]byte, error)

// Verify signature
func VerifySignature(message, signature []byte, address string) bool
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
func (ks *KeyStore) HasAddress(addr common.Address) bool

// Unlock account
func (ks *KeyStore) Unlock(a Account, passphrase string) error

// Lock account
func (ks *KeyStore) Lock(addr common.Address) error

// Sign transaction
func (ks *KeyStore) SignTx(a Account, tx *core.Transaction, chainID *big.Int) (*core.Transaction, error)

// Sign message
func (ks *KeyStore) SignMessage(a Account, message []byte) ([]byte, error)
```

#### Account

```go
type Account struct {
    Address common.Address
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