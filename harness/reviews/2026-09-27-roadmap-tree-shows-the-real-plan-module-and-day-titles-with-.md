---
plan: harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md
verdict: fail
bugs: [harness/ideas/_inbox/roadmap-page-never-refetches-today-goes-stale-and-a-new-lear.md, harness/ideas/_inbox/roadmap-hom-nay-chip-and-hoc-ngay-button-put-white-text-on-g.md, harness/ideas/_inbox/roadmap-tree-marks-day-28-as-today-forever-once-the-28-days-.md, harness/ideas/_inbox/growth-moment-plant-name-and-roadmap-tree-ui-is-built-on-kit.md]
---
# Review — Roadmap tree shows the real plan: module and day titles with true per-day completion

**Plan:** `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md`
**Branch/worktree:** `harness/2026-09-26-medium-roadmap-tree-shows-the-real-plan-module-and-day-titles-with-` / `.worktrees/roadmap-tree-shows-the-real-plan-module-and-day-titles-with-`
**Diff:** `git diff main...harness/2026-09-26-medium-roadmap-tree-shows-the-real-plan-module-and-day-titles-with- --stat`

## Plan vs idea
The read path is delivered and correct: `GET /api/v1/roadmap` returns the §6.2 outline joined with `daily_progress` by `DayDate` (the exact inverse of `DayNumber`, pinned across DST), no exercise content, and `/roadmap` renders four module headers, 28 named, dated days and the five truthful states. But the idea's promise — the tree "tells the truth about each day" — breaks on the page itself: the store is loaded once per page lifetime and never refetched, so today, minutes and "no roadmap" go stale in the SPA (blocker). The plan's decision 4 ("refetched on page open") is not implemented.

## Code vs plan
Branch head `f37d62b` (base `67ad0c0`); CI run 36229227097 green on that head.
- Tasks 1–4 (backend `DayDate`, repo reads, `Service.Roadmap`, handler/route/specs): followed. Deviations 1 (14 integration tests, not 13) and 2 (integration test landed in two commits) are justified.
- Task 5 (store + util): followed.
- Task 6 (page/components/sw): mostly followed. Deviation 3 (markers as row icons, not centred on the rail) is honest and visually minor. **Missing:** "refetched on page open" (blocker). **Skipped:** Step 7 "Look at it" — the executor proved the screen only through `roadmapPage.test.ts`; this review did the live check (below).
- Task 7 (CODEMAP): followed; accurate for what exists.

Re-run in a detached worktree at the branch head, isolated stack `COMPOSE_PROJECT_NAME=rv-frontend` (Postgres 5445, Redis 6395):
```
backend: env -u … make check -> fmt/vet silent, all packages ok (-race), exit 0
backend: TEST_DATABASE_URL=…:5445 TEST_REDIS_URL=…:6395 go test ./... -count=1 -p 1 -run TestIntegration -v
  -> 14 --- PASS incl. TestIntegrationRoadmapOutlineJoinsDailyProgress; 0 SKIP
go test ./internal/quests/ -run 'TestRoadmap|TestDayDate|TestFakeProgressBetween' -v | grep -c '^--- PASS' -> 15
grep -c '^func Test' internal/quests/roadmap_test.go -> 12 ; TestIntegration in quests -> 2
content in roadmap.go -> none ; main.go route -> 176 guarded.GET("/roadmap", …) only
service.go/service_test.go diff -> none ; handler_test.go -> 1 insertion
specs api/v1/roadmap -> 1 and 1 ; sw.ts registerRoute -> 3 ; "open question|before today" in utils/roadmap.ts -> none
quest-store import -> utils/roadmap.ts only ; stores/quest.ts + stores/pet.ts diff -> none
frontend: npm ci && lint && typecheck && test:unit && build -> 0 / 0 / 18 files, 106 tests passed / 0
CODEMAP matches -> 2 ; harness/ diff outside CODEMAP -> none ; cli.py validate -> 0
```
(`npm run test:e2e` not re-run; the executor's 3/3 `login.spec.ts` does not visit `/roadmap`.) Merge with `origin/main`: `frontend/components/roadmap/RoadmapNode.vue` conflicts (main changed the old pill's `text-white` → `text-ground-0`; this branch rewrote the file) and `harness/CODEMAP.md`.

