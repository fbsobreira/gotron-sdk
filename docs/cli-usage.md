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

These options are accepted by every command:

```
-n, --node <host>          TRON node address (default: grpc.trongrid.io:50051)
-k, --apiKey <key>         TronGrid API key
    --withTLS              Use TLS connection
-s, --signer <signer>      Signer account name or address
    --ks-dir <path>        Config root holding account-keys/ and pq-keys/ (default: ~/.tronctl)
    --passphrase           Prompt for the keystore passphrase
    --passphrase-file <path>  Read the keystore passphrase from a file
-e, --ledger               Use a Ledger hardware wallet
    --no-wait              Do not wait for transaction confirmation
    --no-pretty            Disable pretty-printed JSON output
-v, --verbose              Enable verbose output (same as GOTRON_SDK_DEBUG=true)
```

Defaults for `node`, `apiKey`, `withTLS`, `ledger`, `verbose`, `nopretty` and `timeout` come from the config file at
`~/.config/tronctl/config.default` (see [Configuration](#configuration)).

### Transaction options (signing commands)

Commands that sign and broadcast a transaction also accept these options; every other command rejects
them as unknown flags:

```
    --dry-run              Sign and print the transaction without broadcasting it
    --timeout <seconds>    Confirmation wait in seconds; 0 = don't wait (default: 20)
```

They apply to `account activate|send|withdraw|freeze|freezeV2|unfreezeV2|vote|permission`,
`contract deploy|trigger`, `exchange create|inject|withdraw|trade`, `proposal approve|withdraw|create`,
`sr create|brokerage`, `trc10 issue|send|ico` and `trc20 send`. With `--dry-run` the signed transaction
is printed as JSON on stdout and its txID on stderr; nothing is broadcast.

## Account Commands

### Get Account Balance

```bash
tronctl account balance <address>

# Options
--details                Show detailed account information

# Example
tronctl account balance TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n
```

### Send TRX

```bash
tronctl account send <to-address> <amount>

# <amount> is in TRX

# Options
-s, --signer <name>      Signer account name or address (required)

# Example
tronctl account send TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 100.5 --signer myaccount
```

### Freeze Resources

```bash
tronctl account freezeV2 <amount>

# <amount> is in TRX. `account freeze <amount>` is the Stake 1.0 variant and also accepts --delegate <address>.

# Options
-s, --signer <name>      Account to freeze from (required)
-t, --type <number>      0 - Bandwidth (default) / 1 - Energy

# Example
tronctl account freezeV2 1000 --type 1 --signer myaccount
```

### Unfreeze Resources

```bash
tronctl account unfreezeV2 <amount>

# Options
-s, --signer <name>      Account to unfreeze from (required)
-t, --type <number>      0 - Bandwidth (default) / 1 - Energy

# Example
tronctl account unfreezeV2 1000 --type 0 --signer myaccount
```

### Vote for Witnesses

```bash
tronctl account vote --wv <witness-address>:<votes>[,<witness-address>:<votes>...]

# Options
-s, --signer <name>      Voter account name (required)
--wv <list>              witness1:vote1,witness2:vote2

# Example
tronctl account vote --wv TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH:100000 --signer myaccount
```

### Get Account Resources

```bash
tronctl account info <address>

# Example
tronctl account info TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n
```

### Update Account Permissions

```bash
tronctl account permission --allow <TYPE:THRESHOLD:ADDRESS1-WEIGHT+ADDRESS2-WEIGHT>

# Options
-s, --signer <name>      Account whose permissions are updated (required)
--allow <rule>           TYPE:THRESHOLD:ADDRESS1-WEIGHT+ADDRESS2-WEIGHT; TYPE is O (owner), A (active) or W (witness).
                         Repeat the flag or comma-separate rules.

# Example
tronctl account permission --signer myaccount --allow O:1:TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n-1 --allow A:2:TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9-1+TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH-1
```

### Sign Message

```bash
tronctl account sign <message>

# Options
-s, --signer <name>      Signer account name (required)
--hashMessage            Hash the message with Keccak256 before signing
--useFixedLength         Use the fixed-length message format

# Example
tronctl account sign "Hello TRON" --signer myaccount
```

## Key Management

### Create New Account

```bash
tronctl keys add <account-name>

# Options
--passphrase             Global flag: prompt for the keystore passphrase (otherwise the default passphrase is used)

# Example
tronctl keys add myaccount
```

### Import Private Key

```bash
tronctl keys import-private-key <private-key> [account-name]
tronctl keys import-ks <keystore-file> [account-name]

# Options
--passphrase             Global flag: prompt for the keystore passphrase (otherwise the default passphrase is used)
--quiet                  Do not print the imported account name

# Example
tronctl keys import-private-key 8f5c7e1a2b3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f myaccount
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
tronctl keys export-private-key <address>
tronctl keys export-ks <address> <directory>

# Example
tronctl keys export-private-key TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n
```

### List Accounts

```bash
tronctl keys list

# Example output (tab-separated columns; SCHEME is last):
# NAME                                  ADDRESS                               SCHEME
#
# alice                                 TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n    ECDSA
# alice-pq                              TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9    FN_DSA_512
```

### Remove Account

```bash
tronctl keys remove <account-name>

# Example
tronctl keys remove myaccount
```

### Post-quantum keys (experimental)

> **Experimental.** TIP-899 post-quantum signatures are a draft and live only on the Nile testnet. See
> [Post-Quantum Signatures](post-quantum.md).

PQ keys have no mnemonic: the keystore file is the only backup. They are stored in
`~/.tronctl/pq-keys/<name>/`, next to `account-keys/`, and account names are unique across both.

```bash
# Create a key (no mnemonic is shown)
tronctl keys add <account-name> --scheme fn-dsa-512|ml-dsa-44

# Import a PQ wallet-cli keystore (detected by its "scheme" field)
tronctl keys import-ks <keystore-file> [account-name]

# Import a TRON Toolkit plaintext JSON key (the source file stays unencrypted: delete it afterwards)
tronctl keys import-toolkit <toolkit-key-file> [account-name]

# List accounts; the SCHEME column comes last. A PQ wallet that cannot be read is
# reported on stderr and left out of the table.
tronctl keys list

# Copy the PQ keystore file to a directory
tronctl keys export-ks <address> <directory>

# Remove a key
tronctl keys remove <account-name>
```

`keys export-private-key` refuses PQ keys; back up the keystore file with `keys export-ks`.

PQ keys sign transactions (`account send`, `trc10`/`trc20` transfers, `contract`, freeze, vote, ...): pass the
key's name or address to `-s`. The transaction carries a `pq_auth_sig` and no ECDSA signature. The node must have
the scheme enabled (Nile today: FN-DSA-512 on, ML-DSA-44 off); otherwise `tronctl` refuses before broadcasting
with `post-quantum signature scheme disabled on node: ML_DSA_44` (or `FN_DSA_512`).

Message signing (`account sign` and `account verify`) is ECDSA-only: TIP-899 defines no post-quantum message
signature, so `account sign` refuses a PQ signer.

```bash
# 1. Create a key and note its address
tronctl keys add pqkey --scheme fn-dsa-512

# 2. Fund the address from the Nile faucet

# 3. Send 1 TRX signed with the PQ key
tronctl -n grpc.nile.trongrid.io:50051 -s pqkey account send <to-address> 1
```

**Build requirements.** ML-DSA-44 works in every build. FN-DSA-512 key creation and signing need a Falcon build
with [liboqs](https://openquantumsafe.org/liboqs/) installed (macOS: `brew install liboqs pkgconf`; Linux: build
liboqs from source with `scripts/build-liboqs.sh --prefix <dir>`, and set `CGO_LDFLAGS=-lcrypto` for the default
static liboqs):

```bash
CGO_ENABLED=1 go build -tags falcon -o ./bin/tronctl ./cmd/tronctl
```

The default release binaries (`tronctl_*` archives, Homebrew, `install.sh`) are built without CGO: they can
import, list and export FN-DSA-512 keys but cannot create or sign with them. Each release also ships
`tronctl-pq_*` archives with FN-DSA-512 built in, for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and
windows/amd64 (`install.sh --pq` installs one); see
[tronctl builds with FN-DSA-512](post-quantum.md#tronctl-builds-with-fn-dsa-512).

`tronctl upgrade` downloads the release archive for your platform, verifies it against the release's
`checksums.txt` (SHA-256), checks that the new binary runs and reports the expected version, and then replaces the
binary in place; any failure leaves the current binary untouched. A `tronctl-pq` build upgrades only to a
`tronctl-pq` archive and stops with an error when the release has none for its platform. Homebrew builds are told to
run `brew upgrade tronctl`, and binaries not built from a release (`go install` or `go build`) to run `go install`
or download a release, instead.

## Blockchain Commands

### Get Node Information

```bash
tronctl bc node

# Example output shows node version, network, and configuration
```

### Get Transaction by ID

```bash
tronctl bc tx <transaction-id>

# Example
tronctl bc tx 7c2d4206c03a883dd9066d620335dc1be272a8dc733cfa3f6d10308faa37facc
```

### Get Next Maintenance Time

```bash
tronctl bc mt
```

## Smart Contract Commands

### Deploy Contract

```bash
tronctl contract deploy <contract-name>

# Options
-s, --signer <name>      Deployer account name (required)
--bcFile <file>          Bytecode file (or --bc <hex>)
--abiFile <file>         ABI file (or --abi <json>)
--params <json>          Constructor parameters as JSON
--curPercent <percent>   User resource consumption percentage (default: 100)
--feeLimit <sun>         Maximum fee in SUN (default: 1000000000)
--oeLimit <energy>       Origin energy limit (default: 1000000)

# Example
tronctl contract deploy MyToken --bcFile Token.bin --abiFile Token.abi --signer myaccount --params '["My Token", "MTK", 1000000]'
```

### Call Contract (Read)

```bash
tronctl contract constant <contract-address> <method> [parameters]

# Example
tronctl contract constant TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 "balanceOf(address)" '[{"address": "TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n"}]'
```

### Trigger Contract (Write)

```bash
tronctl contract trigger <contract-address> <method> [parameters]

# Options
-s, --signer <name>     Caller account name (required)
--feeLimit <sun>        Maximum fee in SUN (default: 10000000)
--value <amount>        TRX to send with the call (default: 0)
--token <id>            TRC10 token id to send with the call
--tokenValue <amount>   TRC10 token amount (default: 0)
--estimate              Only estimate the energy required; do not sign or broadcast

# Example
tronctl contract trigger TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 "transfer(address,uint256)" '[{"address": "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"}, {"uint256": "1000"}]' --signer myaccount --feeLimit 10000000
```

## TRC10 Token Commands

### Issue Token

```bash
tronctl trc10 issue <name> <description> <symbol> <url> <total-supply> <ratio>

# <ratio> is TRX:TOKEN, e.g. 1:100

# Options
-s, --signer <name>      Issuer account name (required)
-p, --decimals <number>  Decimal places (max 6)
-d, --duration <days>    ICO duration in days (default: 30)
--start <time>           ICO start time (default: now)
-0, --frozen <list>      Frozen supply day1:amount1,day2:amount2

# Example
tronctl trc10 issue MyToken "My token" MTK https://example.com 1000000 1:100 --decimals 6 --signer myaccount
```

### Transfer TRC10 Token

```bash
tronctl trc10 send <to-address> <amount> <token-id-or-name>

# Options
-s, --signer <name>      Account name (required)

# Example
tronctl trc10 send TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 1000 1000001 --signer myaccount
```

### Participate in Token ICO

```bash
tronctl trc10 ico <token-id-or-name> <amount>

# Options
-s, --signer <name>      Account name (required)

# Example
tronctl trc10 ico 1000001 100 --signer myaccount
```

### Get Asset List

```bash
tronctl trc10 list

# Details of one token
tronctl trc10 info <token-id-or-issuer-address-or-name>
```

## TRC20 Token Commands

### Send TRC20 Token

```bash
tronctl trc20 send <to-address> <amount> <contract-address>

# Options
-s, --signer <name>      Account name (required)
--feeLimit <sun>         Maximum fee in SUN (default: 10000000)
--decimals <number>      Token decimals override (default: query the contract's decimals())

# Example
tronctl trc20 send TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9 100 TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t --signer myaccount --feeLimit 10000000
```

### Get TRC20 Balance

```bash
tronctl trc20 balance <account-address> <contract-address>

# Example
tronctl trc20 balance TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
```

## Super Representative Commands

### List Witnesses

```bash
tronctl sr list

# Options
--elected                Return elected SRs only
--brokerage              Add each SR's brokerage

# Example output shows SR addresses, votes, and URLs
```

### Create Super Representative

```bash
tronctl sr create <url>

# Options
-s, --signer <name>      Account name (required)

# Example
tronctl sr create "https://my-sr.com" --signer myaccount
```

### Get Brokerage Info

```bash
tronctl sr list --brokerage
```

### Update Brokerage Rate

```bash
tronctl sr brokerage <rate>

# Options
-s, --signer <name>      Account name (required)

# Note: Rate is 0-100 (percentage)
# Example
tronctl sr brokerage 20 --signer myaccount
```

## Proposal Commands

### List Proposals

```bash
tronctl proposal list

# Options
--new                   Show only new proposals

# Example
tronctl proposal list --new
```

### Create Proposal

```bash
tronctl proposal create --params <id:value>[,<id:value>...]

# Options
--params <list>         Parameters to change: ID:VALUE,ID:VALUE
-s, --signer <name>     Account name (required)(must be SR)

# Example
tronctl proposal create --params 0:100000,1:200000 --signer myaccount
```

### Approve Proposal

```bash
tronctl proposal approve <proposal-id> <true|false>

# Options
-s, --signer <name>      Account name (required)(must be SR)

# Example
tronctl proposal approve 15 true --signer myaccount
```

### Delete Proposal

```bash
tronctl proposal withdraw <proposal-id>

# Options
-s, --signer <name>      Account name (required)

# Example
tronctl proposal withdraw 15 --signer myaccount
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
tronctl exchange trade <exchange-id> <token-id> <amount>

# Options
-s, --signer <name>      Account name (required)
-x, --expected <amount>  Expected amount in return (default: 0)

# Example
tronctl exchange trade 1 TRX 100 --expected 45 --signer myaccount
```

## Configuration

The config file is created with defaults at `~/.config/tronctl/config.default` on first run.

### Show Configuration

```bash
tronctl config get <key>

# Example
tronctl config get node
```

### Set Configuration Value

```bash
tronctl config set <key> <value>

# Keys: node, apiKey, withTLS, ledger, verbose, nopretty, timeout

# Examples
tronctl config set node grpc.trongrid.io:50051
tronctl config set withTLS true
tronctl config set apiKey your-api-key
```

## Utility Commands

### Convert Address

```bash
tronctl utility base58-to-addr <base58-address>
tronctl utility addr-to-base58 <hex-address>

# Examples
tronctl utility base58-to-addr TPjGUuQfq6R3FMBmsacd6Z5dvAgrD2rz4n
tronctl utility addr-to-base58 41928c9af0651632157ef27a2cf17ca72c575a4d21
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
tronctl bc tx <tx-id>
```

### Deploy and Interact with Contract

```bash
# 1. Deploy contract
tronctl contract deploy MyToken --bcFile Token.bin --abiFile Token.abi --signer myaccount --params '["My Token", "MTK", 1000000]'

# 2. Call read method
tronctl contract constant <contract-address> "totalSupply()"

# 3. Trigger write method
tronctl contract trigger <contract-address> "transfer(address,uint256)" '[{"address": "TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9"}, {"uint256": "1000"}]' --signer myaccount --feeLimit 10000000
```

### Participate in Network Governance

```bash
# 1. Freeze TRX for voting power
tronctl account freezeV2 10000 --type 0 --signer myaccount

# 2. Vote for SR
tronctl account vote --wv TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH:10000 --signer myaccount

# 3. Check vote status
tronctl sr list
```

## Environment Variables

The following environment variables affect tronctl behavior:

- `TRONCTL_NODE`: Default node address
- `TRONCTL_TLS`: Enable TLS (true/1)
- `TRONCTL_SIGNER`: Default signer (account name or address)
- `TRONCTL_KS_DIR`: Config root, same as `--ks-dir`. Keystores are read from its
  `account-keys` folder and post-quantum keys from `pq-keys`, so the path should not end
  in `account-keys`. A relative path is resolved against the current directory, `~`
  against the home directory.
- `TRONGRID_APIKEY`: Trongrid API key
- `GOTRON_SDK_DEBUG`: Enable debug output

## Notes

- Most commands that modify blockchain state require a signer (account with private key)
- Transaction fees apply to most operations
- Use `--feeLimit` (in SUN) to control maximum fees for smart contract interactions
- Always test on testnet before mainnet operations
