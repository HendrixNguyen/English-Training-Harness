---
name: daily-ship
schedule: "0 22 * * *"
schedule_utc: "0 15 * * *"
role: .agents/roles/executor.md
skills: []
writes: one `release/v<x.y.z>` branch (origin/production + origin/main merged in + a CHANGELOG.md section), its PR into `production` (merged by this run when green), one back-merge PR `production → main` (the owner merges it)
pr_title: "Release v<x.y.z>"
budget: 60 minutes
---

# Daily ship — 22:00 local

Unattended release run for the English-Training-Harness repo (GitHub, `gh`). It runs after the 20:00 review run has merged the day's code PR (or the owner has), and does one thing: if `origin/main` has commits `production` lacks and `main`'s CI is green, cut a versioned release, merge it into `production` through a pull request once that PR's CI is green, and report the `Deploy` run that follows. `production` is the only branch that ships: CI runs on every push to it, `.github/workflows/deploy.yml` starts when that run is green (PWA to Cloudflare Pages, smoke checks, then the `v<x.y.z>` tag and GitHub Release), and Railway rebuilds the API from the same push with Wait for CI on — this routine never calls Railway. It never pushes `main` or `production` directly, never force-pushes, never merges into `main`. Make no other choices; report at the end.

1. **Sync** per `.agents/routines/README.md`: abort if the working tree is dirty (a regenerated `harness/STATE.md` may be restored) or if on `main`; then `git fetch origin main production --tags`. Do **not** merge harness PRs here — the review run's green-gated daily merge is the gate, and this run ships only what is already on `origin/main`. If `origin/production` does not exist, stop and report; never create it here.

2. **Nothing new?** `git log --oneline origin/production..origin/main`. Empty → stop: report "production is at `<sha>` (`v<x.y.z>`); nothing to ship". A back-merge makes `main` contain `production`'s commits, not the other way round, so this check is what matters.

3. **Require green CI on `main`.** `gh run list --workflow CI --branch main --limit 1 --json headSha,status,conclusion,url`. The `headSha` must equal `origin/main` and the run must be `completed` / `success`. Anything else (in progress, failure, cancelled, an older sha) → stop and report the run URL; never release a red or unproven `main`.

4. **Pick the version.** The last release is the first `## [x.y.z]` heading in `git show origin/production:CHANGELOG.md`. What ships: every `harness/plans/*.md` whose frontmatter is `status: done` on `origin/main` but not on `origin/production` (`git diff --name-only origin/production origin/main -- harness/plans/`, then read `type`, `title` and `status` from each side), plus the subjects of `git log --first-parent --merges --format=%s origin/production..origin/main` for anything not tied to a plan. Any `feature` or `mvp-slice` plan → **minor** bump (`x.(y+1).0`); otherwise **patch** (`x.y.(z+1)`). Never bump major. If tag `v<new>` already exists, stop and report — the version history is broken and the owner decides.

5. **Cut the release branch.** `git switch -c release/v<new> origin/production`, then `git merge --no-ff origin/main -m "release v<new>: merge main"` — merging `main` into this working branch is the only merge allowed. A conflict means a hotfix on `production` never made it back to `main`: `git merge --abort`, delete the local branch, stop and report the conflicting files and the open back-merge PR for the owner. Then add the section to `CHANGELOG.md` directly above the previous `## [` heading:

   ```
   ## [<new>] - <YYYY-MM-DD>

   ### Added      ← feature / mvp-slice plans: "- <plan title> (<plan file>)"
   ### Fixed      ← bug plans
   ### Changed    ← merge subjects not tied to a plan (omit empty headings)
   ```

   Commit only `CHANGELOG.md` as `release: v<new>`, push `release/v<new>`.

6. **Release PR, merged on green.** `gh pr create --base production --head release/v<new> --title "Release v<new>" --body-file <the section>`, and prove it with `gh pr view --json url -q .url`. `gh pr checks <n> --watch` (CI's `pull_request` run). All green and mergeable → `gh pr merge <n> --merge` (a merge commit, never squash or rebase, so `main`'s commits stay ancestors of `production`). Red, refused or still pending when the budget runs out → leave the PR open and report it; do not ship.

7. **Watch the ship.** The merge is a push to `production`: first its CI run (`gh run list --workflow CI --branch production --limit 1 --json databaseId,headSha,status,conclusion,url`, `headSha` = the merge commit; `gh run watch <id> --exit-status`), then the `Deploy` run it triggers (`gh run list --workflow deploy.yml --limit 1 --json databaseId,headSha,status,url`, same `headSha`; `gh run watch <id> --exit-status`, job `timeout-minutes` 25). Then `gh run view <id> --log | grep -E '(ok|FAIL|WARN) |healthz:|::error::|version: v'` for the smoke lines, and `gh release view v<new> --json url -q .url` for the release. A red production CI run means Deploy never started — report it.

8. **Back-merge PR.** If no open PR from `production` into `main` exists, `gh pr create --base main --head production --title "Back-merge v<new> into main" --body "Carries the v<new> CHANGELOG section (and any hotfix) back to main. Owner merges."`; otherwise leave the open one — it now also carries `v<new>`. Never merge it.

9. **Report**: old and new `production` sha and version, the `main` CI run that qualified it, the release PR URL, the production CI and Deploy run URLs and conclusions, every `ok`/`FAIL`/`WARN` smoke line (a `WARN spa fallback` line is expected until the Pages deep-link bug is fixed), the GitHub Release URL, the back-merge PR URL, and anything waiting on the owner. A red Deploy run means the PWA is stale or half-shipped and the tag was not created — say which step failed; the API rolls out from the same push through Railway, so a failed `Smoke-check the API` step can mean the new API failed to deploy — point the owner at the Railway dashboard. A run that merged the release PR but could not find or watch the Deploy run is a failed run; say so.

**Hotfix, manual ship and rollback** (the owner, any time) are in `deploy/README.md` "Ship from `production`" and "Versions and rollback".