**Screen walk (design `harness/designs/roadmap-tree.md`).** Branch `.output` on :3105 against a stub API (day 9 of 28; days 1 and 4 met, day 3 12 min, today 10 min), DOM-verified at 375 px (the pane's screenshots were stale, so geometry was read from `getBoundingClientRect`):
- Header: eyebrow "LỘ TRÌNH HỌC 28 NGÀY", roadmap title, "Trình độ B1 · Đã hoàn thành 2/28 ngày" ✓; four module headers with `n/7` ✓; 28 rows; no horizontal scroll (`scrollWidth` 375) ✓.
- Day states: 1 "★ · Ngày 1 · T3 1/9 · Đã hoàn thành", 2 "○ · Bỏ lỡ", 3 "◐ · 12/30 phút", 10 "🔒 · Chưa mở khóa" ✓; weekday correct (2026-09-01 is a Tuesday).
- Today: `aria-current="step"`, the only `aria-expanded="true"` row, tasks in vocabulary → reading → practice order (stub sent practice first) with "10 phút", "HÔM NAY" chip and "Học ngay →" ✓; scrolled into view ✓. Chip/CTA text is white on growth — 2.5:1 on this branch, 1.7:1 on main's kit token (bug, medium).
- **Staleness ✗:** server day moved 9 → 10, `router.push('/')` → `router.push('/roadmap')`: still `day-9` as today, no second `GET /api/v1/roadmap`. Stub switched from 404 to an active roadmap after the empty state had shown: the reopened page still says "Bạn chưa có lộ trình học." (blocker).
- Empty and error states: pinned by `roadmapPage.test.ts`; empty state seen live ✓.

## Quality
- Boundaries: `quests` reads only its own tables (`roadmaps`, `daily_progress`, `users.timezone`), no Redis, no pet tables; `airouter` as a production import is type-only and justified in the plan. Range read is one query per request with `BETWEEN` on the `(user_id, date)` unique index — fine.
- Correctness: stale store (blocker); day 28 stays "today" forever after the plan ends because `DayNumber` clamps (bug, low); corrupt/wrong-shape documents are a logged 500, not padded — good.
- Tests: honest; the DST round-trip and the content-leak key test are strong. Gaps from the read-only test-gap pass: the per-module day-count branch and a finished roadmap are untested (folded into the low bug); the `PgRepo` SQL error branches are untested, as elsewhere in the package (pre-existing pattern, not filed). No test mounts the page twice — which is how the blocker slipped through.
- UI kit: v1 build per its design; `text-white` on growth is an accessibility regression against main's kit (medium); the rest is kit v2 drift (shared low bug; `MapNode` is the v2 target).

## Bugs filed
- `harness/ideas/_inbox/roadmap-page-never-refetches-today-goes-stale-and-a-new-lear.md` (**high, blocker**) — `/roadmap` never refetches; stale today and a sticky "no roadmap".
- `harness/ideas/_inbox/roadmap-hom-nay-chip-and-hoc-ngay-button-put-white-text-on-g.md` (medium) — white on growth fails contrast.
- `harness/ideas/_inbox/roadmap-tree-marks-day-28-as-today-forever-once-the-28-days-.md` (low) — day 28 stuck as today; wrong-shape test gap.
- `harness/ideas/_inbox/growth-moment-plant-name-and-roadmap-tree-ui-is-built-on-kit.md` (low, shared) — kit v2 drift.

## Verdict
**fail** — one blocker. Not an executor gate failure (everything the executor claimed reproduces); it is a plan decision left unimplemented, only visible when the page is opened twice. The branch must not go into today's daily PR until the blocker's amend lands on it (a one-condition fix plus a page test).
