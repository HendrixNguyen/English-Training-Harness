---
description: Capture a human idea and evaluate it immediately
argument-hint: "<idea text>" [--type feature|bug]
---

1. Find or create today's manual run: `RUN=$(ls -d harness/ideas/$(date +%F)-run-* 2>/dev/null | tail -1)`; if empty, `RUN=$(python3 tools/harness/cli.py new-run)`.
2. Derive a short title (≤8 words) from `$ARGUMENTS`, then `python3 tools/harness/cli.py new-idea --run $RUN --title "<title>" --type <feature unless --type bug given> --source human`.
3. Fill the idea body from the user's text: their reasoning goes in `## Why`, anything they described as the result in `## Expected output`; write "human proposal, see conversation" in `## Evidence` if none given.
4. Spawn the `harness-evaluator` agent on that idea file and print its report.
