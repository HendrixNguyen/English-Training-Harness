---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Overtaken: all four CI jobs now carry timeout-minutes: 10; the remaining ask (pinning python-version from 3.x) is tidiness with no live breakage."
---
# CI jobs have no timeout-minutes and the harness job floats python-version 3.x

## Why
Two hardening gaps in `.github/workflows/ci.yml`, both cheap to close and both the kind that only
show up once the workflow has been running unattended for a while.

**No `timeout-minutes` on any of the three jobs.** GitHub's default is 360 minutes. The realistic
hang is in `backend-integration`: a service container that comes up healthy but then stalls, or a
future integration test that blocks on a connection with no context deadline, burns six hours of
billed runner time per job before GitHub kills it — with no failure signal until then. The whole
suite finishes in well under a minute locally (`go test ./... -count=1` ≈ 3 s, the integration run
≈ 1.3 s), so a 10-minute cap is two orders of magnitude of headroom.

**`python-version: "3.x"` in `harness-tooling`** (`.github/workflows/ci.yml:99-101`) resolves to
whatever the newest CPython on the runner is, which will drift away from the 3.9.6 the tooling is
actually developed and tested against on this machine. When a future 3.x removes something
`tools/harness/` uses, the job goes red on a version nobody here can reproduce, and the failure looks
like a harness-artifact problem rather than an interpreter problem. I grepped `tools/harness/` for the
usual casualties (`distutils`, `imp`, `datetime.utcnow`, `asyncio.get_event_loop`) and found none, so
there is no live breakage today — this is about not leaving a floating dependency in the one job that
guards the harness's own state.

## Expected output
Each job declares `timeout-minutes: 10`. `harness-tooling` pins a concrete minor, e.g.
`python-version: "3.13"`, or runs a small matrix (`["3.9", "3.13"]`) if the intent is that the tooling
keep working on the owner's system interpreter as well as a current one — that would additionally turn
the local/CI version gap into something CI tests rather than something CI hides.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ci-on-github-actions-for-backend-and-harness-tooling.md`
- `.github/workflows/ci.yml` — jobs at lines 16, 42, 95; no `timeout-minutes` anywhere.
- `.github/workflows/ci.yml:99-101` — `actions/setup-python@v7` with `python-version: "3.x"`.
- Local timings, worktree `.worktrees/ci-on-github-actions-for-backend-and-harness-tooling`:
  `unit exit=0` in ~3 s, `3/3 integration tests ran and passed` in ~1.3 s,
  `Ran 30 tests in 0.178s`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — overtaken.** `.github/workflows/ci.yml` lines 21/48/111/124 each set `timeout-minutes: 10`. `python-version: "3.x"` still floats (line 116) but the reviewer found nothing that would break; not worth a branch on its own.
