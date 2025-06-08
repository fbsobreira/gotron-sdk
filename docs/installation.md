# Installation Guide

This guide provides detailed instructions for installing GoTRON SDK and the `tronctl` CLI tool.

## Table of Contents

- [Requirements](#requirements)
- [Installation Methods](#installation-methods)
  - [Install from Source](#install-from-source)
  - [Install with Go Get](#install-with-go-get)
  - [Install Pre-built Binaries](#install-pre-built-binaries)
- [Platform-Specific Instructions](#platform-specific-instructions)
  - [Linux](#linux)
  - [macOS](#macos)
  - [Windows](#windows)
- [Verification](#verification)
- [Shell Completion](#shell-completion)
- [Troubleshooting](#troubleshooting)

## Requirements

Before installing GoTRON SDK, ensure you have the following prerequisites:

- **Go**: Version 1.18 or higher
- **Git**: For cloning the repository
- **Make**: For building from source (optional but recommended)
- **Protocol Buffers**: Only required if regenerating proto files

### Checking Prerequisites

```bash
# Check Go version
go version

# Check Git version
git --version

# Check Make version (optional)
make --version
```

## Installation Methods

### Install from Source

This is the recommended method as it ensures you get the latest version with all features.

```bash
# Clone the repository
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk

# Build and install to ~/.local/bin
make install

# Or install to GOPATH/bin
make build
go install ./cmd/...
```

### Install with Go Get

For quick installation as a Go module:

```bash
# Install the CLI tool
go install github.com/fbsobreira/gotron-sdk/cmd/tronctl@latest

# Add as dependency to your project
go get github.com/fbsobreira/gotron-sdk
```

### Install Pre-built Binaries

Download pre-built binaries from the [releases page](https://github.com/fbsobreira/gotron-sdk/releases):

```bash
# Linux example
wget https://github.com/fbsobreira/gotron-sdk/releases/download/v1.3.0/tronctl-linux-amd64
chmod +x tronctl-linux-amd64
sudo mv tronctl-linux-amd64 /usr/local/bin/tronctl

# Verify installation
tronctl version
```

## Platform-Specific Instructions

### Linux

#### Ubuntu/Debian

```bash
# Install dependencies
sudo apt-get update
sudo apt-get install -y build-essential git golang

# Clone and build
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk
make install

# Add to PATH if needed
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

#### Fedora/RHEL/CentOS

```bash
# Install dependencies
sudo dnf install -y git golang make

# Clone and build
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk
make install

# Add to PATH if needed
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

#### Arch Linux

```bash
# Install dependencies
sudo pacman -S git go make

# Clone and build
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk
make install
```

### macOS

#### Using Homebrew

```bash
# Install Go if not already installed
brew install go

# Clone and build
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk
make install

# Add to PATH if needed
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

#### Manual Installation

```bash
# Install Xcode Command Line Tools
xcode-select --install

# Clone and build
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk
make install
```

### Windows

#### Using Git Bash or WSL

```bash
# Clone repository
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk

# Build for Windows
make build-windows

# Or use go build directly
GOOS=windows GOARCH=amd64 go build -o tronctl.exe ./cmd/main.go
```

#### Using PowerShell

```powershell
# Clone repository
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk

# Build
go build -o tronctl.exe ./cmd/main.go

# Add to PATH
$env:Path += ";$PWD"
```

## Verification

After installation, verify that `tronctl` is working correctly:

```bash
# Check version
tronctl version

# Display help
tronctl help

# Test connection to TRON network
tronctl bc getnodeinfo
```

## Shell Completion

Enable shell completion for better CLI experience:

### Bash

```bash
# Add to ~/.bashrc
echo 'source <(tronctl completion bash)' >> ~/.bashrc
source ~/.bashrc
```

### Zsh

```bash
# Add to ~/.zshrc
echo 'source <(tronctl completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

### Fish

```bash
# Generate completion file
tronctl completion fish > ~/.config/fish/completions/tronctl.fish
```

### PowerShell

```powershell
# Add to PowerShell profile
tronctl completion powershell | Out-String | Invoke-Expression
```

## Configuration

After installation, create a configuration file:

```bash
# Create config directory
mkdir -p ~/.tronctl

# Create config file
cat > ~/.tronctl/config.yaml << EOF
node: grpc.trongrid.io:50051
network: mainnet
timeout: 60s
tls: true
EOF
```

## Troubleshooting

### Common Issues

#### Command not found

If `tronctl` is not found after installation:

```bash
# Check if binary exists
ls -la ~/.local/bin/tronctl

# Add to PATH
export PATH="$HOME/.local/bin:$PATH"

# For permanent solution, add to shell profile
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```

#### Permission denied

```bash
# Make binary executable
chmod +x ~/.local/bin/tronctl
```

#### Go version too old

```bash
# Update Go to latest version
# On Linux/macOS
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
```

#### Build errors

```bash
# Clean and rebuild
make clean
go mod download
make build
```

### Getting Help

If you encounter issues:

1. Check the [GitHub Issues](https://github.com/fbsobreira/gotron-sdk/issues)
2. Read the [FAQ](https://github.com/fbsobreira/gotron-sdk/wiki/FAQ)
3. Join the community discussions

## Next Steps

After successful installation:

1. Read the [CLI Usage Guide](cli-usage.md) to learn about available commands
2. Check out [Examples](examples.md) for common use cases
3. Explore the [SDK Usage Guide](sdk-usage.md) if building applications

## Updating

To update to the latest version:

```bash
# If installed from source
cd gotron-sdk
git pull origin master
make clean
make install

# If installed with go get
go get -u github.com/fbsobreira/gotron-sdk/cmd/tronctl@latest
```