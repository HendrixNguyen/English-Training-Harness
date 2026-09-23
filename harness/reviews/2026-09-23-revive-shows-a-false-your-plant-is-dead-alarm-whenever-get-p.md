---
plan: harness/plans/2026-09-23-revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/no-test-pins-the-stale-data-wins-branch-order-on-revive-so-a.md, harness/ideas/_inbox/the-revive-missed-days-line-renders-bo-hoc2-ngay-with-no-spa.md]
---
# Review — /revive: render an error state when GET /pet/status fails, never a false wilted plant

**Plan:** `harness/plans/2026-09-23-revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md`
**Branch/worktree:** `harness/2026-09-23-high-revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p` / `.worktrees/revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p`
**Diff:** `git diff main...harness/2026-09-23-high-revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p --stat`

## Plan vs idea

The idea's *Expected output* has four clauses; all four hold on the branch.

1. *"renders an error state with a retry action when `pet.error` is set and `pet.status` is null"* — `pages/revive.vue:109-113`, `StateBlock state="error" … action="Thử lại" @action="pet.load()"`. Verified in a real browser (below) and by test case 1.
2. *"The wilted branch is entered only on real data (`pet.status !== null && pet.isWilted`)"* — `:69` is `v-else-if="pet.status"`, and the preceding branch `:56` already claims `pet.status && !pet.isWilted`, so reaching `:69` implies `isWilted`. The condition is not written as `pet.status && pet.isWilted`, but the branch chain makes the two equivalent; the inline comment says so.
3. *"Offline with a cached status, the last-known state is shown"* — holds, because `load()` leaves a previous `status` in place on failure (`stores/pet.ts:45-55`) and the `pet.status` branch is ordered above the `pet.error` branch. Correct, but not pinned by any test — bug #1 below.
4. *"never that it is dead"* — the copy names the state as unknown ("Chưa thể biết cây có héo hay không"), which is the right register.

**Bonus not claimed by the plan:** on `main` the pre-mount tick (`loading === false`, `status === null`) fell through to the old bare `v-else` and flashed the wilted alarm before `onMounted` ran. The new final `v-else` is the loading skeleton, so that flash is gone too.

## Code vs plan

Diff touches exactly the three files the plan's *File structure* table names, and nothing else:

```
frontend/pages/revive.vue              | 26 ++++++++----
frontend/tests/unit/revivePage.test.ts | 77 +++++++++++++++++++++++++++++++++
harness/CODEMAP.md                     |  2 +-
3 files changed, 96 insertions(+), 9 deletions(-)
```

- **Task 1 — followed.** `tests/unit/revivePage.test.ts` is the plan's listing verbatim. Assertions are all on rendered output (`wrapper.text()`, `find('[role="alert"]')`, `find('[data-stage="wilted"]')`, `find('[role="status"]')`) — no reaching into component internals or store fields.
- **Task 2 — followed.** Template-only, `<script setup>` byte-identical to `main`; branch order is `passed` → healthy → `pet.status` (wilted) → `pet.error` → `v-else` skeleton, exactly as specified. **No store change**, as the plan promised: `git diff main...HEAD -- frontend/stores/` is empty, and `petStore.test.ts` is untouched.
- **Task 3 — followed with the two declared deviations** (port 3102 instead of 3101; Playwright MCP + local stub instead of the Chrome extension). CODEMAP's `shell` bullet carries the page-state clause and it describes the code accurately.

### Verification re-run (clean shell, `env -u NUXT_PUBLIC_API_BASE -u PORT -u HOST`, from `frontend/` in the worktree)

```
npm ci                                       → OK
npm run lint            (eslint .)           → exit 0, no output
npm run typecheck       (nuxi typecheck)     → exit 0
npm run test:unit       (vitest run)         → Test Files 15 passed (15) / Tests 61 passed (61)
                                               ✓ tests/unit/revivePage.test.ts (3 tests) 43ms
npm run build                                → exit 0
```

Every number in the execution summary reproduces. CI on the branch: run `35842983066` — `frontend`, `harness-tooling`, `backend-integration`, `backend-unit` all `success`; `gh run list --branch …` shows both runs green. No executor gate failure.

### My own mutation on the *true wilted* path

The executor's mutation killed cases 1 and 2 but not 3, and reported that as expected "since the wilted path was unchanged". That reasoning holds — their mutation restores `main`'s branch structure, on which case 3 already passed — but it leaves the true alarm unproven. Two mutations of my own, each run as `npx vitest run tests/unit/revivePage.test.ts`:

| Mutation | Result |
| --- | --- |
| `pages/revive.vue:75` `<PlantSvg stage="wilted" …>` → `stage="sprout"` | **killed** — `2 failed \| 1 passed`, cases 2 and 3 fail on `expect(w.find('[data-stage="wilted"]').exists()).toBe(true)` |
| `:69` `v-else-if="pet.status"` → `v-else-if="pet.status && pet.error === null && false"` | **killed** — `2 failed \| 1 passed` |

