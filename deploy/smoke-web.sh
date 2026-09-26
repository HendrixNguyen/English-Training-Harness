#!/bin/sh
# Post-deploy smoke check for the PWA (deploy/README.md "Smoke check").
#   deploy/smoke-web.sh <web-base-url>
# Exit 0 only if every check passes. Needs curl.
set -eu
web=${1:?usage: smoke-web.sh <web-base-url>}
fail=0
check() { if [ "$2" = "$3" ]; then echo "ok   $1: $3"; else echo "FAIL $1: expected '$2', got '$3'"; fail=1; fi; }
status() { curl -sS --max-time 10 -o /dev/null -w '%{http_code}' "$1"; }
cache_control() { curl -sS --max-time 10 -I "$1" | tr -d '\r' | awk -F': ' 'tolower($1)=="cache-control"{print $2}'; }

check "index" "200" "$(status "$web/")"
check "login return trip (/login?code&state)" "200" "$(status "$web/login?code=x&state=y")"
# no -L: the 2026-09-25 outage was a 308 to /login/ that dropped the OAuth query.
# On Pages this holds because the upload carries no 404.html (implicit SPA
# mode); in the Caddy image because of try_files.
check "spa fallback (/learn/abc)" "200" "$(status "$web/learn/abc")"
check "sw.js present" "200" "$(status "$web/sw.js")"
check "sw.js cache-control" "no-cache" "$(cache_control "$web/sw.js")"
check "manifest cache-control" "no-cache" "$(cache_control "$web/manifest.webmanifest")"
# One hashed asset: the first /_nuxt/ script index.html references.
asset=$(curl -sS --max-time 10 "$web/" | grep -o '/_nuxt/[^"]*\.js' | head -1)
check "hashed asset found" "1" "$([ -n "$asset" ] && echo 1 || echo 0)"
[ -n "$asset" ] && check "asset cache-control" "public, max-age=31536000, immutable" "$(cache_control "$web$asset")"

exit $fail
