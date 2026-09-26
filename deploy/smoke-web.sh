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
# SMOKE_WEB_SPA_WARN=1 downgrades only this check to a warning: Cloudflare Pages
# serves the generated 404.html before the _redirects splat, so deep links
# answer 404 there until that fix lands (harness inbox: pages-ignores-the-
# redirects-spa-rewrite-while-404-html-exist). The deploy workflow sets it; the
# docker-images CI job does not, so the Caddy image is still held to 200.
spa=$(status "$web/learn/abc")
if [ "${SMOKE_WEB_SPA_WARN:-}" = "1" ] && [ "$spa" != "200" ]; then
  echo "WARN spa fallback (/learn/abc): expected '200', got '$spa' (SMOKE_WEB_SPA_WARN=1)"
else
  check "spa fallback (/learn/abc)" "200" "$spa"
fi
check "sw.js present" "200" "$(status "$web/sw.js")"
check "sw.js cache-control" "no-cache" "$(cache_control "$web/sw.js")"
check "manifest cache-control" "no-cache" "$(cache_control "$web/manifest.webmanifest")"
# One hashed asset: the first /_nuxt/ script index.html references.
asset=$(curl -sS --max-time 10 "$web/" | grep -o '/_nuxt/[^"]*\.js' | head -1)
check "hashed asset found" "1" "$([ -n "$asset" ] && echo 1 || echo 0)"
[ -n "$asset" ] && check "asset cache-control" "public, max-age=31536000, immutable" "$(cache_control "$web$asset")"

# The app shell must never be cached by heuristics (a redeploy would strand an old index.html).
check "index cache-control" "no-cache" "$(cache_control "$web/")"
# Caddy only (SMOKE_WEB_ASSET_404=1): a chunk that no longer exists is a 404, never index.html
# with a one-year header. Pages' implicit SPA mode answers 200 for any miss, so it is not checked there.
if [ "${SMOKE_WEB_ASSET_404:-}" = "1" ]; then
  check "missing asset is 404 (/_nuxt/does-not-exist.js)" "404" "$(status "$web/_nuxt/does-not-exist.js")"
fi

exit $fail
