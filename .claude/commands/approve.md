---
description: Approve a draft plan for execution (human gate)
argument-hint: <plan-file>
---

Run `python3 tools/harness/cli.py set $ARGUMENTS status=approved`. If it fails, show the error. On success, `git add harness && git commit -m "harness: approve $(basename $ARGUMENTS .md)"` and show the Approved section of `harness/STATE.md`.
