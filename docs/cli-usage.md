# CLI Usage Guide

This guide provides comprehensive documentation for the `tronctl` command-line interface.

## Table of Contents

- [Global Options](#global-options)
- [Account Commands](#account-commands)
- [Key Management](#key-management)
- [Blockchain Commands](#blockchain-commands)
- [Smart Contract Commands](#smart-contract-commands)
- [TRC10 Token Commands](#trc10-token-commands)
- [TRC20 Token Commands](#trc20-token-commands)
- [Super Representative Commands](#super-representative-commands)
- [Proposal Commands](#proposal-commands)
- [Exchange Commands](#exchange-commands)
- [Configuration](#configuration)
- [Utility Commands](#utility-commands)
- [Examples](#examples)

## Global Options

These options can be used with any command:

```
--node <address>           TRON node address (default: grpc.trongrid.io:50051)
--apiKey <key>            Trongrid API key
--withTLS                 Use TLS connection
--timeout <duration>      Request timeout (default: 60s)
--verbose                 Enable verbose output
--config <path>          Config file path (default: ~/.tronctl/config.yaml)
```

## Account Commands

### Get Account Balance

```bash
tronctl account balance <address>

# Example
tronctl account balance TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n
```

### Send TRX

```bash
tronctl account send <to-address> <amount>

# Options
--signer <name>          Signer account name (required)

# Example
tronctl account send TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 100.5 --signer myaccount
```

### Freeze Resources

```bash
tronctl account freeze <amount> <resource>

# Parameters
resource: BANDWIDTH or ENERGY

# Options
--signer <name>          Account to freeze from (required)
--days <number>          Days to freeze (default: 3)

# Example
tronctl account freeze 1000 ENERGY --signer myaccount
```

### Unfreeze Resources

```bash
tronctl account unfreeze <resource>

# Example
tronctl account unfreeze BANDWIDTH
```

### Vote for Witnesses

```bash
tronctl account vote <witness-address> <votes>

# Options
--signer <name>          Voter account name (required)

# Example
tronctl account vote TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH 100000 --signer myaccount
```

### Get Account Resources

```bash
tronctl account resources <address>

# Example
tronctl account resources TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n
```

### Update Account Permissions

```bash
tronctl account permission <address>

# Options
--owner <keys>           Owner permission keys (comma-separated)
--active <keys>          Active permission keys (comma-separated)
--witness <key>          Witness permission key

# Example
tronctl account permission TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n --active TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9,TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH
```

### Sign Message

```bash
tronctl account sign <message>

# Options
--signer <name>          Signer account name (required)

# Example
tronctl account sign "Hello TRON" --signer myaccount
```

## Key Management

### Create New Account

```bash
tronctl keys add <account-name>

# Options
--passphrase             Use passphrase encryption

# Example
tronctl keys add myaccount
```

### Import Private Key

```bash
tronctl keys import <private-key>

# Options
--passphrase             Use passphrase encryption

# Example
tronctl keys import 8f5c7e1a2b3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f
```

### Recover from Mnemonic

```bash
tronctl keys recover-from-mnemonic <account-name>

# Interactive prompt will ask for:
# - Mnemonic phrase
# - Mnemonic password (optional)

# Example
tronctl keys recover-from-mnemonic myaccount
```

### Export Private Key

```bash
tronctl keys export <account-name>

# Example
tronctl keys export myaccount
```

### List Accounts

```bash
tronctl keys list

# Example output:
# Address                                Balance     Type
# TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n    1000 TRX   Local
# TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9    500 TRX    Ledger
```

### Remove Account

```bash
tronctl keys remove <account-name>

# Example
tronctl keys remove myaccount
```

## Blockchain Commands

### Get Node Information

```bash
tronctl bc getnodeinfo

# Example output shows node version, network, and configuration
```

### Get Transaction by ID

```bash
tronctl bc gettransaction <transaction-id>

# Example
tronctl bc gettransaction 7c2d4206c03a883dd9066d620335dc1be272a8dc733cfa3f6d10308faa37facc
```

### Get Block by Number

```bash
tronctl bc getblock <block-number>

# Example
tronctl bc getblock 30000000
```

### Get Latest Block

```bash
tronctl bc getblockbylatest
```

### Get Next Maintenance Time

```bash
tronctl bc nextmaintenancetime
```

## Smart Contract Commands

### Deploy Contract

```bash
tronctl contract deploy <contract.bin> <abi.json>

# Options
--signer <name>          Deployer account name (required)
--name <string>          Contract name
--consume-user-energy <percent>  User energy consumption percentage
--fee-limit <amount>     Maximum TRX to spend
--constructor <args>     Constructor arguments (JSON format)

# Example
tronctl contract deploy Token.bin Token.abi --signer myaccount --name "MyToken" --constructor '["My Token", "MTK", 1000000]'
```

### Call Contract (Read)

```bash
tronctl contract call <contract-address> <method> <args>

# Options
--abi <file>            ABI file path

# Example
tronctl contract call TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 balanceOf '["TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n"]'
```

### Trigger Contract (Write)

```bash
tronctl contract trigger <contract-address> <method> <args>

# Options
--signer <name>         Caller account name (required)
--abi <file>            ABI file path
--fee-limit <amount>    Maximum TRX to spend
--call-value <amount>   TRX to send with call

# Example
tronctl contract trigger TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 transfer '["TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", 1000]' --signer myaccount --fee-limit 10
```

### Get Contract Info

```bash
tronctl contract get <contract-address>

# Example
tronctl contract get TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9
```

## TRC10 Token Commands

### Issue Token

```bash
tronctl trc10 issue

# Options
--name <string>          Token name
--abbr <string>          Token abbreviation
--supply <amount>        Total supply
--decimal <number>       Decimal places
--description <text>     Token description
--url <string>          Project URL
--start-time <time>      ICO start time
--end-time <time>        ICO end time
--frozen-supply <list>   Frozen supply schedule

# Example
tronctl trc10 issue --name "MyToken" --abbr "MTK" --supply 1000000 --decimal 6
```

### Transfer TRC10 Token

```bash
tronctl trc10 send <to-address> <amount> <token-id>

# Options
--signer <name>          Account name (required)

# Example
tronctl trc10 send TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 1000 1000001 --signer myaccount
```

### Participate in Token ICO

```bash
tronctl trc10 participate <issuer-address> <token-id> <amount>

# Options
--signer <name>          Account name (required)

# Example
tronctl trc10 participate TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n 1000001 100 --signer myaccount
```

### Get Asset List

```bash
tronctl trc10 list

# Options
--limit <number>         Number of results (default: 20)
--offset <number>        Starting offset

# Example
tronctl trc10 list --limit 50
```

## TRC20 Token Commands

### Send TRC20 Token

```bash
tronctl trc20 send <contract-address> <to-address> <amount>

# Options
--signer <name>          Account name (required)
--fee-limit <amount>    Maximum TRX to spend

# Example
tronctl trc20 send TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 100 --signer myaccount --fee-limit 10
```

### Get TRC20 Balance

```bash
tronctl trc20 balance <contract-address> <account-address>

# Example
tronctl trc20 balance TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n
```

### Get TRC20 Token Info

```bash
tronctl trc20 info <contract-address>

# Example
tronctl trc20 info TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
```

## Super Representative Commands

### List Witnesses

```bash
tronctl sr list

# Example output shows SR addresses, votes, and URLs
```

### Create Super Representative

```bash
tronctl sr create <url>

# Options
--signer <name>          Account name (required)

# Example
tronctl sr create "https://my-sr.com" --signer myaccount
```

### Update Super Representative

```bash
tronctl sr update <url>

# Options
--signer <name>          Account name (required)

# Example
tronctl sr update "https://my-new-sr-url.com" --signer myaccount
```

### Get Brokerage Info

```bash
tronctl sr brokerage <address>

# Example
tronctl sr brokerage TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH
```

### Update Brokerage Rate

```bash
tronctl sr update-brokerage <rate>

# Options
--signer <name>          Account name (required)

# Note: Rate is 0-100 (percentage)
# Example
tronctl sr update-brokerage 20 --signer myaccount
```

## Proposal Commands

### List Proposals

```bash
tronctl proposal list

# Options
--status <string>       Filter by status (pending, approved, canceled)

# Example
tronctl proposal list --status pending
```

### Create Proposal

```bash
tronctl proposal create

# Options
--param <key=value>     Parameter to change (can use multiple times)
--signer <name>          Account name (required)(must be SR)

# Example
tronctl proposal create --param 0=100000 --param 1=200000 --signer myaccount
```

### Approve Proposal

```bash
tronctl proposal approve <proposal-id>

# Options
--signer <name>          Account name (required)(must be SR)

# Example
tronctl proposal approve 15 --signer myaccount
```

### Delete Proposal

```bash
tronctl proposal delete <proposal-id>

# Options
--signer <name>          Account name (required)

# Example
tronctl proposal delete 15 --signer myaccount
```

## Exchange Commands

### Create Exchange

```bash
tronctl exchange create <token1> <token1-balance> <token2> <token2-balance>

# Options
--signer <name>          Account name (required)

# Example
tronctl exchange create TRX 10000 1000001 50000 --signer myaccount
```

### Inject Exchange

```bash
tronctl exchange inject <exchange-id> <token-id> <amount>

# Options
--signer <name>          Account name (required)

# Example
tronctl exchange inject 1 TRX 1000 --signer myaccount
```

### Withdraw from Exchange

```bash
tronctl exchange withdraw <exchange-id> <token-id> <amount>

# Options
--signer <name>          Account name (required)

# Example
tronctl exchange withdraw 1 1000001 500 --signer myaccount
```

### Trade on Exchange

```bash
tronctl exchange trade <exchange-id> <token-id> <amount> <expected>

# Options
--signer <name>          Account name (required)

# Example
tronctl exchange trade 1 TRX 100 45 --signer myaccount
```

## Configuration

### Initialize Configuration

```bash
tronctl config init

# Creates default config at ~/.tronctl/config.yaml
```

### Show Configuration

```bash
tronctl config show

# Example output:
# node: grpc.trongrid.io:50051
# network: mainnet
# timeout: 60s
# tls: true
```

### Set Configuration Value

```bash
tronctl config set <key> <value>

# Examples
tronctl config set node grpc.trongrid.io:50051
tronctl config set tls true
tronctl config set apiKey your-api-key
```

## Utility Commands

### Convert Address

```bash
tronctl utils addr2hex <base58-address>
tronctl utils hex2addr <hex-address>

# Examples
tronctl utils addr2hex TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n
tronctl utils hex2addr 41928c9af0651632157ef27a2cf17ca72c575a4d21
```

### Generate Address

```bash
tronctl utils genaddr

# Options
--count <number>        Number of addresses to generate
--prefix <string>       Desired address prefix

# Example
tronctl utils genaddr --count 10 --prefix TR
```

## Examples

### Complete Transaction Flow

```bash
# 1. Create account
tronctl keys add myaccount

# 2. Check balance
tronctl account balance TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n

# 3. Send TRX
tronctl account send TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 100 --signer myaccount

# 4. Check transaction
tronctl bc gettransaction <tx-id>
```

### Deploy and Interact with Contract

```bash
# 1. Deploy contract
tronctl contract deploy Token.bin Token.abi --signer myaccount --name "MyToken" --constructor '["My Token", "MTK", 1000000]'

# 2. Call read method
tronctl contract call <contract-address> totalSupply '[]'

# 3. Trigger write method
tronctl contract trigger <contract-address> transfer '["TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9", 1000]' --signer myaccount --fee-limit 10
```

### Participate in Network Governance

```bash
# 1. Freeze TRX for voting power
tronctl account freeze 10000 BANDWIDTH --signer myaccount

# 2. Vote for SR
tronctl account vote TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH 10000 --signer myaccount

# 3. Check vote status
tronctl sr list
```

## Environment Variables

The following environment variables affect tronctl behavior:

- `TRON_NODE`: Default node address
- `TRON_NODE_TLS`: Enable TLS (true/false)
- `TRONGRID_APIKEY`: Trongrid API key
- `GOTRON_SDK_DEBUG`: Enable debug output

## Notes

- Most commands that modify blockchain state require a signer (account with private key)
- Transaction fees apply to most operations
- Use `--fee-limit` to control maximum fees for smart contract interactions
- Always test on testnet before mainnet operations