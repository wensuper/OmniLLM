#!/bin/sh
# Builds the web client (phones and tablets) into $1 (default dist/web),
# and with $2 also its manifest: the build and the sha256 of every file,
# which releases publish so anyone can check a served page against them
# (omnillm verify web):
# the page, Go's wasm_exec.js and the device code compiled to WebAssembly.
# The service worker's cache name is stamped with a hash of the files, so a
# new build replaces the old one as a whole.
set -eu
out=${1:-dist/web}
manifest=${2:-}
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
# Size matters on a phone's first open (docs/design.md §13.3). nojsonv2 keeps
# encoding/json on its classic code instead of the json/v2 engine Go 1.27
# uses by default: same behaviour, about 250 KB less after gzip. If a later
# Go drops the experiment, remove it here (the build says so).
GOEXPERIMENT=nojsonv2 GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$out/omni.wasm" ./cmd/omniweb
hash=$(cd "$out" && find . -type f ! -name sw.js | LC_ALL=C sort | xargs shasum -a 256 | shasum -a 256 | cut -c1-16)
sed "s/const VERSION = \"dev\";/const VERSION = \"$hash\";/" web/sw.js > "$out/sw.js"
# The same build stamp in the page and its script (app.js checks they match).
sed "s/<meta name=\"omnillm-build\" content=\"dev\">/<meta name=\"omnillm-build\" content=\"$hash\">/" web/index.html > "$out/index.html"
sed "s/^const BUILD = \"dev\";/const BUILD = \"$hash\";/" web/app.js > "$out/app.js"
grep -q "$hash" "$out/index.html" && grep -q "$hash" "$out/app.js" || { echo "build stamp missing" >&2; exit 1; }
if [ -n "$manifest" ]; then
  (cd "$out" && find . -type f | sed 's|^\./||' | LC_ALL=C sort | while read -r f; do
    printf '%s %s\n' "$(shasum -a 256 "$f" | cut -d' ' -f1)" "$f"
  done) | awk -v b="$hash" 'BEGIN { printf "{\n  \"build\": \"%s\",\n  \"files\": {", b } { printf "%s\n    \"%s\": \"%s\"", (NR > 1 ? "," : ""), $2, $1 } END { printf "\n  }\n}\n" }' > "$manifest"
fi
echo "web client built in $out ($hash)"
