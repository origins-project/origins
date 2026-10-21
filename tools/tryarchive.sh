#!/usr/bin/env bash
set -euo pipefail

# Tries a release archive on the system it was built for: unpacked, the binary says
# its version, starts a corpus and checks it, and serves the application rather than
# the placeholder a build without Node embeds.
#
#   tools/tryarchive.sh dist/origins_0.1.0_linux_amd64.tar.gz [0.1.0]
#
# With a version, the binary has to report that one.
archive="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
want="${2:-}"

tmp="$(mktemp -d)"
server=""
trap '[ -z "$server" ] || kill "$server" 2>/dev/null || true; rm -rf "$tmp"' EXIT
fail() { echo "tryarchive: $*" >&2; exit 1; }

case "$archive" in
*.zip) (cd "$tmp" && unzip -q "$archive") ;;
*) (cd "$tmp" && tar -xzf "$archive") ;;
esac
for f in LICENSE README.md THIRD_PARTY_NOTICES; do
	[ -s "$tmp/$f" ] || fail "the archive has no $f"
done
origins="$tmp/origins"
[ -f "$origins.exe" ] && origins="$origins.exe"
[ -x "$origins" ] || fail "the archive has no binary"

"$origins" version
first="$("$origins" version | head -n 1)"
case "$first" in
"origins dev" | "origins ") fail "the binary was built without its version: $first" ;;
esac
[ -z "$want" ] || [ "$first" = "origins $want" ] || fail "the binary says $first, not origins $want"

# A git of its own, so the commits init makes need nothing of the machine's.
export GIT_CONFIG_GLOBAL="$tmp/gitconfig" GIT_CONFIG_NOSYSTEM=1
git config --global user.name 'Rosa Bianchi'
git config --global user.email rosa@example.invalid
"$origins" init --name 'Rossi family' "$tmp/rossi"
"$origins" check "$tmp/rossi"

address=127.0.0.1:8191
"$origins" browse --listen "$address" --no-open "$tmp/rossi" &
server=$!
for _ in $(seq 60); do
	curl -fsS -o "$tmp/index.html" "http://$address/" 2>/dev/null && break
	sleep 0.5
done
[ -s "$tmp/index.html" ] || fail "browse served nothing at $address"
if grep -q 'built without its application' "$tmp/index.html"; then
	fail "browse serves the placeholder"
fi

script="$(grep -o 'assets/index-[A-Za-z0-9_-]*\.js' "$tmp/index.html" | head -n 1)"
[ -n "$script" ] || fail "the page names no script"
curl -fsS -o "$tmp/index.js" "http://$address/$script"
wasm="$(grep -o 'assets/chart-[A-Za-z0-9_-]*\.wasm' "$tmp/index.js" | head -n 1)"
[ -n "$wasm" ] || fail "the script names no chart"
curl -fsS -o "$tmp/chart.wasm" "http://$address/$wasm"
[ "$(head -c 4 "$tmp/chart.wasm" | od -An -tx1 | tr -d ' \n')" = 0061736d ] ||
	fail "the chart served is not WebAssembly"

echo "tryarchive: $(basename "$archive") is whole"
