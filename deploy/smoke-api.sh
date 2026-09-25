#!/bin/sh
# Post-deploy smoke check for the API (deploy/README.md "Smoke check").
#   deploy/smoke-api.sh <api-base-url> <frontend-origin>
# e.g. deploy/smoke-api.sh https://api.example.com https://app.example.com
# Exit 0 only if every check passes. Needs curl.
set -eu
api=${1:?usage: smoke-api.sh <api-base-url> <frontend-origin>}
origin=${2:?usage: smoke-api.sh <api-base-url> <frontend-origin>}
fail=0
check() { # check <label> <expected> <actual>
  if [ "$2" = "$3" ]; then echo "ok   $1: $3"; else echo "FAIL $1: expected '$2', got '$3'"; fail=1; fi
}

# 1. /healthz answers 200 with every dependency ok (internal/health).
body=$(curl -sS --max-time 10 -w '\n%{http_code}' "$api/healthz")
check "healthz status" "200" "$(printf '%s' "$body" | tail -1)"
check "healthz body" "1" "$(printf '%s' "$body" | head -1 | grep -c '"status":"ok"')"

# 2. POST /api/v1/auth/google is routed: an empty body is a 400 invalid_request
#    (the real code exchange can only be driven by a browser through the PWA).
code=$(curl -sS --max-time 10 -o /dev/null -w '%{http_code}' -X POST \
  -H 'Content-Type: application/json' -d '{}' "$api/api/v1/auth/google")
check "auth/google empty body" "400" "$code"

# 3. CORS: a preflight from FRONTEND_ORIGIN is 204 and echoes the origin …
hdr=$(curl -sS --max-time 10 -i -X OPTIONS "$api/api/v1/auth/google" \
  -H "Origin: $origin" -H 'Access-Control-Request-Method: POST' -o - )
check "preflight status" "204" "$(printf '%s' "$hdr" | head -1 | awk '{print $2}')"
check "preflight allow-origin" "$origin" "$(printf '%s' "$hdr" | tr -d '\r' | awk -F': ' 'tolower($1)=="access-control-allow-origin"{print $2}')"

# 4. … and a preflight from anywhere else is refused (403), so the allow-list is exact.
code=$(curl -sS --max-time 10 -o /dev/null -w '%{http_code}' -X OPTIONS "$api/api/v1/auth/google" \
  -H 'Origin: https://evil.example' -H 'Access-Control-Request-Method: POST')
check "preflight foreign origin" "403" "$code"

exit $fail
