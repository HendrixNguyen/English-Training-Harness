---
name: daily-ship
schedule: "0 22 * * *"
schedule_utc: "0 15 * * *"
role: .agents/roles/executor.md
skills: []
writes: the remote branch `production` (pure fast-forward of `origin/main` only) — no repository files
pr_title: none — this routine commits nothing and opens no PR; it reports
budget: 30 minutes
---

# Daily ship — 22:00 local

Unattended ship run for the English-Training-Harness repo (GitHub, `gh`). It runs after the 20:00 review run and after the owner's daily merge, and does one thing: if `origin/main` has moved past `production` and its CI is green, fast-forward `production` to it and report the `Deploy` run that push starts (`.github/workflows/deploy.yml` builds the PWA and uploads it to Cloudflare Pages, then smoke-checks both public URLs). The API is not part of this run — Railway keeps auto-deploying it from `main` on every push through its own GitHub connection, so it is already live by the time this routine runs. It never merges anything, never pushes `main`, never force-pushes, and changes no files. Make no other choices; report at the end.

1. **Sync** per `.agents/routines/README.md`: abort if the working tree is dirty (a regenerated `harness/STATE.md` may be restored) or if on `main`; then `git fetch origin main production`. Do **not** merge harness PRs here — the owner's daily merge is the gate, and this run ships only what is already on `origin/main`.

2. **Nothing new?** `git rev-parse origin/main origin/production`. If they are equal, stop: report "production is already at `<sha>`; nothing to ship" and end the run. (If `origin/production` does not exist, stop and report — the bootstrap task of the ship plan creates it; never create it here.)

3. **Require green CI on `main`.** `gh run list --workflow CI --branch main --limit 1 --json headSha,status,conclusion,url`. The `headSha` must equal `origin/main` and the run must be `completed` / `success`. Anything else (in progress, failure, cancelled, an older sha) → stop and report the run URL; never ship a red or unproven `main`.

4. **Fast-forward only.** `git merge-base --is-ancestor origin/production origin/main` must succeed; if it fails, `production` has a commit `main` does not — stop and report `git log --oneline origin/main..origin/production` for the owner. Otherwise push the fast-forward: `git push origin origin/main:production` (no `--force`, no merge commit, no local branch needed). This push is the one allowed push to `production` (AGENTS.md).

5. **Watch the Deploy run.** Within a minute `gh run list --workflow deploy.yml --branch production --limit 1 --json databaseId,headSha,status,url` shows a run for the new sha; `gh run watch <id> --exit-status` (the job's `timeout-minutes` is 25). Then `gh run view <id> --log | grep -E '^ *(ok|FAIL|WARN) |healthz:|::error::'` for the smoke lines.

6. **Report**: the old and new `production` sha, the CI run URL that qualified `main`, the Deploy run URL and its conclusion, every `ok`/`FAIL`/`WARN` smoke line (a `WARN spa fallback` line is expected until the Pages deep-link bug is fixed), and anything waiting on the owner (a red Deploy run means the PWA is stale or half-shipped — say which step failed; the API is unaffected since Railway ships it separately). A run that pushed but could not find or watch the Deploy run is a failed run; say so.

**Manual ship** (the owner, any time): the same `git fetch origin main production && git push origin origin/main:production`, or `gh workflow run deploy.yml --ref production` to re-ship the current `production` commit (tick `dry_run` to only build).
