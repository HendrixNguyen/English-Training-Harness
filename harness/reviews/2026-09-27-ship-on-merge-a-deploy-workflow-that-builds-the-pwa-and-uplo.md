---
plan: harness/plans/2026-09-25-ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo.md
verdict: pass
bugs: []
---
# Review — Ship on merge: a deploy workflow that builds the PWA and uploads it to Cloudflare Pages on every push to main, then smoke-checks both public URLs

**Plan:** `harness/plans/2026-09-25-ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo.md`
**Branch/worktree:** `harness/2026-09-25-high-ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo` / `.worktrees/ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo`
**Diff:** `git diff main...harness/2026-09-25-high-ship-on-merge-a-deploy-workflow-that-builds-the-pwa-and-uplo --stat`

## Plan vs idea
Delivered, and already on `main`. The plan (revised by the owner to ship from a `production` branch) gave the idea's Expected output: a `Deploy` workflow that builds the PWA, uploads it to Cloudflare Pages and smoke-checks both public URLs, plus the nightly ship routine. This branch was **squash-merged as PR #36** (`61224f6`, 2026-09-25T15:41Z, `gh pr view 36` → headRefName = this branch, MERGED). `main` has since moved past it: PR #37 (`eec58cb`, branch `harness/ship-api-from-production`) replaced the push trigger with `workflow_run` on CI-green `production`, release PRs + CHANGELOG versioning, tagging and rollback. The plan's `merged: false` flag is stale.

## Code vs plan
Reviewed at `origin/<branch>` head `283ec20` (4 commits ahead of its base, 62 behind `origin/main`).

- Task 1 `deploy.yml` — followed, with the owner's mid-execution deviation (no Railway step) recorded in the execution summary. Justified.
- Task 2 `SMOKE_WEB_SPA_WARN` knob — followed.
- Task 3 `daily-ship.md` + routines README row — followed (fast-forward-only design, later replaced on main by #37's release-PR design).
- Task 4 docs (AGENTS.md, deploy/README.md, CODEMAP) — followed.
- Task 5 `production` bootstrap — done (executor recorded `68529ad`).

Re-run evidence:
```
$ gh run list --branch <branch> --limit 1
completed  success  docs: ship-from-production runbook, ...  CI  push  36155907491  1m24s  2026-09-25T15:41:48Z
$ actionlint .github/workflows/deploy.yml && echo actionlint ok
actionlint ok
$ ruby -ryaml -e '... p on.keys.sort; p on["push"]; p y["concurrency"]; p y["jobs"].keys'
["push", "workflow_dispatch"]
{"branches"=>["production"]}
{"group"=>"deploy", "cancel-in-progress"=>false}
["ship"]
$ sh -n deploy/smoke-web.sh && echo parses
parses
$ git merge-tree --write-tree origin/main origin/<branch>; echo exit=$?
exit=1  — CONFLICT in .agents/routines/README.md, .agents/routines/daily-ship.md, .github/workflows/deploy.yml, AGENTS.md, deploy/README.md, harness/CODEMAP.md
```
The live smoke-web runs and `nuxi generate` were not re-run: the content is already on `main` and superseded; re-proving a superseded file adds nothing.

## Quality
The branch as it stands **contradicts `origin/main`**: its `deploy.yml` triggers on `push: production` with `contents: read` and no tagging, whereas main's triggers on `workflow_run` of CI on `production`, reads the version from CHANGELOG.md and tags `v<x.y.z>`; its `daily-ship.md` fast-forwards `production` by direct push, which main's AGENTS.md now forbids (production moves only by PR). Re-merging this branch would either conflict (6 files, above) or, resolved carelessly, revert #37. That is not a defect in the delivered work (it was correct against its base and is merged); it is a merge-hygiene fact for the orchestrator. No bug filed: filing a blocker would stop the correct next state (`merged=true`).

## Bugs filed
None.

## Verdict
`pass` — idea delivered and live on `main` via PR #36. **Do not merge this branch into `harness/daily-2026-09-27`**; the orchestrator should set the plan `merged=true` (already merged by squash).
