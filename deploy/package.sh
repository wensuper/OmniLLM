#!/bin/sh
# Run on a Mac: fetch release TAG's Linux relay program and web client from
# GitHub, check them against the signed checksums, and put them in a folder
# ready to copy to the server.  Usage: deploy/package.sh v0.2.0-alpha.1 [dir]
set -eu
tag=$1
out=${2:-"$(mktemp -d)/omnillm-$tag"}
arch=${ARCH:-amd64}
here=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$out"
cp "$here/omnillm-relay.service" "$here/install-server.sh" "$out/"
cd "$out"
gh release download "$tag" -R wensuper/OmniLLM --clobber \
  -p checksums.txt -p checksums.txt.sigstore.json -p "omnillm_linux_${arch}.tar.gz" -p omnillm-web.tar.gz
if command -v cosign >/dev/null 2>&1; then
  cosign verify-blob --bundle checksums.txt.sigstore.json \
    --certificate-identity-regexp '^https://github\.com/wensuper/OmniLLM/\.github/workflows/release\.yml@refs/tags/v' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com checksums.txt
else
  echo "cosign not installed: checking checksums only" >&2
fi
grep -E " (omnillm_linux_${arch}\.tar\.gz|omnillm-web\.tar\.gz)\$" checksums.txt > want.txt
[ "$(wc -l < want.txt)" -eq 2 ] || { echo "release lacks a file" >&2; exit 1; }
shasum -a 256 -c want.txt
mkdir -p bin web
tar -xzf "omnillm_linux_${arch}.tar.gz" -C bin omnillm
tar -xzf omnillm-web.tar.gz -C web
echo "$tag" > TAG
echo "$out"
