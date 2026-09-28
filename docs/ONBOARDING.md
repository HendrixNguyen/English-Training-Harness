# Onboarding — what to read, in what order

Nothing here repeats another document. This is the order, and why.

## Every session, any agent

1. `AGENTS.md` — the rules that bind you: harness pipeline, branch and PR policy, CI gates, tooling caveats on this machine.
2. `python3 tools/harness/cli.py context` — live state, the actionable work, and the code map index. Read-only; never rewrites `STATE.md`.
3. `harness/CODEMAP.md` — the paragraph for the package you are about to touch, before you open any file in it. One paragraph per package; it is dense and it is accurate.

## Before you change code

4. The canonical spec for your layer, in `project-base/`. The backend spec's §6 is the REST contract; §8 is the plant arithmetic. The frontend spec's §3 is service-worker caching, §4 the stores, §7 the wireframes. Quote the section in your plan.
5. `CLAUDE.md` — the cross-layer flows and the data-model boundaries, for when your change crosses them.
6. The plan's `design:` doc if it has one, then `harness/UI-KIT.md`.

## Once, to know the product

7. `docs/PRODUCT.md` — who this is for and what it promises.
8. `README.md` — layout, setup, and the exact test commands.
9. `CHANGELOG.md` — what has already shipped, and `deploy/README.md` if you are deploying.

## Keeping it true

`harness/CODEMAP.md` and the documents in step 5 and 8 are maintained by the `knowledge-sync` skill, which every executor runs at the end of a plan and the reviewer checks. `python3 tools/harness/cli.py knowledge` fails when a package has no paragraph. If you find a stale claim, fix it in the same branch as the change that made it stale.