So the true-alarm path is genuinely pinned: the fix removes the false alarm without unpinning the real one. A third mutation — moving the `pet.error` branch above the `pet.status` branch — **survived**, which is bug #1. `pages/revive.vue` restored after each; `git diff --quiet pages/revive.vue` clean, worktree `git status --porcelain` empty.

### Browser proof — judgement on the substitution, and my own run

The substitution is sound, and I agree it is the better instrument here. The Chrome extension was unavailable, and the Playwright *MCP* surface exposes no `page.route`, so scripted request interception was not on the table at all; a real HTTP responder also proves more than an interceptor would, because it exercises the built SPA's real fetch path, CORS, the `{error}` envelope → `ApiError` mapping, and a full navigation. (The executor's stated reason — that route overrides "can't survive a full navigation" — is not quite right for Playwright's own `page.route`, which does persist across navigations; the tooling gap is the real justification. The conclusion is unaffected.)

I did not take the claim on trust — I re-ran all three states independently, on the branch's production build, on my own ports, with a stub whose mode I flipped between navigations (preview on 3103, stub API on 3104, both stopped and verified free afterwards):

1. **Nothing listening on the API base** (the idea's exact repro): snapshot is `status → paragraph "Không tải được trạng thái cây. Chưa thể biết cây có héo hay không." + button "Thử lại"`. No `alert`, no plant image, no revive CTA. The reported bug is gone.
2. **Retry with the stub up in wilted mode** (clicking the real "Thử lại" button, not remounting): `alert: "⚠️ Cây xanh đang bị héo rũ!"`, `img "Cây đang ở giai đoạn wilted, máu 0%"`, `paragraph "Cây héo - 0%"`, `button "🚨 Cứu cây ngay (Quiz 15 phút)"`. The true alarm still fires, and the error→data transition works end to end in a browser.
3. **Stub in healthy mode** (`health_points: 85, stage: flowering`): `img "Cây đang ở giai đoạn flowering, máu 85%"` + `paragraph "Cây của bạn vẫn khỏe 🌱"`. No error card, no alarm.

All three states were genuinely exercised end to end, by me as well as by the executor.

## Quality

- **Pattern match with the other routes.** Same component and same idiom as `pages/index.vue:40` and `pages/roadmap.vue:30` — `StateBlock state="error" message="Không tải được…" action="Thử lại" @action="<store>.load()"`, with the store action called straight from the template. No fourth style invented. The one structural difference is deliberate and defensible: `/` and `/roadmap` order their branches `loading → error → data`, while `/revive` must put `passed` and `notWilted` first, so it ends up `data → error → loading` with the skeleton as the catch-all. Sibling pages spell the loading guard `loading && !data`; here "no status and no error" carries the same meaning and additionally covers the pre-mount tick.
- **Latent coupling worth knowing about.** `pet.error` is written by both `load()` (`stores/pet.ts:51`) and `revive()` (`:95`). Today a failed revive cannot surface the load-error card, because the user is inside the `pet.status` branch, which is ordered first — the page's own inline `role="alert"` message handles it. That safety comes entirely from branch order, which is what bug #1 asks to pin.
- **Accessibility / boundaries / CODEMAP.** Error card uses `role="status"` (polite) like the sibling pages; the alarm keeps `role="alert"`. No store, API-client or backend surface touched. The CODEMAP clause is accurate.
- **Repo configuration.** Three labels were created (`harness`, `type: bug`, `priority: high`) and applied to PR #6. `gh repo view` shows stock settings (default branch `main`, all three merge methods on, `deleteBranchOnMerge: false`), `rulesets` is empty, and the branch list contains only the five prior harness branches plus this one. Nothing else about the repo was altered.

## Bugs filed

- `harness/ideas/_inbox/no-test-pins-the-stale-data-wins-branch-order-on-revive-so-a.md` — **low**. The `pet.status`-before-`pet.error` order is the behaviour, and no test fails when it is reversed (mutation survived, 3/3 green). Not a blocker: the shipped code is correct.
- `harness/ideas/_inbox/the-revive-missed-days-line-renders-bo-hoc2-ngay-with-no-spa.md` — **low**, pre-existing on `main` (`revive.vue:83`, identical to `main:88`), surfaced by my browser run: Vue's `whitespace: 'condense'` eats the space, so the wilted headline reads "Bạn đã bỏ học2 ngày liên tiếp." Outside this plan's scope; moved verbatim by this diff.

Neither blocks the merge. No blocker filed: `python3 tools/harness/cli.py blockers --plan <plan>` exits 0.

## Verdict

**pass-with-bugs.** The plan delivers the idea, the diff is exactly the three files the plan named with no store change, every verification command reproduces on a clean shell, CI is green on the branch, and the true wilted alarm survives two mutations of mine while the false one is gone — confirmed in a browser against a production build, not only in jsdom. The two findings are low and neither touches the merge. Mark PR #6 ready; `/harness merge harness/plans/2026-09-23-revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md`.
