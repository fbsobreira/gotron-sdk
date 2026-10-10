#!/bin/bash

# Script to process proto files for Tron SDK

set -e  # Exit on any error

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_msg() {
  echo -e "${GREEN}$1${NC}"
}

print_warn() {
  echo -e "${YELLOW}$1${NC}"
}

# Check if necessary tools are installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed. Please install Protocol Buffers."
    exit 1
fi
if ! command -v patch &> /dev/null; then
    echo "Error: patch is not installed (needed for proto/patches)."
    exit 1
fi

# The committed pkg/proto was generated with this protoc; any other version
# rewrites the version header of every generated file.
PROTOC_EXPECTED="libprotoc 29.3"
if [ "$(protoc --version)" != "$PROTOC_EXPECTED" ]; then
    print_warn "⚠️  $(protoc --version) found, pkg/proto was generated with $PROTOC_EXPECTED — expect header-only churn."
fi

# Set up directories
PROTO_SRC_DIR="./proto/tron"
PROTO_BACKUP_DIR="./proto/tron_backup"
PROTO_OUT_DIR="./pkg/proto"

restore_protos() {
    if [ -d "$PROTO_BACKUP_DIR" ]; then
        rm -rf "$PROTO_SRC_DIR"
        mv "$PROTO_BACKUP_DIR" "$PROTO_SRC_DIR"
    fi
}

# Create the output directory if it doesn't exist
mkdir -p "$PROTO_OUT_DIR"

# Backup original .proto files if needed
if [ -d "$PROTO_SRC_DIR" ]; then
    print_msg "⚠️  Creating proto backup..."
    # Remove existing backup if it exists
    if [ -d "$PROTO_BACKUP_DIR" ]; then
        rm -rf "$PROTO_BACKUP_DIR"
    fi
    cp -r "$PROTO_SRC_DIR" "$PROTO_BACKUP_DIR"
    # Restore on any exit, so a failed step never leaves the submodule modified
    # (.gitmodules sets ignore = dirty, which would hide it from git status).
    trap restore_protos EXIT
else
    print_warn "⚠️  No .proto files found to copy. Please check your source files."
    exit 1
fi

# Modify import references in all .proto files
print_msg "🔄 Updating import references in .proto files..."
find "$PROTO_SRC_DIR" -name "*.proto" -type f -exec sed -i.bak 's|github.com/tronprotocol/grpc-gateway|github.com/fbsobreira/gotron-sdk/pkg/proto|g' {} \;

# Remove .bak files on macOS (sed behaves differently)
find "$PROTO_SRC_DIR" -name "*.bak" -type f -delete

# Apply vendored patches for protocol changes not yet merged upstream
for p in ./proto/patches/*.patch; do
    [ -e "$p" ] || continue
    print_msg "🩹 Applying $(basename "$p")..."
    if ! patch -p1 -F0 --forward --no-backup-if-mismatch -r - -d "$PROTO_SRC_DIR" < "$p"; then
        echo "Error: $p does not apply. If upstream merged it, delete the patch;" >&2
        echo "if it was already applied by an interrupted run, reset with: git submodule update --force proto/tron" >&2
        exit 1
    fi
done

# Ensure the required directories exist
mkdir -p "$PROTO_OUT_DIR/core"
mkdir -p "$PROTO_OUT_DIR/api"
mkdir -p "$PROTO_OUT_DIR/util"

# --- Includes and protoc flags ---
INCLUDES=(
  -I="$PROTO_SRC_DIR"
  -I=./proto/googleapis
  -I=/usr/lib
)

FLAGS=(
  --go_out="$PROTO_OUT_DIR"
  --go_opt=paths=source_relative
  --go-grpc_out="$PROTO_OUT_DIR"
  --go-grpc_opt=paths=source_relative
)

# --- Run protoc ---
print_msg "📦 Generating proto files..."
protoc "${INCLUDES[@]}" "${FLAGS[@]}" \
  $PROTO_SRC_DIR/core/*.proto \
  $PROTO_SRC_DIR/core/contract/*.proto \
  $PROTO_SRC_DIR/api/*.proto

# --- Build util protos ---
print_msg "🛠 Generating util protos..."
protoc "${INCLUDES[@]}" -I=./proto/util \
  --go_out="$PROTO_OUT_DIR/util" \
  --go_opt=paths=source_relative \
  ./proto/util/*.proto

# --- Restore original .proto files ---
print_msg "🔄 Restoring original .proto files..."
restore_protos

# --- Move files from core/contract to core ---
if [ -d "$PROTO_OUT_DIR/core/contract" ]; then
    print_msg "📂 Moving files from core/contract to core..."
    mv "$PROTO_OUT_DIR/core/contract"/* "$PROTO_OUT_DIR/core/"
    # Remove the empty directory
    rmdir "$PROTO_OUT_DIR/core/contract"
else
    print_warn "⚠️  Directory $PROTO_OUT_DIR/core/contract does not exist, nothing to move."
fi

print_msg "✅ All operations completed successfully!"