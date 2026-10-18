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
# Fixed on purpose: no environment variable can point this at another
# server (a mirror could serve its own archive and matching checksums).
BASE="https://github.com/$REPO/releases/latest/download"

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
# 200: a release exists. 404: none published yet (build from source).
# Anything else (network error, blocked request): stop; never fall back to
# an unverified build because a download failed.
code=$(curl -sSL --proto '=https' --tlsv1.2 -o "$work/checksums.txt" -w '%{http_code}' "$BASE/checksums.txt" 2>/dev/null || true)
case "$code" in
  200|404) ;;
  *) fail "could not reach GitHub releases (HTTP ${code:-none}); check the network and run this again" ;;
esac
if [ "$code" = "200" ]; then
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
      --certificate-identity-regexp "^https://github\\.com/wensuper/OmniLLM/\\.github/workflows/release\\.yml@refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z.]+)?\$" \
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
      "$work/checksums.txt" >/dev/null 2>&1 || fail "signature check failed: refusing to install"
    say "Signature OK (built by $REPO's release workflow)."
  else
    say "cosign not found: checked the checksum only."
  fi
  tar -xzf "$work/$asset" -C "$work" omnillm || fail "archive is damaged"
  bin="$work/omnillm"
else
  say "No release published yet: building from source (not signature-checked)."
  command -v go >/dev/null 2>&1 || fail "no release yet and Go is not installed. Install Go 1.27.2+ from https://go.dev/dl/ and run this again."
  GOBIN="$work" GOFLAGS=-trimpath go install "github.com/$REPO/cmd/omnillm@latest" || fail "build failed"
  bin="$work/omnillm"
fi

chmod 0755 "$bin"
"$bin" install "$@"
