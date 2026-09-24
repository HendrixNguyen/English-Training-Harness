---
name: daily-ideate
schedule: "0 2 * * *"
schedule_utc: "0 19 * * *"
role: .agents/roles/ideator.md
skills: [harness-ideate]
writes: harness/ideas/<date>-run-NN/
pr_title: "harness: ideate <date>"
budget: 2.5h
---

# Daily ideate — 02:00 local

Unattended ideation run for the English-Training-Harness repo (GitHub, `gh`). Make routine choices yourself and report at the end. Budget: 2.5 hours; when it is spent, commit what exists.

1. **Sync, recover stranded work, merge green harness PRs** as `.agents/routines/README.md` describes.

2. **Load the product.** Read `README.md`, the three specs in `project-base/` (§1 goals of the 1st-thinking doc, the frontend spec's wireframes §7, the backend spec's §6 contracts), `harness/CODEMAP.md`, and the last three days of `harness/plans/` and `harness/reviews/` — what shipped, what failed, what the reviewer keeps filing. The product is an adaptive English-learning PWA whose retention loop is the daily 30-minute quest and the pet/plant; ideas are judged on whether a real learner notices them.

3. **Ideate.** Spawn the ideator role (load skill harness-ideate), features mode, count 5. Each idea names the user-visible outcome, the layer(s) touched, the spec sections it relies on, and an estimate that fits inside one execute run (a working day or less). Prefer improving an existing path over adding a new surface; the evaluator decides, the ideator only proposes. Inbox bugs are not the ideator's — do not rewrite or triage them.

4. **Bookkeeping PR** titled `harness: ideate <date>` per the README. The 06:00 decide run merges it.

5. **Report.** Run folder path, the five titles with layer and estimate, anything you saw in reviews that looks like a systemic problem the owner should know about.
