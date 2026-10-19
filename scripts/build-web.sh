#!/bin/sh
# Builds the web client (phones and tablets) into $1 (default dist/web):
# the page, Go's wasm_exec.js and the device code compiled to WebAssembly.
# The service worker's cache name is stamped with a hash of the files, so a
# new build replaces the old one as a whole.
set -eu
out=${1:-dist/web}
cd "$(dirname "$0")/.."
rm -rf "$out"
mkdir -p "$out"
# jsQR 1.4.0 (Apache-2.0) from npm, checked: it runs with the page's rights.
want=$(cat web/vendor/jsqr.sha256)
got=$(shasum -a 256 web/vendor/jsqr.js | cut -d' ' -f1)
[ "$want" = "$got" ] || { echo "web/vendor/jsqr.js changed (sha256 $got)" >&2; exit 1; }
cp -R web/. "$out/"
rm -f "$out/vendor/jsqr.sha256"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/wasm_exec.js"
GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$out/omni.wasm" ./cmd/omniweb
hash=$(cd "$out" && find . -type f ! -name sw.js | LC_ALL=C sort | xargs shasum -a 256 | shasum -a 256 | cut -c1-16)
sed "s/const VERSION = \"dev\";/const VERSION = \"$hash\";/" web/sw.js > "$out/sw.js"
# The same build stamp in the page and its script (app.js checks they match).
sed "s/<meta name=\"omnillm-build\" content=\"dev\">/<meta name=\"omnillm-build\" content=\"$hash\">/" web/index.html > "$out/index.html"
sed "s/^const BUILD = \"dev\";/const BUILD = \"$hash\";/" web/app.js > "$out/app.js"
grep -q "$hash" "$out/index.html" && grep -q "$hash" "$out/app.js" || { echo "build stamp missing" >&2; exit 1; }
echo "web client built in $out ($hash)"
