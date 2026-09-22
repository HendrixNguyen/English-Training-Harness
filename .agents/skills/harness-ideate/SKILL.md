---
name: harness-ideate
description: Run one ideation cycle for the harness — create a run folder, sweep inbox bugs, research, write idea files and a run summary. Use when asked to ideate, find new features, or produce MVP slices.
---

# harness-ideate

Adopt the role in `.agents/roles/ideator.md`. Inputs: `mode` (`features` default, or `mvp`), `count` (default 5; ignored in mvp mode).

## Procedure

1. **Validate first.** Run `python3 tools/harness/cli.py validate`. If it exits 1, stop and report the invalid files — do not build on a broken state.
2. **Create the run.** `RUN=$(python3 tools/harness/cli.py new-run [--mvp])`.
3. **Sweep the inbox.** For every `harness/ideas/_inbox/*.md`: `git mv` it into `$RUN/`, then `python3 tools/harness/cli.py set $RUN/<file> run=<run-name>`. Record each in `_run.md` under *Inbox swept*.
4. **Read, in this order, and no more than needed:** `harness/CODEMAP.md`; the spec sections relevant to the mode (`project-base/1st-thinking-architecture-doc.md` §1, §5, §7 for features; §2–§4, §6, §7 for mvp); the last two `_run.md` files; if a `remembering-conversations` skill is available, query it for prior decisions about this project.
5. **Research (features mode only).** Look for 2–3 external references on retention mechanics in language-learning apps or on the specific gap you are targeting. Record URLs under `## Evidence`.
6. **Write ideas.** For each: `python3 tools/harness/cli.py new-idea --run $RUN --title "<Title>" --type <feature|bug|mvp-slice> --source ideator [--order N]`, then fill the three body sections of the created file (body only — leave frontmatter alone).
7. **Write `_run.md`.** Fill *Read*, *Inbox swept*, *Proposed* (one line per idea path + title), *Notes* (what you considered and dropped).
8. **Finish.** `python3 tools/harness/cli.py validate && python3 tools/harness/cli.py state`. Commit: `git add harness && git commit -m "harness: ideation run <run-name>"`.
9. **Report** the run path, the idea list with one-line summaries, and any inbox bugs swept. Stop — do not evaluate.
