---
name: daily-decide
schedule: "0 6 * * *"
schedule_utc: "0 23 * * *"
role: .agents/roles/evaluator.md
skills: [harness-evaluate, frontend-design]
writes: harness/ideas/ (status), harness/plans/, harness/designs/
pr_title: "harness: evaluate <date>"
budget: 4h
---

# Daily decide — 06:00 local

Unattended evaluation run for the English-Training-Harness repo (GitHub, `gh`). This run chooses the day's work; it never executes. Make routine choices yourself and report at the end. Budget: 4 hours.

1. **Sync and merge green harness PRs** per `.agents/routines/README.md` — that pulls in last night's `harness: ideate <date>` and `harness: daily review <date>` PRs.

2. **Queue.** `python3 tools/harness/cli.py blockers` first — any blocker is evaluated before anything else. Then `python3 tools/harness/cli.py next --stage evaluate --all`: reviewer bugs from `harness/ideas/_inbox/` and ideator features together, ranked by the evaluator on user impact (AGENTS.md standing priority: data loss, security and happy-path breakage outrank tidiness).

3. **Decide.** Spawn the evaluator role (load skill harness-evaluate) on the queue. Select **at most 5 bug items and at most 5 feature items** for today (owner, 2026-09-25: the 10:00 and 14:00 runs are separate capacity, so the two queues are ranked independently and neither displaces the other; feature slots go to the oldest `selected`-but-unplanned features first, and an empty feature slot is never given to a bug), each one an executor can finish inside a single execute run; reject or defer the rest with a reason. For each selected item write the plan:
   - Anything touching `frontend/` gets its design doc first: the evaluator spawns the designer role (load skill harness-design), which works inside `harness/UI-KIT.md` and the frontend spec's §7 wireframes.
   - Backend work follows YAGNI, the package layout in `harness/CODEMAP.md`, the backend spec's §6 DTO contracts, and reuses existing helpers rather than duplicating them.
   - Where the specs disagree, the layer's own spec wins; say so in the plan.

4. **Approval** is automatic for every `type: bug` plan, every `type: mvp-slice` plan and every `priority: high` feature plan (owner, 2026-09-24) — the evaluator sets `status=approved` as it writes them. Medium/low feature plans stay `draft`; only the owner's `/approve` moves them. Do not wait for a human before continuing.

5. **Bookkeeping PR — this is the hand-off, not an afterthought.** Work on branch `harness/plan-<date>` (create it from `origin/main` if the session did not start on one). Then per the README: `validate`, `state`, commit `harness/`, `git push -u origin harness/plan-<date>`, `gh pr create --base main --title "harness: evaluate <date>"`, and prove it with `gh pr view --json url -q .url`. The 10:00 bugfix run and the 14:00 feature run read the plans from that PR once it is on `main`; a plan that exists only in this worktree does not exist for them. Do not end the run, and do not report "nothing is pushed", before the PR URL prints.

6. **Report.** Per selected item: type, priority, plan path, approved or draft, which execute run will take it. Rejected/deferred items with reasons. Drafts waiting on `/approve`.
