# Design: {title}

**Idea:** `{idea path}`
**Inherits:** `harness/designs/frontend-shell.md` (+ the screen's existing doc, if any)
**Kit:** `harness/UI-KIT.md` — tokens, components, states, copy, motion, flow
**Spec wireframe:** frontend spec §7.x — note every deliberate departure and why

## 0. Research
- **Learner's job on this screen** (one sentence).
- **The moment that earns the next minute** — what makes them continue.
- **What today does wrong** — evidence from the app, the idea or the spec.
- **Open questions and how they were answered** (spec, evidence, or the one question asked to the owner).

## 1. The signature
The one thing this screen does that a generic version would not. Describe it as the learner experiences it.

## 2. Flow
Entry points → this screen → exits. What changes on the screens the learner returns to.

## 3. Layout (mobile-first, `max-w-md`; desktop scaling in one line)
Region by region, top to bottom, with the kit component used for each and the data it reads (API field names).

## 4. States
Loading · empty · error · offline · first-time · done · reduced motion — each with its copy.

## 5. Interactions and feedback
Every control: what it does, what the learner sees within 100 ms, what is saved and where (store → endpoint).

## 6. Components
| Component | New/existing | Props / events | Notes |
Kit additions (tokens, components) with reasons — these land only through the plan.

## 7. Copy
Vietnamese, sentence case; the plant speaks as "tớ". List every string.

## 8. Self-critique
What was traded away, what the executor might get wrong, what to check in review.
