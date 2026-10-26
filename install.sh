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
  fail "$(t "不要用 root 或 sudo 运行：OmniLLM 只为当前用户安装，去掉 sudo 再运行一次。" "Don’t run this as root or with sudo: OmniLLM installs for the current user only. Run it again without sudo.")"
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  darwin|linux) ;;
  *) fail "$(t "暂不支持这个系统：${os}（目前只支持 macOS 和 Linux）。" "This system isn’t supported yet: $os (macOS and Linux only for now).")" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) fail "$(t "暂不支持这种 CPU：${arch}。" "This CPU isn’t supported yet: $arch.")" ;;
esac

work=$(mktemp -d "${TMPDIR:-/tmp}/omnillm.XXXXXX")
trap 'rm -rf "$work"' EXIT
trap 'exit 130' INT TERM

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
code=$(curl -sSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o "$work/checksums.txt" -w '%{http_code}' "$BASE/checksums.txt" 2>/dev/null || true)
case "$code" in
  200|404) ;;
  *) fail "$(t "连不上 GitHub（HTTP ${code:-none}），请检查网络后再运行一次。" "Couldn’t reach GitHub (HTTP ${code:-none}). Check the network and run this again.")" ;;
esac
if [ "$code" = "404" ]; then
  # No stable release: use the newest pre-release, if any. Only the tag
  # name is taken from the API answer, and it must look like a version;
  # downloads still come from this repository's release files.
  # An update started by the hub names the version it found; otherwise
  # the API, and when GitHub limits this address (60 API calls an hour),
  # the release feed, which isn't limited that way.
  tag=""
  if [ -n "${OMNILLM_UPDATE_TO:-}" ]; then
    tag="v${OMNILLM_UPDATE_TO#v}"
  elif api=$(curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 -H 'Accept: application/vnd.github+json' \
    "https://api.github.com/repos/$REPO/releases?per_page=30" 2>/dev/null); then
    # The API lists releases in no version order (alpha.10 comes after
    # alpha.5), so pick the highest version: numbers compare as numbers.
    tag=$(printf '%s' "$api" | grep -o '"tag_name": *"[^"]*"' | sed 's/.*"\([^"]*\)"$/\1/' | newest)
  else
    feed=$(curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 "https://github.com/$REPO/releases.atom" 2>/dev/null) || fail "$(t "连不上 GitHub，请检查网络后再运行一次。" "Couldn’t reach GitHub. Check the network and run this again.")"
    tag=$(printf '%s' "$feed" | grep -o '/releases/tag/v[^"]*"' | sed 's|.*/tag/||; s|"$||' | newest)
  fi
  if [ -n "$tag" ]; then
    printf '%s' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$' || fail "$(t "GitHub 给出的版本名不对，没有安装。" "GitHub returned an unexpected version name, so nothing was installed.")"
    BASE="https://github.com/$REPO/releases/download/$tag"
    say "$(t "最新版本是 ${tag#v}（预发布版）。" "The newest version is ${tag#v} (a pre-release).")"
    code=$(curl -sSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o "$work/checksums.txt" -w '%{http_code}' "$BASE/checksums.txt" 2>/dev/null || true)
    [ "$code" = "200" ] || fail "$(t "版本 ${tag} 缺少校验文件（HTTP ${code:-none}），没有安装。" "Version $tag has no checksum file (HTTP ${code:-none}), so nothing was installed.")"
  fi
fi
if [ "$code" = "200" ]; then
  case "$os" in darwin) osname=macOS ;; *) osname=Linux ;; esac
  say "$(t "正在下载 OmniLLM（${osname} ${arch}）……" "Downloading OmniLLM ($osname $arch)…")"
  curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o "$work/$asset" "$BASE/$asset" || fail "$(t "下载没有完成，请检查网络后再运行一次。" "The download didn’t finish. Check the network and run this again.")"
  want=$(grep "  $asset\$" "$work/checksums.txt" | cut -d' ' -f1)
  [ -n "$want" ] || fail "$(t "校验文件里没有这个系统的程序，没有安装。" "The checksum file has no program for this system, so nothing was installed.")"
  got=$(sha256 "$work/$asset")
  [ "$want" = "$got" ] || fail "$(t "下载的文件和发布的不一致，没有安装。" "The downloaded file doesn’t match the published one, so nothing was installed.")"
  say "$(t "下载已核对无误。" "Download verified.")"
  if command -v cosign >/dev/null 2>&1; then
    if curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o "$work/checksums.txt.sigstore.json" "$BASE/checksums.txt.sigstore.json" 2>/dev/null; then
      sigargs="--bundle $work/checksums.txt.sigstore.json"
    else
      curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o "$work/checksums.txt.pem" "$BASE/checksums.txt.pem" || fail "$(t "缺少签名证书，没有安装。" "The signature certificate is missing, so nothing was installed.")"
      curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o "$work/checksums.txt.sig" "$BASE/checksums.txt.sig" || fail "$(t "缺少签名，没有安装。" "The signature is missing, so nothing was installed.")"
      sigargs="--certificate $work/checksums.txt.pem --signature $work/checksums.txt.sig"
    fi
    # shellcheck disable=SC2086
    cosign verify-blob $sigargs \
      --certificate-identity-regexp "^https://github\\.com/wensuper/OmniLLM/\\.github/workflows/release\\.yml@refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z.]+)?\$" \
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
      "$work/checksums.txt" >/dev/null 2>&1 || fail "$(t "签名核对没有通过，没有安装。" "The signature check failed, so nothing was installed.")"
    say "$(t "签名也核对通过（由 ${REPO} 的发布流程构建）。" "Signature OK (built by ${REPO}’s release workflow).")"
  fi
  tar -xzf "$work/$asset" -C "$work" omnillm || fail "$(t "下载的压缩包损坏了，请再运行一次。" "The archive is damaged. Run this again.")"
  bin="$work/omnillm"
else
  say "$(t "还没有发布的版本：从源代码编译（没有签名可核对）。" "No release published yet: building from source (not signature-checked).")"
  command -v go >/dev/null 2>&1 || fail "$(t "还没有发布的版本，这台电脑上也没有 Go。从 https://go.dev/dl/ 装 Go 1.27.2 或更新的版本后再运行一次。" "No release yet, and Go isn’t installed. Install Go 1.27.2 or newer from https://go.dev/dl/ and run this again.")"
  GOBIN="$work" GOFLAGS=-trimpath go install "github.com/$REPO/cmd/omnillm@latest" || fail "$(t "编译失败。看上面的报错；如果是网络问题，换个网络再运行一次。" "The build failed. See the errors above; if it’s the network, try another network and run this again.")"
  bin="$work/omnillm"
fi

chmod 0755 "$bin"
new=$("$bin" version 2>/dev/null) || fail "$(t "下载的程序在这台电脑上运行不了，没有安装。" "The downloaded program doesn’t run on this computer, so nothing was installed.")"
new=${new#omnillm }
old=""
if [ -x "$HOME/.local/bin/omnillm" ]; then
  old=$("$HOME/.local/bin/omnillm" version 2>/dev/null) || old=""
  old=${old#omnillm }
fi
if [ -n "$old" ] && [ "$old" = "$new" ]; then
  say "$(t "已经是 ${new}，重新安装……" "Already on ${new}; reinstalling…")"
elif [ -n "$old" ]; then
  say "$(t "正在从 ${old} 更新到 ${new}……" "Updating ${old} → ${new}…")"
fi
"$bin" install "$@"
}

main "$@"
