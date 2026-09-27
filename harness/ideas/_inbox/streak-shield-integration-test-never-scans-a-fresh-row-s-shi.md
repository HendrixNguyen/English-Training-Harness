---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Streak-shield integration test never scans a fresh row's shields and races no shielded PenaliseMiss

## Why
The only real-Postgres check of a freshly ensured pet row never asserts `Shields == 0` / `LastShieldUsedOn == nil`, so the non-COALESCE `p.shields` scan and `to_char(NULL)` on `last_shield_used_on` are never exercised end-to-end; every other shield assertion seeds via a raw `UPDATE` or the fake repo. The shielded branch of `penaliseMissSQL` is only run sequentially, although `TestIntegrationVerdictWritesAreConditional` races the unshielded one. `TestSpec8Constants` does not pin `MaxShields`/`ShieldEveryDays`.

## Expected output
- `TestIntegrationEnsureCreatesExactlyOnePetRow` asserts `Shields == 0` and `LastShieldUsedOn == nil`.
- A concurrent test proves exactly one of N `PenaliseMiss` calls on a shielded row spends the shield.
- `TestSpec8Constants` asserts `MaxShields == 2`, `ShieldEveryDays == 7`.

## Evidence
- Plan `harness/plans/2026-09-25-pet-streak-shield-earned-by-target-days.md`, branch `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days`.
- `backend/internal/pet/integration_test.go:79`, `backend/internal/pet/engine_test.go:133` on that branch.
- Found by the test-gap pass of the 2026-09-27 daily review; test honesty only, not a blocker.
