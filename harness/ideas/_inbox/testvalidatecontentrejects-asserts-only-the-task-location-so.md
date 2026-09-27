---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# TestValidateContentRejects asserts only the task location so a row can pass for the wrong rule

## Why
`TestValidateContentRejects` (the branch's rejection table) checks only that the error wraps `ErrInvalidRoadmap` and contains `module 1 day 1 task 1`. It never checks which rule fired. This is the same weakness the reviewer flagged on `TestParseRoadmapRejects` (since fixed by that branch's amend). A row such as `"option key outside A..D"` is actually rejected by the "missing option D" check, and `"three options"` by the options count. If a check were deleted, the row could still pass through another rule, so the table is weaker than its names claim.

## Expected output
Each row carries a `want` substring of its `invalid(...)` message (for example `"answer \"b\" is not one of its options"`, `"has 2 words, want 5..8"`, `"passage is"`, `"needs a unique id"`, `"has no explanation"`), asserted with `strings.Contains`. Deleting any single check in `validateQuestions`/`validateContent` then fails at least one row.

## Evidence
- Plan `harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md`, Task 2.
- `backend/internal/airouter/content_test.go` `TestValidateContentRejects` loop: only `errors.Is` + `strings.Contains(err.Error(), "module 1 day 1 task 1")`.
- Precedent: `harness/ideas/_inbox/testparseroadmaprejects-checks-only-that-an-error-occurred-s.md`.
