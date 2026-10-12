#!/usr/bin/env bash
# Build and install the static liboqs that the FN-DSA-512 (falcon build tag)
# backend links against. This file is the single home of the liboqs pin: the
# CI test job (.github/workflows/falcon.yaml) and the tronctl-pq release
# builds (.github/workflows/tronctl-pq.yaml) both call it, and their caches
# are keyed on its hash.
#
# Usage: scripts/build-liboqs.sh --prefix DIR [--openssl on|off]
#                                [--dist-build on|off] [-- CMAKE_ARGS...]
#
#   --prefix DIR        install prefix (required); libraries go to DIR/lib
#   --openssl on|off    link liboqs against OpenSSL (default on, as in a stock
#                       liboqs). A static liboqs built with OpenSSL needs
#                       CGO_LDFLAGS=-lcrypto at link time.
#   --dist-build on|off pass -DOQS_DIST_BUILD (portable code with run-time CPU
#                       feature detection). Left to the liboqs default when
#                       not given.
#   -- CMAKE_ARGS...    extra arguments for the cmake configure step, for
#                       example -GNinja or -DCMAKE_OSX_DEPLOYMENT_TARGET=12.0
#
# The build is Falcon-512 only and installs just the library, plus the licence
# texts and notices of liboqs and of every third-party component it links in,
# under DIR/share/liboqs/, so that release archives can ship them.
#
# Runs under bash on Linux, macOS and the MSYS2 shell on Windows. Needs git,
# cmake and a C compiler on PATH.

set -euo pipefail

# The liboqs release the backend is tested against (also the Homebrew version
# used for local development).
LIBOQS_VERSION=0.16.0
# The commit that tag must resolve to; the build fails if the tag moves.
# 0.16.0 is a lightweight tag: git ls-remote shows no ^{} entry.
LIBOQS_COMMIT=5a1a854b0dc9f2141bdc771c555ee60c37950183
LIBOQS_REPO=https://github.com/open-quantum-safe/liboqs.git

usage() {
	sed -n '8,19p' "$0" | sed 's/^# \{0,1\}//' >&2
	exit 2
}

on_off() {
	case "$2" in
	on | ON) echo ON ;;
	off | OFF) echo OFF ;;
	*)
		echo "build-liboqs.sh: $1 expects on or off, got '$2'" >&2
		exit 2
		;;
	esac
}

prefix=""
openssl=ON
dist_build=""
while [ $# -gt 0 ]; do
	case "$1" in
	--prefix)
		[ $# -ge 2 ] || usage
		prefix=$2
		shift 2
		;;
	--openssl)
		[ $# -ge 2 ] || usage
		openssl=$(on_off "$1" "$2")
		shift 2
		;;
	--dist-build)
		[ $# -ge 2 ] || usage
		dist_build=$(on_off "$1" "$2")
		shift 2
		;;
	--)
		shift
		break
		;;
	-h | --help) usage ;;
	*)
		echo "build-liboqs.sh: unknown argument '$1'" >&2
		usage
		;;
	esac
done
[ -n "$prefix" ] || usage

work=$(mktemp -d "${TMPDIR:-/tmp}/liboqs.XXXXXX")
trap 'rm -rf "$work"' EXIT

git clone --quiet --depth 1 --branch "$LIBOQS_VERSION" "$LIBOQS_REPO" "$work/src"
head=$(git -C "$work/src" rev-parse HEAD)
if [ "$head" != "$LIBOQS_COMMIT" ]; then
	echo "build-liboqs.sh: liboqs tag $LIBOQS_VERSION resolves to $head, expected $LIBOQS_COMMIT" >&2
	exit 1
fi

cmake_args=(
	-DCMAKE_BUILD_TYPE=Release
	"-DCMAKE_INSTALL_PREFIX=$prefix"
	-DCMAKE_INSTALL_LIBDIR=lib
	-DOQS_MINIMAL_BUILD=SIG_falcon_512
	-DOQS_BUILD_ONLY_LIB=ON
	"-DOQS_USE_OPENSSL=$openssl"
)
if [ -n "$dist_build" ]; then
	cmake_args+=("-DOQS_DIST_BUILD=$dist_build")
fi

cmake -S "$work/src" -B "$work/build" "${cmake_args[@]}" "$@"
cmake --build "$work/build" --parallel
cmake --install "$work/build"

# Licence files for every component a Falcon-512 build links in, verified
# against the objects in liboqs.a on linux/amd64 and linux/arm64 (DIST_BUILD
# compiles every CPU variant). The full set is installed on every platform;
# scripts/tronctl-pq.sh package concatenates it into THIRD_PARTY_NOTICES.txt.
# The public-domain parts (AES-NI and ARMv8 AES, SHA-2) need no notice.
lic="$prefix/share/liboqs"
src="$work/src"
install -d "$lic"

