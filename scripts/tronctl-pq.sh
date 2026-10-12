#!/usr/bin/env bash
# Build, smoke-test and package the tronctl-pq release binaries: tronctl built
# with -tags falcon and a statically linked liboqs, so that it can create
# FN-DSA-512 keys and sign with them. .github/workflows/tronctl-pq.yaml runs
# these steps on each native runner; running them locally gives the same
# result.
#
# Usage:
#   scripts/tronctl-pq.sh build OUT
#       Check the Go toolchain (go1.26.5 or a later 1.26 patch), run
#       go test -tags falcon ./pkg/pqc/..., build tronctl to OUT and check that
#       the binary carries the falcon tag. liboqs must be discoverable through
#       pkg-config (PKG_CONFIG_PATH).
#   scripts/tronctl-pq.sh linux-container
#       The Linux build, run as root inside the golang:1.26-alpine image
#       pinned in tronctl-pq.yaml, with the repository at /src and a writable
#       /work: build liboqs into /work/liboqs (unless LIBOQS_CACHE_HIT=true),
#       then build a fully static musl binary to /work/tronctl and check that
#       it is statically linked.
#   scripts/tronctl-pq.sh smoke BIN
#       Offline smoke test: version, then keys add --scheme fn-dsa-512 and
#       keys list in a throwaway home directory. Never touches the network.
#   scripts/tronctl-pq.sh package BIN LIBOQS_PREFIX GOOS GOARCH OUTDIR
#       Write OUTDIR/tronctl-pq_<version>_<os>_<arch>.tar.gz (.zip for
#       windows) with the binary, LICENSE, README.md and THIRD_PARTY_NOTICES.txt
#       (liboqs and the third-party code it links in) at the archive root, like
#       the GoReleaser tronctl_* archives.
#
# Environment:
#   TRONCTL_PQ_VERSION    release version without the leading v (required for
#                         build and package)
#   TRONCTL_PQ_COMMIT     short commit (default: git rev-parse --short HEAD)
#   TRONCTL_PQ_TAGS       build tags (default: falcon)
#   TRONCTL_PQ_EXTLDFLAGS flags for the external linker; when set, tronctl is
#                         linked with -linkmode external -extldflags '<value>'

set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)

# Release binaries need a toolchain with the crypto/tls, net/textproto and
# crypto/x509 fixes (go1.26.5 or later), and stay on Go 1.26: a new Go minor
# can drop operating-system versions (Go 1.27 drops macOS 12). The minor moves
# deliberately, together with the go-version ranges in tronctl-pq.yaml and
# release.yaml, the golang image pinned in tronctl-pq.yaml and the macOS
# deployment target there.
GO_MAJOR=1
GO_MINOR=26
GO_MIN_PATCH=5

# GoReleaser's main.builtBy is goreleaser@gotron.sh; this one tells the two
# builds apart in tronctl version.
BUILT_BY=tronctl-pq@gotron.sh

# Temporary directories of cmd_smoke and cmd_package, removed on exit even
# when a step fails. Scalars rather than an array: bash 3.2 with set -u treats
# an empty "${arr[@]}" as unbound.
smoke_tmp=""
pkg_stage=""
cleanup() {
	if [ -n "$smoke_tmp" ]; then rm -rf "$smoke_tmp"; fi
	if [ -n "$pkg_stage" ]; then rm -rf "$pkg_stage"; fi
}
trap cleanup EXIT

die() {
	echo "tronctl-pq.sh: $*" >&2
	exit 1
}

usage() {
	sed -n '8,27p' "$0" | sed 's/^# \{0,1\}//' >&2
	exit 2
}

# native_path converts an MSYS2 path to a Windows path for native programs;
# elsewhere it prints the path unchanged.
native_path() {
	if command -v cygpath >/dev/null 2>&1; then
		cygpath -w "$1"
	else
		printf '%s\n' "$1"
	fi
}

