#!/bin/sh
# Installs tronctl from a GitHub release of fbsobreira/gotron-sdk.
#
# The archive is checked against the release's checksums.txt before anything
# is installed. Anything that prevents that check (checksums.txt missing, the
# archive not listed in it, a mismatch, no sha256 tool) aborts the install.
#
#   curl -fsSL https://gotron.sh/install | sh
#   curl -fsSL https://gotron.sh/install | sh -s -- --version v0.27.0 --pq
set -eu

REPO="fbsobreira/gotron-sdk"
BINARY="tronctl"
CHECKSUMS="checksums.txt"
INSTALL_DIR="/usr/local/bin"
INSTALL_DIR_SET=0

die() {
  echo "Error: $*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage: install.sh [--version vX.Y.Z] [--pq] [--install-dir DIR] [--dry-run]

  --version VERSION   install this release (default: the latest release)
  --pq                install tronctl-pq, the build that can also sign with
                      FN-DSA-512 (published from v0.27.0)
  --install-dir DIR   install into DIR (default: /usr/local/bin, using sudo
                      when needed, else ~/.local/bin)
  --dry-run           print what would be downloaded and installed, and exit
EOF
}

VERSION=""
DRY_RUN=0
PQ=0
expect=""
for arg in "$@"; do
  if [ -n "$expect" ]; then
    case "$expect" in
      version) VERSION="$arg" ;;
      install-dir)
        INSTALL_DIR="$arg"
        INSTALL_DIR_SET=1
        ;;
    esac
    expect=""
    continue
  fi
  case "$arg" in
    --version=*) VERSION="${arg#--version=}" ;;
    --version) expect="version" ;;
    --install-dir=*)
      INSTALL_DIR="${arg#--install-dir=}"
      INSTALL_DIR_SET=1
      ;;
    --install-dir) expect="install-dir" ;;
    --pq) PQ=1 ;;
    --dry-run) DRY_RUN=1 ;;
    -h | --help)
      usage
      exit 0
      ;;
    *) die "unknown option: $arg (see --help)" ;;
  esac
done
[ -z "$expect" ] || die "--$expect needs a value"
[ -n "$INSTALL_DIR" ] || die "--install-dir needs a value"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) die "unsupported architecture: $ARCH" ;;
esac

case "$OS" in
  linux | darwin) ;;
  *) die "unsupported OS: $OS. For Windows, download from https://github.com/$REPO/releases" ;;
esac

if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
    grep '"tag_name"' | head -1 | cut -d'"' -f4) || VERSION=""
  [ -n "$VERSION" ] || die "could not determine the latest version"
fi

# The version ends up in download URLs, so accept only a release tag.
case "$VERSION" in
  v*) ;;
  *) VERSION="v$VERSION" ;;
esac
case "$VERSION" in
  *[!0-9A-Za-z.+-]*) die "invalid version: $VERSION" ;;
esac
case "$VERSION" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) die "invalid version: $VERSION (expected vMAJOR.MINOR.PATCH)" ;;
esac

NAME="$BINARY"
if [ "$PQ" = "1" ]; then
  NAME="tronctl-pq"
fi
ARCHIVE="${NAME}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/$REPO/releases/download/${VERSION}"

if command -v sha256sum >/dev/null 2>&1; then
  SHA256="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA256="shasum -a 256"
else
  die "no sha256 tool found (sha256sum or shasum), so the download cannot be verified"
fi

if [ "$DRY_RUN" = "1" ]; then
  echo "[dry-run] OS:          $OS"
  echo "[dry-run] Arch:        $ARCH"
  echo "[dry-run] Version:     $VERSION"
  echo "[dry-run] Archive:     $ARCHIVE"
  echo "[dry-run] Download:    $BASE_URL/$ARCHIVE"
  echo "[dry-run] Checksums:   $BASE_URL/$CHECKSUMS"
  if [ "$INSTALL_DIR_SET" = "1" ] || [ -w "$INSTALL_DIR" ]; then
    echo "[dry-run] Install to:  $INSTALL_DIR/$BINARY"
  elif command -v sudo >/dev/null 2>&1; then
    echo "[dry-run] Install to:  $INSTALL_DIR/$BINARY (via sudo)"
  else
    [ -n "${HOME:-}" ] || die "HOME is not set, so there is no ~/.local/bin to fall back to; pass --install-dir DIR"
    echo "[dry-run] Install to:  $HOME/.local/bin/$BINARY (fallback)"
  fi
  exit 0
