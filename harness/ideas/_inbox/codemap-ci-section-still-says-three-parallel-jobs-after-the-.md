---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# CODEMAP CI section still says three parallel jobs after the frontend job made it four

## Why
`harness/CODEMAP.md` is the map every role reads before exploring code, so a count that contradicts the list immediately below it costs a reader a verification round-trip. This slice appended a fourth job to `.github/workflows/ci.yml` and correctly added a fourth bullet to CODEMAP's CI section, but left the section's opening sentence at "Three parallel GitHub Actions jobs". AGENTS.md makes the pushed branch's CI run a review gate, so the number of jobs a reviewer should expect to be green is load-bearing, not cosmetic — a reviewer trusting the sentence would accept three green checks where four are required.

## Expected output
`harness/CODEMAP.md:28` opens "Four parallel GitHub Actions jobs, each capped at `timeout-minutes: 10`, …". One word.

## Evidence
- Plan: `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` (Task 15, Step 3).
- `harness/CODEMAP.md:28` — "Three parallel GitHub Actions jobs…"; the bullet list at lines 30-34 enumerates `backend-unit`, `backend-integration`, `harness-tooling` and `frontend`.
- `.github/workflows/ci.yml` on the branch defines four jobs (`frontend` at line 122).
- The reviewer prepared this one-word fix on the branch but the sandbox denied the commit; filed here instead of left unrecorded.
