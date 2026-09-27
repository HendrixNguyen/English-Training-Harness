---
idea: harness/ideas/_inbox/codemap-does-not-document-the-config-and-health-packages-and.md
status: approved
priority: low
merged: false
---
# CODEMAP does not document the config and health packages and still says three CI jobs — Plan

**Idea:** `harness/ideas/_inbox/codemap-does-not-document-the-config-and-health-packages-and.md`
**Goal:** `harness/CODEMAP.md` gains a `**config**` bullet that matches `backend/internal/config/config.go`, and its CI section states the real job count.

**Current state (checked 2026-09-27 on `main`):**
- The `**health**` bullet now exists (CODEMAP line 11) and matches the idea, so that half is already done.
- No `**config**` bullet exists.
- The CI section (line 37) opens "Three parallel GitHub Actions jobs", while it lists five (`backend-unit`, `backend-integration`, `harness-tooling`, `frontend`, `docker-images`), and `.github/workflows/ci.yml` defines those five.

**Scope / files:** `harness/CODEMAP.md` only. No code. Documentation-only, with no test to write. The executor still verifies every claim against the source.

## Tasks

### Task 1: `**config**` bullet
Files: `harness/CODEMAP.md`, "Backend packages" list. Insert the bullet directly before `**health**`.
1. Read `backend/internal/config/config.go` (`Load`, the `Config` fields and the constants) and `config_test.go`. Write one bullet in the same dense style as its neighbours:
   - `Load()` reads, from the process env: `DATABASE_URL`, `REDIS_URL`, `PORT`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `JWT_SECRET` (≥ `MinJWTSecretBytes` = 32), `ENCRYPTION_SECRET_KEY` (64 hex → 32 bytes via `secrets.ParseHexKey`, required), `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` (optional as a pair; without both the notify worker does not start), `VAPID_SUBJECT` (default `DefaultVAPIDSubject`), `GIN_MODE`, `FRONTEND_ORIGIN` (default `DefaultFrontendOrigin`).
   - For each variable, state exactly what `Load` does with it: which are required, which are defaulted, and what each validation checks. Take every one from the code, not from this list. If the code disagrees with this list, the code wins.
   - Name the test file.
2. The `airouter` provider variables are read by `airouter`, not `config`. Check with `grep -n Getenv backend/internal/airouter/*.go`, and if so say it in one clause, so nobody looks for them in `config`.
3. Commit: `harness: CODEMAP — config bullet`.

### Task 2: CI job count
Files: `harness/CODEMAP.md`, CI section.
1. Count the jobs with `grep -nE '^  [a-z-]+:$' .github/workflows/ci.yml`, ignoring the `on:` keys (`push:`/`pull_request:`). Replace "Three parallel GitHub Actions jobs" with the matching number word ("Five" today).
2. Commit: `harness: CODEMAP — CI job count`.

## Verification
```bash
grep -n '^- \*\*config\*\*' harness/CODEMAP.md
grep -c 'Getenv' backend/internal/config/config.go   # every variable read there is named in the bullet
grep -n 'parallel GitHub Actions jobs' harness/CODEMAP.md
python3 tools/harness/cli.py validate
```
- Every `os.Getenv("X")` in `config.go` appears in the config bullet. Every default and validation the bullet claims can be pointed to in `config.go`.
- The CI section's number matches the jobs in `ci.yml`.
- `git diff --name-only origin/main` for this task lists only `harness/CODEMAP.md`.
