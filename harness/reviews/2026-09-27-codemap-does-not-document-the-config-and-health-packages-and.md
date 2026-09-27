---
plan: harness/plans/2026-09-27-codemap-does-not-document-the-config-and-health-packages-and.md
verdict: pass
bugs: []
---
# Review — CODEMAP does not document the config and health packages and still says three CI jobs

**Plan:** `harness/plans/2026-09-27-codemap-does-not-document-the-config-and-health-packages-and.md`
**Branch/worktree:** `harness/2026-09-27-low-codemap-does-not-document-the-config-and-health-packages-and` / `.worktrees/codemap-does-not-document-the-config-and-health-packages-and`
**Diff:** `git diff main...harness/2026-09-27-low-codemap-does-not-document-the-config-and-health-packages-and --stat`

## Plan vs idea
Delivered. The idea's two halves are both done. The `health` bullet was already on main, as the plan noted. This branch adds the `config` bullet and corrects the CI job count. The CI timeout clause (docker-images is 15 min) was also wrong; the executor fixed it, which falls within the plan's goal of an accurate CI section.

## Code vs plan
Branch head `995ad0e`, CI `36296611935` completed/success. `git merge-tree origin/main origin/<branch>` is clean. The diff touches only `harness/CODEMAP.md`.

- Task 1 (`config` bullet before `health`): followed. I checked each claim against `backend/internal/config/config.go` at the branch head:
  - required: `DATABASE_URL`/`REDIS_URL` (L62-67), `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`JWT_SECRET` (L76-84)
  - `MinJWTSecretBytes` = 32 (L86)
  - `ENCRYPTION_SECRET_KEY` via `secrets.ParseHexKey`: `hex.DecodeString(strings.TrimSpace(s))`, must be 32 bytes (`secrets.go:36-41`)
  - `PORT` → 8080
  - `VAPID_SUBJECT` → `DefaultVAPIDSubject`
  - `GIN_MODE` → release, then the switch
  - `FRONTEND_ORIGIN` passed through
  - VAPID keys read but not validated
  - `cmd/api/main.go:149` gates the worker on both keys
  - `main.go:89` `airouter.ConfigFromEnv(os.Getenv)`
- Task 2 ("Five parallel", `docker-images: 15`): followed. `grep -nE '^  [a-z-]+:$|timeout-minutes' .github/workflows/ci.yml` gives five jobs (backend-unit, backend-integration, harness-tooling, frontend, docker-images), timeouts 10/10/10/10/15, and the same on `origin/main`.

```
$ grep -n '^- \*\*config\*\*' harness/CODEMAP.md    → line 11 (branch)
$ grep -n 'parallel GitHub Actions jobs' harness/CODEMAP.md   → "Five parallel … (docker-images: 15)"
$ git grep -n 'Three parallel' origin/main -- harness/CODEMAP.md   → origin/main:harness/CODEMAP.md:37 (the defect being fixed)
$ gh run list --branch <branch> --limit 1   → completed success … 36296611935
```

## Quality
- Accuracy nit, no bug filed: the bullet says errors name the variable "and, for the secrets, how to generate one". A *missing* `JWT_SECRET` returns plain `config: JWT_SECRET is required` with no hint; only the too-short and `ENCRYPTION_SECRET_KEY` errors carry one. Also, when more than one of the three map-checked variables is missing, which one is named first depends on Go's random map order. Neither detail misleads a reader about behaviour.
- **Merge note:** every CODEMAP-editing branch in today's set touches this file (plans 2, 3, 4, 5, 6, 7). This one's hunks are an inserted line in *Backend packages* and the first line of *CI*. Plan 2 edits the `backend-unit`/`backend-integration` bullets just below that, so expect nearby-hunk conflicts to resolve by hand.

## Bugs filed
None.

## Verdict
`pass`. It may go into today's daily PR.
