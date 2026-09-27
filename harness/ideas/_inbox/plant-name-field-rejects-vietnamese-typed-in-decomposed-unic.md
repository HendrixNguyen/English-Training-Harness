---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Plant-name field rejects Vietnamese typed in decomposed Unicode, including the suggested 'Mầm Non'

## Why
The onboarding plant-name field validates with `/^[\p{L}\p{N} ]{1,30}$/u` (`frontend/pages/onboarding.vue:31`). Vietnamese typed or pasted in decomposed Unicode (NFD — e.g. Unikey/EVKey in "Unicode tổ hợp" mode, text copied from some macOS sources) carries its tone marks as combining characters (`\p{M}`), which the class rejects. Reproduced in a browser on the branch build: `'Mầm Non'.normalize('NFD')` → start button disabled, `aria-invalid="true"`, note "Tên cây dài 1–30 ký tự, chỉ gồm chữ, số và dấu cách." — the suggested default name itself is refused. The server counts runes, so an NFD name also spends up to twice the length budget.

## Expected output
Normalise to NFC (`trim().normalize('NFC')`) before validating and before sending `plant_name`; allow `\p{M}` after a letter if normalisation is not enough. Test: the NFD form of "Mầm Non" is accepted and sent as the NFC string. Optionally normalise server-side too (`golang.org/x/text/unicode/norm` is not a dependency today — keep it client-side unless the evaluator wants the dependency).

## Evidence
- Plan: `harness/plans/2026-09-25-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md`, design `harness/designs/plant-name.md` §1 ("any script's letters and digits").
- `node -e 'const R=/^[\p{L}\p{N} ]{1,30}$/u; console.log(R.test("Mầm Non".normalize("NFD")))'` → `false`.
