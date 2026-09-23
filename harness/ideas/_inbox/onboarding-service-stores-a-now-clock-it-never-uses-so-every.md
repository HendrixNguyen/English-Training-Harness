---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# onboarding Service stores a now clock it never uses so every caller passes a dead dependency

## Why
`onboarding.Service` declares a `now func() time.Time` field
(`service.go:29`), `NewService` takes it as its sixth parameter, defaults a nil
to `time.Now` (`service.go:34-37`) and assigns it (`service.go:37`) — and
nothing ever reads `s.now`. `grep -rn '\.now\b' backend/internal/onboarding/`
returns only the constructor lines. The slice has no time-dependent logic:
`created_at` comes from Postgres `CURRENT_TIMESTAMP`, the quiz TTL comes from
`store.PlacementQuizTTL`.

The cost is small but real. `cmd/api/main.go:123` passes `time.Now` and every
test passes `fixedClock(sept22)` (`service_test.go:34`, `handler_test.go:86`),
which reads as though the tests control this service's clock — they do not, and
a reader trying to work out what `sept22` pins will find nothing. A dead
constructor parameter also survives refactors by being copied forward.

## Expected output
- Either `Service.now` is removed along with the `NewService` parameter and the
  `fixedClock` arguments in `main.go` and the tests, or the field is actually
  used by something time-dependent and a test pins that use.
- `NewService`'s signature carries only dependencies the service needs.

## Evidence
- Plan: `harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md`.
- Declared: `backend/internal/onboarding/service.go:29`.
- Defaulted and assigned, never read: `backend/internal/onboarding/service.go:34-37`.
- `grep -rn '\.now\b|now func\(\)|now:' backend/internal/onboarding/` →
  only `service.go:33` and `service.go:37`.
- Callers that supply it: `backend/cmd/api/main.go:123` (`time.Now`),
  `backend/internal/onboarding/service_test.go:34` and
  `backend/internal/onboarding/handler_test.go:86` (`fixedClock(sept22)`).