check_go_version() {
	local v major minor patch rest
	v=$(go env GOVERSION)
	echo "Go toolchain: $v"
	rest=${v#go}
	major=${rest%%.*}
	rest=${rest#"$major"}
	rest=${rest#.}
	minor=${rest%%[!0-9]*}
	rest=${rest#"$minor"}
	patch=0
	if [ "${rest:0:1}" = . ]; then
		rest=${rest#.}
		patch=${rest%%[!0-9]*}
	fi
	case "$major$minor$patch" in
	'' | *[!0-9]*) die "cannot parse Go version '$v'" ;;
	esac
	if [ "$major" -ne "$GO_MAJOR" ] || [ "$minor" -ne "$GO_MINOR" ]; then
		die "$v is not Go $GO_MAJOR.$GO_MINOR; release binaries stay on go$GO_MAJOR.$GO_MINOR.x (see GO_MINOR)"
	fi
	if [ "$patch" -lt "$GO_MIN_PATCH" ]; then
		die "$v is older than go$GO_MAJOR.$GO_MINOR.$GO_MIN_PATCH, the floor for release binaries"
	fi
}

cmd_build() {
	[ $# -eq 1 ] || usage
	local out=$1 version commit built_at tags ldflags info
	version=${TRONCTL_PQ_VERSION:?TRONCTL_PQ_VERSION must be set (no leading v)}
	commit=${TRONCTL_PQ_COMMIT:-$(git -C "$root" rev-parse --short HEAD)}
	built_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	tags=${TRONCTL_PQ_TAGS:-falcon}

	check_go_version
	export CGO_ENABLED=1
	pkg-config --exists liboqs || die "liboqs not found via pkg-config (PKG_CONFIG_PATH=${PKG_CONFIG_PATH:-})"
	echo "liboqs $(pkg-config --modversion liboqs): $(pkg-config --cflags --libs liboqs)"

	go -C "$root" test -count=1 -tags "$tags" ./pkg/pqc/...

	ldflags="-s -w -X main.version=v$version -X main.commit=$commit -X main.builtAt=$built_at -X main.builtBy=$BUILT_BY"
	if [ -n "${TRONCTL_PQ_EXTLDFLAGS:-}" ]; then
		ldflags="$ldflags -linkmode external -extldflags '$TRONCTL_PQ_EXTLDFLAGS'"
	fi
	go -C "$root" build -tags "$tags" -ldflags "$ldflags" -o "$out" ./cmd/tronctl

	info=$(go version -m "$out")
	printf '%s\n' "$info"
	printf '%s\n' "$info" | grep -Eq '[[:space:]]build[[:space:]]+-tags=([^[:space:]]*,)?falcon(,|[[:space:]]|$)' ||
		die "$out was not built with -tags falcon"
	printf '%s\n' "$info" | grep -Eq '[[:space:]]build[[:space:]]+CGO_ENABLED=1([[:space:]]|$)' ||
		die "$out was not built with CGO_ENABLED=1"
}

cmd_linux_container() {
	[ $# -eq 0 ] || usage
	local desc

	# The checkout belongs to the runner user, not root: without this, git
	# refuses the repository and go build fails to stamp VCS information.
	git config --global --add safe.directory "$root"

	if [ "${LIBOQS_CACHE_HIT:-}" = true ] && [ -f /work/liboqs/lib/liboqs.a ]; then
		echo "liboqs restored from cache"
	else
		rm -rf /work/liboqs
		# No OpenSSL (nothing to link statically or ship) and a distributable
		# build (portable code, run-time CPU feature detection).
		bash "$root/scripts/build-liboqs.sh" --prefix /work/liboqs --openssl off --dist-build on -- -GNinja
	fi
	export PKG_CONFIG_PATH=/work/liboqs/lib/pkgconfig

	# netgo and osusergo keep net and os/user in pure Go, so the static musl
	# binary never needs glibc NSS at run time.
	TRONCTL_PQ_TAGS=falcon,netgo,osusergo TRONCTL_PQ_EXTLDFLAGS=-static cmd_build /work/tronctl

	desc=$(file -L /work/tronctl)
	printf '%s\n' "$desc"
	case "$desc" in
	*"statically linked"* | *"static-pie linked"*) ;;
	*) die "/work/tronctl is not statically linked" ;;
	esac
}

cmd_smoke() {
	[ $# -eq 1 ] || usage
	local bin tmp out
	bin=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
	[ -x "$bin" ] || die "$bin is not an executable"
	tmp=$(mktemp -d "${TMPDIR:-/tmp}/tronctl-pq-smoke.XXXXXX")
	smoke_tmp=$tmp
	mkdir -p "$tmp/home" "$tmp/ks" "$tmp/cwd"
	# A throwaway passphrase for a throwaway key.
	printf 'smoke-passphrase\n' >"$tmp/pp"

	# tronctl loads .env from the working directory, so run it from an empty
	# one, with every config and key location inside the temp directory.
	(
		cd "$tmp/cwd"
		export HOME USERPROFILE TRONCTL_KS_DIR
		HOME=$(native_path "$tmp/home")
		USERPROFILE=$HOME
		TRONCTL_KS_DIR=$(native_path "$tmp/ks")
		pp=$(native_path "$tmp/pp")
		unset TRONCTL_NODE TRONCTL_SIGNER TRONCTL_TLS TRONGRID_APIKEY

		"$bin" version
		"$bin" --passphrase-file "$pp" keys add smoke-fn --scheme fn-dsa-512
		# Captured first: with pipefail, grep -q exiting early could fail on
		# SIGPIPE.
		out=$("$bin" --passphrase-file "$pp" keys list | tr -d '\r')
		printf '%s\n' "$out"
		printf '%s\n' "$out" | grep -Eq '^smoke-fn[[:space:]].*[[:space:]]FN_DSA_512$' ||
			die "keys list does not show the FN_DSA_512 key"
	)
	rm -rf "$tmp"
	smoke_tmp=""
	echo "smoke test passed"
}

cmd_package() {
	[ $# -eq 5 ] || usage
	local bin=$1 prefix=$2 goos=$3 goarch=$4 outdir=$5 version name exe stage
	version=${TRONCTL_PQ_VERSION:?TRONCTL_PQ_VERSION must be set (no leading v)}
	name="tronctl-pq_${version}_${goos}_${goarch}"
	exe=tronctl
	if [ "$goos" = windows ]; then exe=tronctl.exe; fi

	mkdir -p "$outdir"
	outdir=$(cd "$outdir" && pwd)
	stage=$(mktemp -d "${TMPDIR:-/tmp}/tronctl-pq-pkg.XXXXXX")
	pkg_stage=$stage
	cp "$bin" "$stage/$exe"
	chmod 0755 "$stage/$exe"
	cp "$root/LICENSE" "$root/README.md" "$stage/"
	write_notices "$prefix/share/liboqs" >"$stage/THIRD_PARTY_NOTICES.txt"

	if [ "$goos" = windows ]; then
		rm -f "$outdir/$name.zip"
		(cd "$stage" && zip -q -X "$outdir/$name.zip" "$exe" LICENSE README.md THIRD_PARTY_NOTICES.txt)
		echo "$outdir/$name.zip"
	else
		tar -C "$stage" -czf "$outdir/$name.tar.gz" "$exe" LICENSE README.md THIRD_PARTY_NOTICES.txt
		echo "$outdir/$name.tar.gz"
	fi
	rm -rf "$stage"
	pkg_stage=""
}

# notice DIR FILE TITLE APPLIES_TO prints one THIRD_PARTY_NOTICES.txt section.
notice() {
	[ -s "$1/$2" ] || die "no $2 under $1 (build liboqs with scripts/build-liboqs.sh)"
	printf '\n%s\n%s\nApplies to: %s\n%s\n\n' \
		'================================================================================' \
		"$3" "$4" \
		'--------------------------------------------------------------------------------'
	cat "$1/$2"
}

# write_notices DIR prints THIRD_PARTY_NOTICES.txt from the licence files that
# scripts/build-liboqs.sh installs in DIR (PREFIX/share/liboqs). The set is
# the same for every platform: it covers the code linked on any of them.
write_notices() {
	local d=$1
	cat <<'HEADER'
THIRD-PARTY NOTICES FOR tronctl-pq

tronctl itself is licensed under the GNU Lesser General Public License v3.0
or later (see LICENSE). tronctl-pq binaries also contain liboqs
(https://openquantumsafe.org/liboqs/), statically linked and built with only
Falcon-512 (FN-DSA-512) enabled and without OpenSSL. liboqs is MIT licensed,
but it includes third-party code under other licences; the notices for the
code that such a build links in follow. Which CPU-specific variants a binary
contains depends on its platform: the x86-64 variants (Falcon AVX2, Keccak AVX2
and AVX-512VL, AES-NI) only on amd64, the ARMv8 variants (Falcon NEON, AES and
SHA-2 instructions) only on arm64.

The AES-NI, ARMv8 AES and SHA-2 block code (src/common/aes/aes*_ni.c,
aes*_armv8.c, src/common/sha2/sha2_c.c, sha2_armv8.c) is in the public domain
and needs no notice.
HEADER
	notice "$d" LICENSE.txt 'liboqs (MIT License)' \
		'liboqs as a whole, including src/common (AES, SHA-2, SHA-3 wrappers, random numbers), src/sig/falcon/sig_falcon_512.c'
	notice "$d" LICENSE-falcon-512.txt 'Falcon-512 reference and AVX2 code from PQClean (MIT License, Falcon Project)' \
		'src/sig/falcon/pqclean_falcon-512_clean, src/sig/falcon/pqclean_falcon-512_avx2'
	notice "$d" LICENSE-falcon-512-aarch64.txt 'Falcon-512 ARMv8 NEON code (Apache License 2.0, GMU CERG; based on MIT-licensed Falcon reference code)' \
		'src/sig/falcon/pqclean_falcon-512_aarch64 (the Apache License 2.0 text is at the end of this file)'
	notice "$d" NOTICE-aes_c-BearSSL.txt 'AES from BearSSL (MIT License, Thomas Pornin)' \
		'src/common/aes/aes_c.c'
	notice "$d" NOTICE-sha3-avx512vl-Intel.txt 'SHA-3 AVX-512VL code (MIT License, Intel Corporation; the MIT License text is as in the liboqs section above)' \
		'src/common/sha3/avx512vl_low, src/common/sha3/avx512vl_sha3*.c'
	notice "$d" NOTICE-xkcp-CC0.txt 'eXtended Keccak Code Package (CC0 1.0, public domain)' \
		'src/common/sha3/xkcp_low, except brg_endian.h and KeccakP-1600-AVX2.S'
	notice "$d" NOTICE-brg_endian.txt 'brg_endian.h (BSD-style licence, Brian Gladman)' \
		'src/common/sha3/xkcp_low/.../brg_endian.h'
	notice "$d" NOTICE-KeccakP-1600-AVX2.txt 'Keccak-p[1600] AVX2 code (CRYPTOGAMS licence; the licence text follows this section)' \
		'src/common/sha3/xkcp_low/KeccakP-1600/avx2/KeccakP-1600-AVX2.S'
	notice "$d" LICENSE-CRYPTOGAMS.txt 'CRYPTOGAMS licence (BSD-style)' \
		'src/common/sha3/xkcp_low/KeccakP-1600/avx2/KeccakP-1600-AVX2.S'
	notice "$d" NOTICE-rand_nist.txt 'NIST DRBG (NIST software notice)' \
		'src/common/rand/rand_nist.c'
	notice "$d" LICENSE-Apache-2.0.txt 'Apache License 2.0' \
		'src/sig/falcon/pqclean_falcon-512_aarch64, and the Apache-2.0 portions of src/common/common.c'
}

[ $# -ge 1 ] || usage
sub=$1
shift
case "$sub" in
build) cmd_build "$@" ;;
linux-container) cmd_linux_container "$@" ;;
smoke) cmd_smoke "$@" ;;
package) cmd_package "$@" ;;
-h | --help) usage ;;
*) usage ;;
esac
