---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# ParseOrigins accepts FRONTEND_ORIGIN entries no browser sends, so boot succeeds and the PWA is silently blocked

## Why
`middleware.ParseOrigins` is meant to refuse a malformed `FRONTEND_ORIGIN` at boot: "boot refuses a malformed list" (CODEMAP), because "a CORS middleware with no origins would silently block the PWA" (its doc comment). It still accepts entries that can never equal a browser's `Origin` header, and stores them verbatim:

- **Explicit default port.** `https://app.example.com:443` and `http://localhost:80` are accepted with the port kept (`url.Parse` puts `app.example.com:443` in `Host`). Browsers omit the default port from `Origin`, so the exact-match lookup never hits. The live binary returns `403` when the port does not match.
- **Wildcard host.** `https://*.up.railway.app` is accepted as the literal host `*.up.railway.app`. Railway preview and PR environments are exactly where an operator will reach for a wildcard, and the match never succeeds.
- **Trailing-dot FQDN.** `https://app.example.com.` is accepted and never matches.

In each case boot succeeds and logs `cors: allowing [...]`, which looks healthy. Every browser call from the PWA then fails its preflight with `403`. That is the silent total outage this package exists to prevent. The operator gets no error and no warning.

On tests: the exact-map match is correct today. The review drove `null`, `https://app.example.com.evil.com`, `https://evilapp.example.com`, `http://app.example.com`, the upper-case form and the trailing-slash form against the live binary, and all of them got `403`. But `cors_test.go` covers only one disallowed origin (`https://evil.example`). A later change to support previews by suffix or prefix matching would not turn anything red. On the same lines, `ParseOrigins`'s scheme check repeats itself (`u.Scheme != "http" && u.Scheme != "https" && strings.ToLower(u.Scheme) != "http" && …`). `url.Parse` already lower-cases the scheme, and the plan asked for this condition to be simplified.

## Expected output
- `ParseOrigins` returns `ErrBadOrigin` for an entry with a default port that does not belong (`:443` on https, `:80` on http), for any `*` in the host, and for a trailing-dot host. The alternative is to normalise the default port away; pick one and document it. Boot then fails loudly instead of serving a `403` to every browser request.
- `TestParseOrigins` gains those rows.
- `cors_test.go` gains a table of lookalike origins that must get a `403` preflight and no `Access-Control-Allow-Origin`: `null`, a suffix (`https://app.example.com.evil.com`), a prefix (`https://evilapp.example.com`), a scheme downgrade (`http://…`), an explicit port, and a trailing slash.
- The scheme condition is simplified to `u.Scheme != "http" && u.Scheme != "https"`, and the `HTTPS://App.Example.com` row stays green.

## Evidence
- Reviewing `harness/plans/2026-09-24-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md` (Task 2; the plan note under Step 3 asks for the condition to be simplified).
- `backend/internal/middleware/cors.go`: the `ParseOrigins` loop (the `url.Parse` validation condition and `out = append(out, …Host)`).
- `go run` scratch check of `url.Parse` (go1.27.1): `"https://app.example.com:443"` → `host="app.example.com:443"`; `"https://*.up.railway.app"` → `host="*.up.railway.app"`; `"https://app.example.com."` → `host="app.example.com."`. None of these has a path or an error, so all pass the validation.
- Live, on the branch binary with `FRONTEND_ORIGIN=https://app.example.com`, a preflight with `Origin: https://app.example.com:443` returns `403`, which shows the lookup is exact on the port. The eight lookalike origins listed under *Why* all return `403`.
- `backend/internal/middleware/cors_test.go`: `TestPreflightFromAnotherOriginIs403` and `TestAnActualRequestFromAnotherOriginPassesWithoutCORSHeaders` are the only disallowed-origin tests, and both use `https://evil.example`.
