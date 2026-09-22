---
idea: harness/ideas/2026-09-22-run-01/add-unique-user-id-to-pet-states-ddl.md
status: done
priority: high
merged: false
branch: harness/2026-09-22-high-add-unique-user-id-to-pet-states-ddl
worktree: .worktrees/add-unique-user-id-to-pet-states-ddl
---
# Add UNIQUE user_id to pet_states DDL — Plan

**Idea:** `harness/ideas/2026-09-22-run-01/add-unique-user-id-to-pet-states-ddl.md`
**Goal:** Make the §3.2 `pet\_states` DDL match the §3.1 ERD's `(1:1)` by adding `UNIQUE NOT NULL` to `user_id`, and record the invariant in CODEMAP.

**Root cause:** `1st-thinking-architecture-doc.md` line 196 (inside `CREATE TABLE pet\_states (`, line 192) reads `    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,` — no `UNIQUE`. The ERD on line 78 says `FK  user\_id: UUID (1:1)`. The document backslash-escapes `_`; keep that style. The identical `user\_id` line also appears at lines 180, 216, 232 (`push\_subscriptions`, `daily\_progress`, `roadmaps`) — those are 1:N and must **not** change, so the edit is scoped to the `pet\_states` block.

Shell-only; no app code, no tests to write. Run all commands from the worktree root.

## Tasks

### Task 1: Add `UNIQUE NOT NULL` to `pet\_states.user\_id` in the spec
**Files:** `1st-thinking-architecture-doc.md` (line 196 only)

1. Confirm the target line exists once inside the `pet\_states` block:
   ```sh
   sed -n '/^CREATE TABLE pet\\_states (/,/^);/p' 1st-thinking-architecture-doc.md | grep -c '^    user\\_id UUID REFERENCES users(id) ON DELETE CASCADE,$'
   ```
   Expected: `1`
2. Apply the scoped edit (macOS `sed -i ''`; on GNU sed use `sed -i`):
   ```sh
   sed -i '' '/^CREATE TABLE pet\\_states (/,/^);/ s/^    user\\_id UUID REFERENCES users(id) ON DELETE CASCADE,$/    user\\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,/' 1st-thinking-architecture-doc.md
   ```
3. Verify exactly one line changed and it is line 196:
   ```sh
   git diff --stat -- 1st-thinking-architecture-doc.md   # expect: 1 file changed, 1 insertion(+), 1 deletion(-)
   git diff -U0 -- 1st-thinking-architecture-doc.md | grep '^[-+] '
   ```
   Expected:
   ```
   -    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,
   +    user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
   ```
4. Commit: `git commit -am "spec: pet_states.user_id UNIQUE NOT NULL (matches §3.1 1:1 ERD)"`

### Task 2: Note the 1:1 invariant in CODEMAP `store`
**Files:** `harness/CODEMAP.md` (line 7, the `**store**` bullet)

1. Edit:
   ```sh
   sed -i '' 's/^\(- \*\*store\*\* — .*DDL from spec §3\.2)\.\)/\1 `pet_states.user_id` is UNIQUE NOT NULL (1:1 with `users`)./' harness/CODEMAP.md
   ```
2. Verify:
   ```sh
   grep -n 'pet_states.user_id' harness/CODEMAP.md
   ```
   Expected: `7:- **store** — Postgres + Redis clients, migrations (DDL from spec §3.2). \`pet_states.user_id\` is UNIQUE NOT NULL (1:1 with \`users\`). Everything else depends on it.`
3. Commit: `git commit -am "codemap: store — pet_states.user_id is UNIQUE (1:1)"`

## Verification
```sh
rg -n 'UNIQUE NOT NULL REFERENCES users' 1st-thinking-architecture-doc.md
# expect exactly one hit:
# 196:    user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,

rg -c '^    user\\_id UUID REFERENCES users\(id\) ON DELETE CASCADE,$' 1st-thinking-architecture-doc.md
# expect: 3   (push_subscriptions, daily_progress, roadmaps untouched)

rg -n '\(1:1\)' 1st-thinking-architecture-doc.md
# expect: 78:     email: VARCHAR(255)                    FK  user\_id: UUID (1:1)   (ERD unchanged)

git diff --stat main -- 1st-thinking-architecture-doc.md harness/CODEMAP.md
# expect: 2 files changed, 2 insertions(+), 2 deletions(-)

rg -n 'pet_states.user_id' harness/CODEMAP.md
# expect one hit on the **store** bullet

python3 tools/harness/cli.py validate; echo exit=$?
# expect: exit=0
```

## Execution summary

**Built** (branch `harness/2026-09-22-high-add-unique-user-id-to-pet-states-ddl`, worktree `.worktrees/add-unique-user-id-to-pet-states-ddl`, 2 commits on top of `main`):
- `27ef616` spec: pet_states.user_id UNIQUE NOT NULL (matches §3.1 1:1 ERD) — `1st-thinking-architecture-doc.md` line 196 only; the three 1:N `user\_id` lines (180, 216, 232) are untouched.
- `509ef3f` codemap: store — pet_states.user_id is UNIQUE (1:1) — one sentence added to the `**store**` bullet in `harness/CODEMAP.md`.

**Deviations:**
- Verification used `grep -n` / `grep -c` instead of `rg`: `rg` is not an executable in this environment (`command rg` → "command not found"). Patterns are equivalent; results below match the plan's expected output exactly.
- Push + Draft PR skipped: `git remote get-url origin` → "No such remote 'origin'" (no remote configured; `gh` is authenticated but has nothing to push to). Branch exists locally only.
- Commit messages carry a `Co-Authored-By` trailer as required by the adapter.

**Verification output** (run in the worktree; `validate` from ROOT):
```
$ grep -n "UNIQUE NOT NULL REFERENCES users" 1st-thinking-architecture-doc.md
196:    user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,

$ grep -c "^    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,$" 1st-thinking-architecture-doc.md
3

$ grep -n "(1:1)" 1st-thinking-architecture-doc.md
78:     email: VARCHAR(255)                    FK  user\_id: UUID (1:1)

$ git diff --stat main -- 1st-thinking-architecture-doc.md harness/CODEMAP.md
 1st-thinking-architecture-doc.md | 2 +-
 harness/CODEMAP.md               | 2 +-
 2 files changed, 2 insertions(+), 2 deletions(-)

$ grep -n "pet_states.user_id" harness/CODEMAP.md
7:- **store** — Postgres + Redis clients, migrations (DDL from spec §3.2). `pet_states.user_id` is UNIQUE NOT NULL (1:1 with `users`). Everything else depends on it.

$ python3 tools/harness/cli.py validate; echo exit=$?
exit=0
```
