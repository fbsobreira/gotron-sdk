# GoTRON SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/fbsobreira/gotron-sdk.svg)](https://pkg.go.dev/github.com/fbsobreira/gotron-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/fbsobreira/gotron-sdk)](https://goreportcard.com/report/github.com/fbsobreira/gotron-sdk)
[![License](https://img.shields.io/github/license/fbsobreira/gotron-sdk)](LICENSE)

GoTRON SDK is a comprehensive Go SDK and CLI tool for interacting with the TRON blockchain. It provides both a command-line interface (`tronctl`) and Go libraries for TRON blockchain operations.

## Features

- 🔧 **Complete CLI Tool**: Manage accounts, send transactions, interact with smart contracts
- 📚 **Go SDK**: Build TRON applications with a clean, idiomatic Go API
- 🔐 **Secure Key Management**: Hardware wallet support, encrypted keystores
- 🚀 **High Performance**: Native gRPC communication with TRON nodes
- 🛠️ **Developer Friendly**: Comprehensive examples and documentation

## Quick Start

### Installation

#### Install from source
```bash
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk
make install
```

#### Install with go get
```bash
go get -u github.com/fbsobreira/gotron-sdk
```

### Basic Usage

#### CLI Usage
```bash
# Create a new account
tronctl keys add <account-name>

# Check balance
tronctl account balance <address>

# Send TRX
tronctl account send <to-address> <amount> --signer <signer-name>
```

#### SDK Usage
```go
package main

import (
	"fmt"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
)

func main() {
	// Create client
	c := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := c.Start(client.GRPCInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer c.Stop()

	// Get account info
	addr, _ := address.Base58ToAddress("TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH")
	account, err := c.GetAccount(addr.String())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Balance: %d\n", account.Balance)
}

```

## Documentation

- [Installation Guide](docs/installation.md) - Detailed installation instructions
- [CLI Usage Guide](docs/cli-usage.md) - Complete CLI command reference
- [SDK Usage Guide](docs/sdk-usage.md) - Go SDK examples and patterns
- [API Reference](docs/api-reference.md) - Detailed API documentation
- [Examples](docs/examples.md) - Common use cases and examples

## Supported Features

### Account Management
- Create and import accounts
- Hardware wallet support (Ledger)
- Keystore management
- Multi-signature support

### Transactions
- TRX transfers
- TRC10 token operations
- TRC20 token operations
- Smart contract interactions
- Transaction signing and broadcasting

### Smart Contracts
- Contract deployment
- Contract calls and triggers
- ABI encoding/decoding
- Event monitoring

### Blockchain Queries
- Block information
- Transaction details
- Account resources
- Witness/SR information
- Proposal management

## Configuration

### Environment Variables
```bash
# Set custom node
export TRON_NODE="grpc.trongrid.io:50051"

# Enable TLS
export TRON_NODE_TLS="true"

# Set Trongrid API key
export TRONGRID_APIKEY="your-api-key"

# Enable debug mode
export GOTRON_SDK_DEBUG="true"
```

### Configuration File
Create `~/.tronctl/config.yaml`:
```yaml
node: grpc.trongrid.io:50051
network: mainnet
timeout: 60s
tls: true
apiKey: your-api-key
```

### Transfer JSON Format
For batch transfers, use a JSON file with the following format:

| Key                 | Value-type | Value-description|
| :------------------:|:----------:| :----------------|
| `from`              | string     | [**Required**] Sender's address, must have key in keystore |
| `to`                | string     | [**Required**] Receiver's address |
| `amount`            | string     | [**Required**] Amount to send in TRX |
| `passphrase-file`   | string     | [*Optional*] File path containing passphrase |
| `passphrase-string` | string     | [*Optional*] Passphrase as string |
| `stop-on-error`     | boolean    | [*Optional*] Stop on error (default: false) |

Example:
```json
[
  {
    "from": "TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH",
    "to": "TKSXDA8HfE9E1y39RczVQ1ZascUEtaSToF",
    "amount": "100",
    "passphrase-string": "",
    "stop-on-error": true
  }
]
```

## Shell Completion

Add to your `.bashrc` or `.zshrc`:
```bash
# Bash
source <(tronctl completion bash)

# Zsh
source <(tronctl completion zsh)
```

## Development

### Requirements
- Go 1.18 or higher
- Make (for building)
- Protocol Buffers compiler (for regenerating protos)

### Building
```bash
# Build binary
make build

# Cross-compile for Windows
make build-windows

# Run tests
make test

# Run linter
make lint

# Generate protobuf files
./gen-proto.sh
```

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Version History

### Note on Versions
The v2.x.x releases were incorrectly tagged without proper Go module versioning. These versions have been retracted. Please use v1.x.x versions or later.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://github.com/fbsobreira/gotron-sdk/tree/master/docs)
- 🐛 [Issue Tracker](https://github.com/fbsobreira/gotron-sdk/issues)
- 💬 [Discussions](https://github.com/fbsobreira/gotron-sdk/discussions)
