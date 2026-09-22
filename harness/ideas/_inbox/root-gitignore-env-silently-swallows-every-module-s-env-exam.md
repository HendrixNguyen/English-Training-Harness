---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Root .gitignore .env* silently swallows every module's .env.example

## Why
The repo-root `.gitignore` carries two overlapping patterns, `*.env` (line 5) and `.env*` (line 6).
Neither has a leading slash, so both apply to **every directory in the repo**, and `.env*` matches
`.env.example` as well as `.env`. The consequence is that a committed environment template is
silently untrackable anywhere in the tree:

```
$ git check-ignore -v frontend/.env.example
.gitignore:6:.env*	frontend/.env.example
```

The store slice already hit this. `harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md`
Task 2 Step 2 asserted "`.env.example` is unaffected by that pattern and stays tracked"; it was
wrong, `git add` refused the file, and the executor had to add a `!.env.example` negation to
`backend/.gitignore` to recover. That workaround is correct and works (a deeper `.gitignore` wins,
and `.env*` excludes a file, not a parent directory), but it is per-module and invisible: the next
person to add `frontend/.env.example` — the Nuxt slice will need one for `NUXT_PUBLIC_*` — gets no
error from `git add` in most workflows, just a file that never appears in `git status`, and
`make dev`/`make up` documentation that points at a file nobody else has.

The failure mode is quiet rather than loud, which is what makes it worth fixing once at the root
instead of re-deriving the negation in each module.

## Expected output
The root `.gitignore` ignores real secret files and never ignores templates, repo-wide and without
per-module workarounds. Concretely:

- Line 5 `*.env` and line 6 `.env*` are replaced by a pattern pair that excludes `.env` and
  `.env.<something>` while re-including any `*.env.example` / `.env.example`, e.g.

  ```
  .env
  .env.*
  *.env
  !.env.example
  !*.env.example
  ```

- `git check-ignore -v frontend/.env.example` exits 1 (not ignored) from a clean checkout, and so
  does the same probe for any future module path.
- `git check-ignore -q .env` and `.env.local` still exit 0 anywhere in the tree.
- The now-redundant `!.env.example` negation in `backend/.gitignore` is removed, and
  `git ls-files backend/.env.example` still lists the file afterwards (proving the root fix alone
  is sufficient).

## Evidence
- Plan under review: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
- Amending plan whose Task 2 Step 2 assertion was falsified:
  `harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md`
  (*Execution summary → Deviations → 1*, where the executor raised this trap).
- `.gitignore:5-6` — `*.env`, `.env*`.
- `backend/.gitignore:3-5` — the per-module `!.env.example` negation and its explanatory comment.
- Reviewer reproduction, repo root: `git check-ignore -v frontend/.env.example` →
  `.gitignore:6:.env*	frontend/.env.example` (exit 0 = ignored).
