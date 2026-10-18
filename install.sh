#!/bin/sh
# OmniLLM installer (macOS, Linux). No sudo, no questions.
#
#   curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh -o omnillm-install.sh
#   sh omnillm-install.sh
#
# 1. Downloads the latest release for this computer and checks its SHA-256
#    against checksums.txt; if cosign is installed, also checks that
#    checksums.txt was signed by this repository's release workflow.
#    No release yet: builds from source with Go (1.27.2 or newer).
# 2. Runs "omnillm install": copies itself to ~/.local/bin, starts the hub
#    as a background service for this user, opens the local console.
set -eu

REPO="wensuper/OmniLLM"
BASE="${OMNILLM_RELEASE_BASE:-https://github.com/$REPO/releases/latest/download}"
case "$BASE" in https://*) ;; *) printf 'omnillm-install: release address must be https\n' >&2; exit 1 ;; esac

say() { printf '%s\n' "$*"; }
fail() { printf 'omnillm-install: %s\n' "$*" >&2; exit 1; }

if [ "$(id -u)" = "0" ]; then
  fail "do not run as root; OmniLLM installs for the current user only"
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  darwin|linux) ;;
  *) fail "unsupported system: $os (macOS and Linux only for now)" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) fail "unsupported CPU: $arch" ;;
esac

work=$(mktemp -d "${TMPDIR:-/tmp}/omnillm.XXXXXX")
trap 'rm -rf "$work"' EXIT INT TERM

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

bin=""
asset="omnillm_${os}_${arch}.tar.gz"
if curl -fsSL --proto '=https' --tlsv1.2 -o "$work/checksums.txt" "$BASE/checksums.txt" 2>/dev/null; then
  say "Downloading $asset ..."
  curl -fsSL --proto '=https' --tlsv1.2 -o "$work/$asset" "$BASE/$asset" || fail "download failed: $asset"
  want=$(grep "  $asset\$" "$work/checksums.txt" | cut -d' ' -f1)
  [ -n "$want" ] || fail "checksums.txt has no entry for $asset"
  got=$(sha256 "$work/$asset")
  [ "$want" = "$got" ] || fail "checksum mismatch for $asset: refusing to install"
  say "Checksum OK."
  if command -v cosign >/dev/null 2>&1; then
    curl -fsSL --proto '=https' --tlsv1.2 -o "$work/checksums.txt.pem" "$BASE/checksums.txt.pem" || fail "missing signature certificate"
    curl -fsSL --proto '=https' --tlsv1.2 -o "$work/checksums.txt.sig" "$BASE/checksums.txt.sig" || fail "missing signature"
    cosign verify-blob \
      --certificate "$work/checksums.txt.pem" --signature "$work/checksums.txt.sig" \
      --certificate-identity-regexp "^https://github.com/$REPO/\\.github/workflows/release\\.yml@refs/tags/v" \
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
      "$work/checksums.txt" >/dev/null 2>&1 || fail "signature check failed: refusing to install"
    say "Signature OK (built by $REPO's release workflow)."
  else
    say "cosign not found: checked the checksum only."
  fi
  tar -xzf "$work/$asset" -C "$work" omnillm || fail "archive is damaged"
  bin="$work/omnillm"
else
  say "No release published yet: building from source."
  command -v go >/dev/null 2>&1 || fail "no release yet and Go is not installed. Install Go 1.27.2+ from https://go.dev/dl/ and run this again."
  GOBIN="$work" GOFLAGS=-trimpath go install "github.com/$REPO/cmd/omnillm@latest" || fail "build failed"
  bin="$work/omnillm"
fi

chmod 0755 "$bin"
"$bin" install "$@"
