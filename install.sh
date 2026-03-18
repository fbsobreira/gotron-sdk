#!/bin/sh
set -e

REPO="fbsobreira/gotron-sdk"
BINARY="tronctl"
INSTALL_DIR="/usr/local/bin"

# Parse arguments
VERSION=""
DRY_RUN=0
for arg in "$@"; do
  case "$arg" in
    --version=*) VERSION="${arg#--version=}" ;;
    --version)   shift_next=1 ;;
    --dry-run)   DRY_RUN=1 ;;
    *)
      if [ "${shift_next:-}" = "1" ]; then
        VERSION="$arg"
        shift_next=0
      fi
      ;;
  esac
done

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Error: unsupported OS: $OS"
    echo "For Windows, download from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
fi

if [ -z "$VERSION" ]; then
  echo "Error: could not determine latest version"
  exit 1
fi

# Ensure version starts with 'v'
case "$VERSION" in
  v*) ;;
  *)  VERSION="v$VERSION" ;;
esac

ARCHIVE="${BINARY}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
CHECKSUMS="checksums.txt"
BASE_URL="https://github.com/$REPO/releases/download/${VERSION}"

if [ "$DRY_RUN" = "1" ]; then
  echo "[dry-run] OS:          $OS"
  echo "[dry-run] Arch:        $ARCH"
  echo "[dry-run] Version:     $VERSION"
  echo "[dry-run] Archive:     $ARCHIVE"
  echo "[dry-run] Download:    $BASE_URL/$ARCHIVE"
  echo "[dry-run] Checksums:   $BASE_URL/$CHECKSUMS"
  if [ -w "$INSTALL_DIR" ]; then
    echo "[dry-run] Install to:  $INSTALL_DIR/$BINARY"
  elif command -v sudo >/dev/null 2>&1; then
    echo "[dry-run] Install to:  $INSTALL_DIR/$BINARY (via sudo)"
  else
    echo "[dry-run] Install to:  $HOME/.local/bin/$BINARY (fallback)"
  fi
  exit 0
fi

echo "Installing $BINARY $VERSION ($OS/$ARCH)..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$BASE_URL/$ARCHIVE" -o "$TMP/$ARCHIVE"

# Verify checksum if available
if curl -fsSL "$BASE_URL/$CHECKSUMS" -o "$TMP/$CHECKSUMS" 2>/dev/null; then
  EXPECTED=$(grep "$ARCHIVE" "$TMP/$CHECKSUMS" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL=$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL=$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')
    else
      ACTUAL=""
      echo "Warning: no sha256 tool found, skipping checksum verification"
    fi
    if [ -n "$ACTUAL" ]; then
      if [ "$ACTUAL" != "$EXPECTED" ]; then
        echo "Error: checksum mismatch"
        echo "  expected: $EXPECTED"
        echo "  got:      $ACTUAL"
        exit 1
      fi
      echo "Checksum verified."
    fi
  fi
else
  echo "Warning: checksums not available, skipping verification"
fi

tar xzf "$TMP/$ARCHIVE" -C "$TMP"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
elif command -v sudo >/dev/null 2>&1; then
  echo "Need sudo to install to $INSTALL_DIR"
  sudo mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
  sudo chmod +x "$INSTALL_DIR/$BINARY"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
  mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
  echo "Installed to $INSTALL_DIR (no sudo available)"
  echo "Make sure $INSTALL_DIR is in your PATH"
fi

chmod +x "$INSTALL_DIR/$BINARY"
echo "$BINARY $VERSION installed to $INSTALL_DIR/$BINARY"
