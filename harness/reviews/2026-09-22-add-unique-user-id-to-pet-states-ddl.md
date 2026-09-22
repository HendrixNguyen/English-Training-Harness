---
plan: harness/plans/2026-09-22-add-unique-user-id-to-pet-states-ddl.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md, harness/ideas/_inbox/spec-never-states-when-the-pet-states-row-is-created.md]
---
# Review — Add UNIQUE user_id to pet_states DDL

**Plan:** `harness/plans/2026-09-22-add-unique-user-id-to-pet-states-ddl.md`
**Branch/worktree:** `harness/2026-09-22-high-add-unique-user-id-to-pet-states-ddl` / `.worktrees/add-unique-user-id-to-pet-states-ddl`
**Diff:** `git diff main...harness/2026-09-22-high-add-unique-user-id-to-pet-states-ddl --stat`

## Plan vs idea
Idea Expected output, checked line by line against the worktree (`git diff main...HEAD`, raw bytes via `repr`):
- `CREATE TABLE pet\_states` `user_id` column reads `user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE` — **delivered** (line 196, inside the block that starts at line 192; `UNIQUE NOT NULL` ordering matches the document's own precedent on lines 156/160). Backslash-escaping style kept: the raw diff line is `+    user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,`.
- "No other content in the document changes" — **delivered**: `--numstat` shows `1 1` for the spec; the three 1:N `user\_id` lines (180, 216, 232) are byte-identical; `git diff --check` is clean; ERD line 78 `(1:1)` unchanged.
- CODEMAP `store` entry notes the 1:1 — **delivered**: line 7 now reads `` `pet_states.user_id` is UNIQUE NOT NULL (1:1 with `users`). `` The wording is accurate to the DDL and tool-neutral.

The plan delivered the idea in full. The two bugs filed below are pre-existing spec defects in the same table that this review surfaced; they are explicitly outside the idea's scope ("No other content in the document changes") and are not defects of this plan.

## Code vs plan
Branch `harness/2026-09-22-high-add-unique-user-id-to-pet-states-ddl`, 2 commits on top of `e7bee53` (merge-base with `main`), clean working tree.

- **Task 1** (spec line 196 only) — followed. Commit `27ef616`; scoped `sed` range edit; only the `pet\_states` line changed.
- **Task 2** (CODEMAP `store` bullet) — followed. Commit `509ef3f`; one sentence inserted exactly where the plan's `sed` puts it.
- **Deviations** (all justified): `grep -n`/`grep -c` in place of `rg` (`rg` not installed here; patterns equivalent); push + Draft PR skipped (`git remote get-url origin` → no remote); `Co-Authored-By` trailer on both commits after a blank line, as the adapter requires. No `pr` in the plan frontmatter, so PR comment / `gh pr ready` are skipped in this review as well.

Verification re-run by the reviewer in the worktree (`validate` from ROOT), unedited:
```
$ grep -n "UNIQUE NOT NULL REFERENCES users" 1st-thinking-architecture-doc.md
196:    user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,

$ grep -c '^    user\\_id UUID REFERENCES users(id) ON DELETE CASCADE,$' 1st-thinking-architecture-doc.md
3

$ grep -n '(1:1)' 1st-thinking-architecture-doc.md
78:     email: VARCHAR(255)                    FK  user\_id: UUID (1:1)

$ git diff --stat main -- 1st-thinking-architecture-doc.md harness/CODEMAP.md
 1st-thinking-architecture-doc.md | 2 +-
 harness/CODEMAP.md               | 2 +-
 2 files changed, 2 insertions(+), 2 deletions(-)

$ grep -n 'pet_states.user_id' harness/CODEMAP.md
7:- **store** — Postgres + Redis clients, migrations (DDL from spec §3.2). `pet_states.user_id` is UNIQUE NOT NULL (1:1 with `users`). Everything else depends on it.

$ python3 tools/harness/cli.py validate; echo exit=$?
exit=0

$ grep -n 'user\\_id UUID' 1st-thinking-architecture-doc.md
180:    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,
196:    user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
216:    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,
232:    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,

$ git diff main...HEAD --numstat
1	1	1st-thinking-architecture-doc.md
1	1	harness/CODEMAP.md
$ git diff main...HEAD --check; echo check exit=$?
check exit=0

$ python3 -m unittest discover -s tools/harness/tests
Ran 27 tests in 0.076s
OK
```

## Quality
- No app code or tests exist yet; the change is spec + CODEMAP only, so build/test coverage is the harness unit suite (27 pass) plus `validate` (exit 0). No test gaps to file for a 2-line markdown diff.
- Boundaries: n/a (no packages yet). The constraint itself is the right one for the `store` slice to generate from; the CODEMAP note is accurate, so no correction was needed (step 7 skipped).
- Code-review pass (low effort) raised seven items. Two are real pre-existing spec defects and are filed below. The other five were judged not bugs: `user_id` as PK instead of surrogate `id` (design preference; would also change the ERD — out of scope), `updated_at` lacking an ON UPDATE trigger (generic Postgres behaviour, applies to every table in §3.2), CODEMAP duplicating the DDL (the idea explicitly asks for this note), ERD line 78 not spelling `UNIQUE` (`(1:1)` already states it and the plan expects the ERD unchanged), and a missing CHANGELOG (this repo has none; `AGENTS.md` is canonical and does not require one).

## Bugs filed
- `harness/ideas/_inbox/reconcile-pet-states-stage-between-erd-and-ddl-wilted-defaul.md` — low. ERD `stage: ENUM('seed'..'fruitful')` (line 84) omits `'wilted'` present in the DDL enum (line 148); DDL default is `'sprout'` (line 202), so `'seed'` is unreachable. Same ERD-vs-DDL drift class as this idea, same table.
- `harness/ideas/_inbox/spec-never-states-when-the-pet-states-row-is-created.md` — low. With one-row-per-user now enforced, the spec still never says who inserts the `pet\_states` row or that the insert is idempotent (§5.1 step 8 "Init Dashboard", §7 `GET /pet/status`).

## Verdict
**pass-with-bugs.** Plan and idea fully delivered; verification reproduced exactly; two low-priority pre-existing spec bugs filed. PR steps skipped: the plan has no `pr` (no remote configured), so the branch exists locally only. Human merge: `/harness merge harness/plans/2026-09-22-add-unique-user-id-to-pet-states-ddl.md`.
