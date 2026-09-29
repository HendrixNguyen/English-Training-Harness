---
type: feature
status: planned
source: ideator
run: 2026-09-22-run-01
priority: low
plan: harness/plans/2026-09-27-spaced-repetition-vocabulary-review-in-the-daily-quest.md
---
# Spaced Repetition Vocabulary Review in the Daily Quest

## Why
The roadmap prompt (§6.1) generates 28 days of fresh vocabulary but never schedules a return to it, so by week 4 most week-1 items are forgotten and the CEFR progression the app promises is hollow — the user hits target minutes every day yet does not advance. The spacing literature is unambiguous: equal study time spread over expanding intervals beats massed exposure on retention tests taken days or weeks later, which is precisely the horizon a 4-module roadmap and a CEFR re-test operate on. Folding due reviews into the existing 10-minute Vocabulary/Grammar slot costs the user zero extra minutes, makes each day's quest feel personal (it contains *their* weak words), and gives us the first measurable learning signal (recall grades) beyond minutes spent — a founder can point at a retention curve of actual vocabulary, not just a streak.

## Expected output
User-visible:
- The Vocabulary/Grammar task in the daily quest opens with up to 8 "review" cards drawn from earlier days, each graded by the user (again / hard / good / easy), followed by the day's new items.
- Quest screen shows "N reviews due" before the task starts; pet view or quest screen shows a small "words retained" count.

Technical:
- New table `vocab_reviews (id UUID, user_id UUID, exercise_id UUID, item_key TEXT, interval_days INT, ease NUMERIC, due_on DATE, reps INT, last_grade SMALLINT, UNIQUE(user_id, item_key))` via store migration.
- `quests` package: when a `task_type='vocabulary'` exercise is completed, seed its items into `vocab_reviews` (due tomorrow); `GET /quests/daily` returns `reviews[]` for items with `due_on <= today` (user timezone), capped at 8; `POST /quests/progress` accepts `review_grades[]` and reschedules with an SM-2 style update. Unit tests for scheduling arithmetic and cap.
- No airouter call on the hot path; review items reuse `exercises.content_json`.
- CODEMAP `quests` paragraph updated.

## Evidence
- Spec §6.1 constraint 4 (10-minute Vocabulary/Grammar task per day); §3.1 `exercises` (task_type, content_json, is_completed); §1 CEFR-progression goal.
- CODEMAP: `quests` package (`GET /quests/daily`, `POST /quests/progress`), `store`.
- Spacing effect review in language learning: https://files.eric.ed.gov/fulltext/EJ1313692.pdf
- Optimising spaced repetition schedules (Tabibian et al.): https://pmc.ncbi.nlm.nih.gov/articles/PMC6410796/
- Reconsolidation account of long-timescale spacing: https://pmc.ncbi.nlm.nih.gov/articles/PMC5476736/

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low.**

*Is the Why real?* Yes — the roadmap never returns to earlier vocabulary, and recall grades would be the first learning signal beyond minutes.

*Achievable in one plan?* No, and it depends on an unbuilt foundation that *is* queued: `vocab_reviews` needs a stable `item_key` per vocabulary item, which only exists once task `content` has a schema — `2026-09-24-run-01/typed-task-content-with-answer-keys-…` (selected today, medium) defines `vocabulary → {words[{term, definition, example}]}`. After that it is still a table, SM-2 scheduling, two endpoint changes and a review UI — two plans.

*Priority.* Low now; re-rank to medium once typed content has landed.

_Evaluator, 2026-09-26 — **deferred** (not planned today)._ Depends on the typed content contract for a stable `item_key` per word; that branch merges tonight. Plan after it is on `main` and the learning room (retro plan 3) renders items — a review deck needs somewhere to be shown.

_Evaluator, 2026-09-27 — daily decide (feature queue, aged selected idea, planned today as **F2** — backend half)._ Checked: the typed-content contract is `done` on `origin/harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` (unmerged — the 2026-09-26 review run never ran): `airouter/content.go` defines `Word{Term, Definition, Example}` (`example` optional) inside `VocabularyContent{Words[], Questions[]}`, 5–8 words per vocabulary task, and `ParseRoadmap` enforces it, so a `vocabulary` exercise's `content_json.words[]` is a stable, validated list to seed from. `POST /quests/progress` on this checkout and on all three unmerged `quests` branches has **no `completed` flag** — every accepted progress call ends in `MarkComplete`, so "completion of a vocabulary exercise" = any accepted progress call whose exercise has `task_type = 'vocabulary'`; seeding is idempotent (`ON CONFLICT (user_id, item_key) DO NOTHING`), so a second call re-seeds nothing. **`item_key` = `<exercise_id>:<normalised term>`**, normalised = trimmed, lower-cased, inner whitespace collapsed to one space — the UUID makes it unique across roadmaps (a word re-taught by a later roadmap is a fresh card, accepted), the term makes it unique within an exercise, and it needs no new id from the model. Deviation from the idea's DDL: `term`, `definition`, `example` are copied into `vocab_reviews` at seed time so the due-list read is one query and outlives a roadmap swap (`exercises` rows are deactivated, not deleted, but the read should not depend on that). Scheduler is Anki-flavoured SM-2 (exact arithmetic in the plan): ease starts 2.50, floor 1.30; first intervals 1 → 6 days, then `interval × ease`; `again` resets reps and interval to 1. Dependency: the 2026-09-26 daily code PR (typed content + the three `quests` branches) must be on `origin/main` first — the plan gates on it. Migration number is "next free" (two unmerged branches already take `0004`), and the RLS convention from `0004_rls` applies. Frontend (the review deck inside the vocabulary task) is a separate plan after a design doc; this plan ships only the two additive wire changes. Stays `selected` / `low`; plan left `draft` for `/approve`.
