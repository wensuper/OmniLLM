#!/bin/sh
# OmniLLM installer (macOS, Linux). No sudo, no questions. Installs, or
# updates what is installed (settings are kept).
#
#   curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh -s -- --device   # a server that only uses a hub
#
# Everything runs from main at the very end, so a download cut off halfway
# runs nothing.
#
# 1. Downloads the latest release for this computer and checks its SHA-256
#    against checksums.txt; if cosign is installed, also checks that
#    checksums.txt was signed by this repository's release workflow.
#    No release yet: builds from source with Go (1.27.2 or newer).
# 2. Runs "omnillm install": copies itself to ~/.local/bin, starts the hub
#    as a background service for this user, opens the local console.
set -eu

main() {

REPO="wensuper/OmniLLM"
# Fixed on purpose: no environment variable can point this at another
# server (a mirror could serve its own archive and matching checksums).
BASE="https://github.com/$REPO/releases/latest/download"

# Messages follow the person's language (the same variables omnillm reads).
case "${OMNILLM_LANG:-${LC_ALL:-${LANG:-}}}" in zh*|ZH*) zh=1 ;; *) zh="" ;; esac
t() { if [ -n "$zh" ]; then printf '%s' "$1"; else printf '%s' "$2"; fi; }
say() { printf '%s\n' "$*"; }
fail() { printf 'omnillm-install: %s\n' "$*" >&2; exit 1; }

if [ "$(id -u)" = "0" ]; then
  fail "$(t "不要用 root 运行（do not run as root）：OmniLLM 只为当前用户安装，去掉 sudo 再运行。" "do not run as root; OmniLLM installs for the current user only")"
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  darwin|linux) ;;
  *) fail "$(t "暂不支持这个系统：${os}（目前只支持 macOS 和 Linux）" "unsupported system: $os (macOS and Linux only for now)")" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) fail "$(t "暂不支持这种 CPU：$arch" "unsupported CPU: $arch")" ;;
esac

work=$(mktemp -d "${TMPDIR:-/tmp}/omnillm.XXXXXX")
trap 'rm -rf "$work"' EXIT INT TERM

# newest prints the highest of the version tags on stdin (vMAJOR.MINOR.PATCH
# with an optional -pre.release.N); anything else is ignored.
newest() {
  awk '
    /^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$/ {
      t = $0; v = substr(t, 2); pre = "~"   # a release sorts after its pre-releases
      i = index(v, "-"); if (i) { pre = substr(v, i + 1); v = substr(v, 1, i - 1) }
      n = split(v, c, "."); k = ""
      for (j = 1; j <= n; j++) k = k sprintf("%09d.", c[j])
      if (pre != "~") { m = split(pre, p, "."); pre = ""
        for (j = 1; j <= m; j++) pre = pre ((p[j] ~ /^[0-9]+$/) ? sprintf("%09d", p[j]) : p[j]) "." }
      print k "|" pre "\t" t
    }' | LC_ALL=C sort | tail -n 1 | cut -f 2
}

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
  *) fail "$(t "连不上 GitHub 的发布页（HTTP ${code:-none}），请检查网络后再运行一次" "could not reach GitHub releases (HTTP ${code:-none}); check the network and run this again")" ;;
esac
if [ "$code" = "404" ]; then
  # No stable release: use the newest pre-release, if any. Only the tag
  # name is taken from the API answer, and it must look like a version;
  # downloads still come from this repository's release files.
  api=$(curl -fsSL --proto '=https' --tlsv1.2 -H 'Accept: application/vnd.github+json' \
    "https://api.github.com/repos/$REPO/releases?per_page=30" 2>/dev/null) || fail "$(t "连不上 GitHub（API），请检查网络后再运行一次" "could not reach GitHub (API); check the network and run this again")"
  # The API lists releases in no version order (alpha.10 comes after
  # alpha.5), so pick the highest version: numbers compare as numbers.
  tag=$(printf '%s' "$api" | grep -o '"tag_name": *"[^"]*"' | sed 's/.*"\([^"]*\)"$/\1/' | newest)
  if [ -n "$tag" ]; then
    printf '%s' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$' || fail "unexpected release name: refusing"
    BASE="https://github.com/$REPO/releases/download/$tag"
    say "$(t "使用预发布版 ${tag}。" "Using pre-release $tag.")"
    code=$(curl -sSL --proto '=https' --tlsv1.2 -o "$work/checksums.txt" -w '%{http_code}' "$BASE/checksums.txt" 2>/dev/null || true)
    [ "$code" = "200" ] || fail "release $tag has no checksums.txt (HTTP ${code:-none}); refusing"
  fi
fi
if [ "$code" = "200" ]; then
  say "$(t "正在下载 $asset ……" "Downloading $asset ...")"
  curl -fsSL --proto '=https' --tlsv1.2 -o "$work/$asset" "$BASE/$asset" || fail "download failed: $asset"
  want=$(grep "  $asset\$" "$work/checksums.txt" | cut -d' ' -f1)
  [ -n "$want" ] || fail "checksums.txt has no entry for $asset"
  got=$(sha256 "$work/$asset")
  [ "$want" = "$got" ] || fail "$(t "$asset 的校验和对不上（checksum mismatch），不安装" "checksum mismatch for $asset: refusing to install")"
  say "$(t "校验和核对通过。" "Checksum OK.")"
  if command -v cosign >/dev/null 2>&1; then
    if curl -fsSL --proto '=https' --tlsv1.2 -o "$work/checksums.txt.sigstore.json" "$BASE/checksums.txt.sigstore.json" 2>/dev/null; then
      sigargs="--bundle $work/checksums.txt.sigstore.json"
    else
      curl -fsSL --proto '=https' --tlsv1.2 -o "$work/checksums.txt.pem" "$BASE/checksums.txt.pem" || fail "missing signature certificate"
      curl -fsSL --proto '=https' --tlsv1.2 -o "$work/checksums.txt.sig" "$BASE/checksums.txt.sig" || fail "missing signature"
      sigargs="--certificate $work/checksums.txt.pem --signature $work/checksums.txt.sig"
    fi
    # shellcheck disable=SC2086
    cosign verify-blob $sigargs \
      --certificate-identity-regexp "^https://github\\.com/wensuper/OmniLLM/\\.github/workflows/release\\.yml@refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z.]+)?\$" \
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
      "$work/checksums.txt" >/dev/null 2>&1 || fail "$(t "签名核对没有通过（signature check failed），不安装" "signature check failed: refusing to install")"
    say "$(t "签名核对通过（由 $REPO 的发布流程构建）。" "Signature OK (built by $REPO's release workflow).")"
  else
    say "$(t "没有找到 cosign：只核对了校验和（checked the checksum only）。" "cosign not found: checked the checksum only.")"
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
new=$("$bin" version 2>/dev/null | sed 's/^omnillm //') || new=""
old=""
[ -x "$HOME/.local/bin/omnillm" ] && old=$("$HOME/.local/bin/omnillm" version 2>/dev/null | sed 's/^omnillm //') || true
if [ -n "$old" ] && [ "$old" = "$new" ]; then
  say "$(t "已经是最新版本 ${new}，重新安装一遍（设置都保留）。" "Already on the newest version ($new); installing it again (settings are kept).")"
elif [ -n "$old" ]; then
  say "$(t "正在从 ${old} 更新到 ${new}（设置都保留）。" "Updating from $old to $new (settings are kept).")"
fi
"$bin" install "$@"
}

main "$@"
