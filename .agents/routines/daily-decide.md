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

3. **Decide.** Spawn the evaluator role (load skill harness-evaluate) on the queue. Select **at most 5** items for today, each one an executor can finish inside a single execute run; reject or defer the rest with a reason. For each selected item write the plan:
   - UI work gets a design doc from the frontend spec (`project-base/… Frontend Technical Specification.md` §6 design system, §7 wireframes) using the frontend-design skill.
   - Backend work follows YAGNI, the package layout in `harness/CODEMAP.md`, the backend spec's §6 DTO contracts, and reuses existing helpers rather than duplicating them.
   - Where the specs disagree, the layer's own spec wins; say so in the plan.

4. **Approval** is automatic for every `type: bug` plan, every `type: mvp-slice` plan and every `priority: high` feature plan (owner, 2026-09-24) — the evaluator sets `status=approved` as it writes them. Medium/low feature plans stay `draft`; only the owner's `/approve` moves them. Do not wait for a human before continuing.

5. **Bookkeeping PR** titled `harness: evaluate <date>` per the README. The 10:00 bugfix run and the 14:00 feature run merge it and read the plans from `main`.

6. **Report.** Per selected item: type, priority, plan path, approved or draft, which execute run will take it. Rejected/deferred items with reasons. Drafts waiting on `/approve`.
