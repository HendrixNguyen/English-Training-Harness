---
plan: harness/plans/2026-09-26-infisical-is-the-source-of-truth-for-deploy-secrets-link-fil.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/infisical-push-back-step-leaves-production-secrets-in-a-worl.md]
---
# Review — Infisical is the source of truth for deploy secrets: `.infisical.json` in the repo, an "Env source of truth" runbook section, Railway sync by Infisical

**Plan:** `harness/plans/2026-09-26-infisical-is-the-source-of-truth-for-deploy-secrets-link-fil.md`
**Branch/worktree:** `harness/2026-09-26-medium-infisical-is-the-source-of-truth-for-deploy-secrets-link-fil` / `.worktrees/infisical-is-the-source-of-truth-for-deploy-secrets-link-fil`
**Diff:** `git diff main...harness/2026-09-26-medium-infisical-is-the-source-of-truth-for-deploy-secrets-link-fil --stat`

## Plan vs idea
Delivered. The idea's expected output is all present:
- `.infisical.json` committed, holding only the id and the env. The workspace id matches the idea.
- A `deploy/README.md` "Env source of truth — Infisical" section covering login, export, push-back with the blank-value filter, inviting a collaborator, and Railway via Infisical secret sync.
- The owner checklist now opens with `infisical login && infisical export …`.
- The optional `infisical run -- … smoke-api.sh` example is included.

The live `infisical export` check is the owner's, since it needs a login. No secret values were read or written, by the executor or by me.

## Code vs plan
Branch head: CI `36303403232` completed/success. `git merge-tree origin/main origin/<branch>` is clean.

- Task 1 (`.infisical.json`, `.gitignore` comment): followed.
- Task 2 (runbook section in plan order, checklist, smoke example, CODEMAP): followed, with two justified deviations. There was no "fill the file" wording to replace, so the box was prepended. A third commit wires `infisical export` into Target B and the local-run note to reach the plan's own `grep -c ≥ 8`.

```
$ python3 -c '...assert set(d)=={"workspaceId","defaultEnvironment"} and d["defaultEnvironment"]=="prod"; print("ok")'
ok
$ git check-ignore -v deploy/.env && ! git check-ignore .infisical.json
.gitignore:8:.env*	deploy/.env      (and .infisical.json not ignored)
$ grep -n 'Env source of truth' deploy/README.md harness/CODEMAP.md
deploy/README.md:31, :101, :132, :145, :148, :149 ; harness/CODEMAP.md:29
$ grep -c 'infisical' deploy/README.md
8
$ git ls-files | grep -c '^deploy/.env$'
0
$ python3 tools/harness/cli.py validate; echo validate=$?
ok / validate=0
```

## Quality
- Security (low bug): the push-back one-liner writes secrets to `/tmp/nonblank.env`. That file is world-readable under umask 022 and is left behind if `infisical secrets set` fails, because the `&&` chain skips `rm`.
- Doc consistency (same bug): the env table's Railway column still says "set on the service", which contradicts step 5's "never edited by hand".
- `.infisical.json` holds a project id only. The owner's idea records that it is not a secret, and I agree: the CLI still needs a login to read anything.
- **Merge note:** with plan 3 (caddyfile), `deploy/README.md` conflicts in the Target B paragraph, where both append a sentence, and `harness/CODEMAP.md` conflicts in the Deploy bullet. Keep both sides. With plan 4 it merges cleanly.
- CODEMAP Deploy bullet: accurate.

## Bugs filed
- `harness/ideas/_inbox/infisical-push-back-step-leaves-production-secrets-in-a-worl.md` (low). A world-readable `/tmp` secrets file survives a failed upload, and the env table's Railway column contradicts step 5.

## Verdict
`pass-with-bugs`. It may go into today's daily PR; resolve the README and CODEMAP conflict with plan 3 by keeping both sides.