fi

echo "Installing $NAME $VERSION ($OS/$ARCH)..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$BASE_URL/$CHECKSUMS" -o "$TMP/$CHECKSUMS" ||
  die "could not download $CHECKSUMS for $VERSION; refusing to install an unverified archive"

# checksums.txt lines are "<sha256>  <file>"; sha256sum's binary mode writes
# "<sha256> *<file>". Compare the file name exactly, not as a pattern, and
# require it once: with two lines it is unclear which one is meant.
COUNT=$(awk -v f="$ARCHIVE" '$2 == f || $2 == "*" f { n++ } END { print n + 0 }' "$TMP/$CHECKSUMS")
[ "$COUNT" -le 1 ] || die "$ARCHIVE is listed more than once in $CHECKSUMS"
EXPECTED=$(awk -v f="$ARCHIVE" '$2 == f || $2 == "*" f { print $1; exit }' "$TMP/$CHECKSUMS")
if [ -z "$EXPECTED" ]; then
  if [ "$PQ" = "1" ]; then
    die "$ARCHIVE is not listed in $CHECKSUMS for $VERSION (tronctl-pq is published from v0.27.0)"
  fi
  die "$ARCHIVE is not listed in $CHECKSUMS for $VERSION"
fi
case "$EXPECTED" in
  *[!0-9a-f]*) die "malformed checksum for $ARCHIVE in $CHECKSUMS" ;;
esac
[ "${#EXPECTED}" -eq 64 ] || die "malformed checksum for $ARCHIVE in $CHECKSUMS"

curl -fsSL "$BASE_URL/$ARCHIVE" -o "$TMP/$ARCHIVE" || die "could not download $ARCHIVE"

ACTUAL=$($SHA256 "$TMP/$ARCHIVE" | awk '{ print $1 }')
if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "Error: checksum mismatch for $ARCHIVE" >&2
  echo "  expected: $EXPECTED" >&2
  echo "  got:      $ACTUAL" >&2
  exit 1
fi
echo "Checksum verified."

# Extract only the binary, so no other path in the archive is written. List
# the archive first, so a missing binary and a broken archive give different
# errors; tar's own message stays visible for the latter.
mkdir "$TMP/x"
tar -tzf "$TMP/$ARCHIVE" >"$TMP/list" || die "could not read $ARCHIVE"
awk -v b="$BINARY" '$0 == b { found = 1 } END { exit !found }' "$TMP/list" ||
  die "$BINARY not found in $ARCHIVE"
# -o: do not keep the archive's owner (uid 1001 in release archives); as root,
# tar would otherwise restore it on the installed binary.
tar -xozf "$TMP/$ARCHIVE" -C "$TMP/x" "$BINARY" || die "could not extract $BINARY from $ARCHIVE"
if [ ! -f "$TMP/x/$BINARY" ] || [ -L "$TMP/x/$BINARY" ]; then
  die "$BINARY in $ARCHIVE is not a regular file"
fi
chmod +x "$TMP/x/$BINARY"

if [ "$INSTALL_DIR_SET" = "1" ]; then
  mkdir -p "$INSTALL_DIR" 2>/dev/null || true
fi

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/x/$BINARY" "$INSTALL_DIR/$BINARY"
elif command -v sudo >/dev/null 2>&1; then
  echo "Need sudo to install to $INSTALL_DIR"
  sudo mv "$TMP/x/$BINARY" "$INSTALL_DIR/$BINARY"
elif [ "$INSTALL_DIR_SET" = "1" ]; then
  die "$INSTALL_DIR is not writable"
else
  [ -n "${HOME:-}" ] || die "$INSTALL_DIR is not writable, no sudo is available and HOME is not set; pass --install-dir DIR"
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
  mv "$TMP/x/$BINARY" "$INSTALL_DIR/$BINARY"
  echo "Installed to $INSTALL_DIR (no sudo available)"
  echo "Make sure $INSTALL_DIR is in your PATH"
fi
echo "$NAME $VERSION installed to $INSTALL_DIR/$BINARY"
