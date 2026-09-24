# Học 30 phút — Product Overview

> Learn English for 30 minutes a day, and keep a plant alive while you do it.

## What it is

Học 30 phút ("Study 30 minutes") is an installable web app (PWA) that helps Vietnamese-speaking adults build a daily English habit. Each learner gets an AI-generated 28-day study plan pitched at their level, a short set of tasks every day, and a virtual plant that grows while they practise and wilts when they stop.

It is not a course catalogue. It is a **habit engine**: the product's job is to get someone to study 30 minutes today, and again tomorrow.

## The problem

Most self-study learners don't fail because the material is bad. They fail because they stop.

- Motivation fades after the first week, and nothing notices when a learner quits.
- Generic courses are too easy or too hard, so learners get bored or lost.
- Study time isn't in the calendar, so it loses to everything else that is.

## How it solves it

| Problem | What the app does |
|---|---|
| Wrong level | A **placement quiz** graded by AI sets a CEFR level (A1–C2). |
| No plan | The AI builds a **28-day roadmap**: 4 modules × 7 days, each day 3 tasks of about 10 minutes (vocabulary/grammar, reading/listening, practice). |
| No time set aside | After onboarding it creates a **recurring 30-minute Google Calendar event** and adds a daily checklist to **Google Tasks**. |
| Forgetting | **Web Push reminders** at the learner's chosen time. |
| Quitting quietly | The **plant**: health 0–100 and a streak. |

### The plant

This is what keeps people coming back:

- **Hit 30 minutes today** → health +20 (max 100), streak +1.
- **Miss the day** → health −30. At 0 the plant is **wilted**.
- A wilted plant can be revived with a **15-minute recovery challenge**. Passing brings it back at 50 health, with the streak reset.
- It grows through stages: seed → sprout → sapling → flowering → fruitful.

Losing the plant hurts a little, but it can always be recovered. That makes a missed day a reason to come back rather than a reason to give up.

## The core user journey

1. **Sign in with Google.** One tap; this also grants Calendar and Tasks access.
2. **Onboard.** Take the placement quiz, set a goal and a reminder time. The roadmap is generated and the calendar event and tasks are created.
3. **Daily loop.** Open today's quest, do 3 tasks, reach 30 minutes, and see the plant grow.
4. **Miss a day** → the plant loses health → get reminded → come back → revive the plant if needed.

## Who it's for

- **Primary:** Vietnamese adults (students and young professionals) studying English on their own, for work, study abroad or exams, who have tried apps before and stopped.
- They have a phone, a Google account and a busy calendar, and can find 30 minutes a day if something holds them to it.

The interface is in Vietnamese (`lang: vi`), so the learner never has to struggle with English just to use the app.

## Why this can work as a business

- **Retention is the product.** Learning apps live or die on whether people keep coming back after the first week. Every feature here (the plant, the streak, the calendar block, reminders, revival) exists to improve that number.
- **Low cost to serve.** It's a PWA, so there are no app-store fees or review delays. The backend is one Go service with Postgres and Redis on Railway. AI calls are routed to the cheapest provider that does each job well (Gemini for plans and placement, DeepSeek for exercises, OpenAI for essay grading) and are capped at 5 calls per user per minute.
- **A daily reason to return.** The app sits in the learner's own Google Calendar and Tasks, so it's present every day without being opened.
- **Clear value to charge for.** Personalised plans, AI grading and a visible streak are things learners already pay for in other apps.

### What success looks like

These are the numbers to watch:

- **Daily target hit rate:** the share of active learners who reach 30 minutes on a given day.
- **D7 / D30 retention:** the share of learners still practising after 1 week and after 1 month.
- **Streak length:** median and distribution.
- **Revival rate:** the share of wilted plants that get revived rather than abandoned.
- **Roadmap completion:** the share who finish all 28 days.

## Open questions (not decided yet)

- **Pricing model.** None of the specs define it. Options include freemium (free plan and plant; paid AI grading, extra roadmaps, streak protection) or a flat monthly subscription.
- **What happens after day 28?** The next roadmap, and how the level is re-assessed.
- **Speaking practice.** An earlier concept, "Fluency Quest" (`project-base/draft-mvp-idea-for-mobile-ui.md`), explored an RPG-style app built around AI-judged speaking "boss raids". It is a possible later direction, not the current product.

## Where the details live

- Product and system spec: `project-base/1st-thinking-architecture-doc.md`
- Backend contracts and plant maths: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md`
- Screens and design system: `project-base/Adaptive English Learning Platform - Frontend Technical Specification.md`