# excerpt FILE FIRST LAST START END OUT copies the licence comment at lines
# FIRST..LAST of FILE to OUT. It fails unless the excerpt contains START and
# its last line contains END, so a liboqs bump that moves or grows the
# comment fails here instead of shipping a cut-off notice.
excerpt() {
	sed -n "$2,$3p" "$src/$1" >"$lic/$6"
	if ! grep -qF "$4" "$lic/$6"; then
		echo "build-liboqs.sh: $1 lines $2-$3 no longer hold the '$4' notice" >&2
		exit 1
	fi
	if ! tail -n 1 "$lic/$6" | grep -qF "$5"; then
		echo "build-liboqs.sh: $1 line $3 no longer ends the notice with '$5'" >&2
		exit 1
	fi
}

cp "$src/LICENSE.txt" "$lic/LICENSE.txt"
# The clean and AVX2 Falcon code carry the same MIT licence; install it once.
# Compared in bash: cmp (diffutils) is not in every MSYS2 base install.
if [ "$(cat "$src/src/sig/falcon/pqclean_falcon-512_clean/LICENSE")" != "$(cat "$src/src/sig/falcon/pqclean_falcon-512_avx2/LICENSE")" ]; then
	echo "build-liboqs.sh: the falcon-512 clean and avx2 licences differ; install both" >&2
	exit 1
fi
cp "$src/src/sig/falcon/pqclean_falcon-512_clean/LICENSE" "$lic/LICENSE-falcon-512.txt"
cp "$src/src/sig/falcon/pqclean_falcon-512_aarch64/LICENSE" "$lic/LICENSE-falcon-512-aarch64.txt"
# The standard Apache License 2.0 text (for the aarch64 Falcon code and the
# Apache-2.0 portions of src/common/common.c). MAYO is not built; its folder
# just holds the nearest copy of the text in the liboqs tree.
cp "$src/src/sig/mayo/pqmayo_mayo-1_opt/LICENSE" "$lic/LICENSE-Apache-2.0.txt"
grep -qF 'TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION' "$lic/LICENSE-Apache-2.0.txt" || {
	echo "build-liboqs.sh: $lic/LICENSE-Apache-2.0.txt is not the Apache License 2.0" >&2
	exit 1
}
excerpt src/common/aes/aes_c.c 3 28 'Copyright (c) 2016 Thomas Pornin' 'SOFTWARE.' NOTICE-aes_c-BearSSL.txt
excerpt src/common/sha3/avx512vl_low/KeccakP-1600-AVX512VL.S 1 3 'Copyright (c) 2025 Intel Corporation' 'SPDX-License-Identifier: MIT' NOTICE-sha3-avx512vl-Intel.txt
excerpt src/common/sha3/xkcp_low/KeccakP-1600/plain-64bits/KeccakP-1600-opt64.c 1 14 'has waived all copyright' 'publicdomain/zero/1.0' NOTICE-xkcp-CC0.txt
excerpt src/common/sha3/xkcp_low/KeccakP-1600/plain-64bits/brg_endian.h 1 26 'Brian Gladman' 'Changes for ARM' NOTICE-brg_endian.txt
excerpt src/common/sha3/xkcp_low/KeccakP-1600/avx2/KeccakP-1600-AVX2.S 1 17 'licensed under the CRYPTOGAMS license' 'Jorrit Jongma' NOTICE-KeccakP-1600-AVX2.txt
excerpt src/common/rand/rand_nist.c 1 11 'NIST-developed software is provided by NIST' 'not subject to copyright protection' NOTICE-rand_nist.txt
# The CRYPTOGAMS licence that KeccakP-1600-AVX2.S points to is not in the
# liboqs tree; this is the text of
# https://github.com/dot-asm/cryptogams/blob/master/LICENSE.
cat >"$lic/LICENSE-CRYPTOGAMS.txt" <<'CRYPTOGAMS'
Copyright (c) 2006, CRYPTOGAMS by <appro@openssl.org>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

      *	Redistributions of source code must retain copyright notices,
	this list of conditions and the following disclaimer.

      *	Redistributions in binary form must reproduce the above
	copyright notice, this list of conditions and the following
	disclaimer in the documentation and/or other materials
	provided with the distribution.

      *	Neither the name of the CRYPTOGAMS nor the names of its
	copyright holder and contributors may be used to endorse or
	promote products derived from this software without specific
	prior written permission.

ALTERNATIVELY, provided that this notice is retained in full, this
product may be distributed under the terms of the GNU General Public
License (GPL), in which case the provisions of the GPL apply INSTEAD OF
those given above.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDER AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
CRYPTOGAMS

echo "build-liboqs.sh: installed liboqs $LIBOQS_VERSION ($LIBOQS_COMMIT) to $prefix (OpenSSL $openssl${dist_build:+, DIST_BUILD $dist_build})"
