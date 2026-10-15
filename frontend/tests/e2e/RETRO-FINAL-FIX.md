# Scoped retro final-review proof

From `frontend/`, after `npm run build`, with installed Chromium at
`/usr/bin/chromium`:

```sh
LEARNING_PROOF_LOGS=/tmp/retro-final-fix node tests/e2e/retro-final-fix.mjs
LEARNING_PROOF_LOGS=/tmp/retro-final-fix node tests/e2e/retro-visual.mjs
```

Both runners start and stop their own production web server (3103 and 3102,
respectively) and browser. API requests are labelled route fixtures; no backend,
provider, external push delivery or Google consent is exercised. Use a fresh
output directory to distinguish each run's evidence.

`retro-final-fix.mjs` uses normal rendered controls and same-session route
navigation to assert an older deferred settings GET cannot replace a confirmed
09:15 save, while a separate dirty 10:45 edit survives an accepted 07:30 refresh.
It then checks fresh cached roadmap entry online and offline initializes and
positions day 18 in region 3, and measures the centered 28rem maximum column
with 16px inner gutters at 375px and 1280px. It writes `focused-results.json` and
a desktop screenshot. It fails on lost settings truth, dirty draft, current-day
positioning or column geometry.

`retro-visual.mjs` retains its 45 state/keyboard/zoom captures, including seven
at CSS zoom 2. Its roadmap checks assert region and container geometry at both
viewports, and unchanged mounted history refresh retains disclosure and scroll.
Its hub offline check requires exactly one visible existing notice alongside
cached quest navigation before capture. Overflow and control visibility gates
remain in place.

Unit regressions additionally cover overlapping read ordering/loading, reads
begun during a save, legitimate later refresh, rejected prior-owner saves,
changed roadmap date and completed/no-today entry starting at the top. Screenshots
are artifacts for visual review, not a claim that every pixel was manually audited.
