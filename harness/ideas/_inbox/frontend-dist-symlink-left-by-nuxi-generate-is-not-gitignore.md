---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# frontend/dist symlink left by nuxi generate is not gitignored

## Why
`npx nuxi generate` (the Pages build command, and a step in the fix plan's Verification) leaves `frontend/dist` as a **symlink** to `.output/public`. The root `.gitignore` has `dist/`, which matches directories only, so git sees the symlink as an untracked file: `git status --short` shows `?? frontend/dist` (or `?? dist` from `frontend/`). A `git add -A`/`git add frontend` after a generate commits a dangling symlink. Pre-existing on `main`, not introduced by the reviewed branch, but the plan's Verification only checks `.output` and misses it.

## Expected output
- `.gitignore` ignores `dist` whether it is a directory or a symlink (`dist` without the trailing slash, scoped to `frontend/` if preferred).
- After `cd frontend && npx nuxi generate`, `git status --short` is empty.

## Evidence
- Review of plan `harness/plans/2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect.md`: in a fresh worktree, `npx nuxi generate` → `git status --short` → `?? dist`; `git check-ignore -v dist` → not ignored; `.gitignore:2: dist/`.
