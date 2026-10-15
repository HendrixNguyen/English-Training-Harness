# Design: Retro final review amendments

**Scope:** Existing-design confirmation for final-review findings I1, I2, M1, M2 and M3 on the approved remaining-retro-screens branch. No feature or redesign.
**Inherits:** `harness/designs/frontend-shell.md`, `harness/designs/retro-roadmap.md`, `harness/designs/retro-settings.md`, and binding `harness/designs/retro-screens-compatibility.md`.
**Kit:** `harness/UI-KIT.md` v2 wins over obsolete frontend-shell/spec §6 tokens; frontend spec §7.2/§7.4 route intent remains.
**Evidence:** `.superpowers/sdd/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded/final-review.md`; production settings race reproduction and roadmap geometry in `/workspace/cloud-setup/retro-screens/final-review/focused-results.json`. This confirms supplied evidence; it is not a new visual audit.

## 0. Research
The learner's job is to trust their reminder choice, find today's place on the journey, and understand what remains available offline. The next minute is earned when returning to study requires no reconstruction of context. Reverting a confirmed save, stretching the map across desktop, and silently showing cached state undermine that trust. The five review findings and inherited contracts resolve scope without an owner interview.

## 1. Signature and flow
Keep the existing inn, four-region map and companion hub. Settings entry refresh reads real preferences; editing remains local until explicit save, and confirmed server save results remain truthful. Leaving and newly entering the roadmap initializes today's region once from an accepted same-owner snapshot. A refresh while the learner is already exploring history preserves that disclosure and viewport when roadmap/date are unchanged. A changed roadmap/date follows the existing reset contract. Hub offline entry retains accepted cached tasks with one visible explanation.

## 2. Layout and states
Restore the roadmap's single centered `max-w-md` (28rem) column with 16px inner side gutters on mobile and desktop. Keep the existing header, four region panels, three-column winding path, readable locked nodes, detail panel and navigation controls. Retain usable controls and visible focus at 200% zoom; do not add map zoom controls or a new layout system.

Loading, empty, failure/retry, first-day, completed journey, push capability and Google states continue to follow the compatibility addendum. A settings refresh may update accepted saved information without replacing a dirty time field or falsely treating that edit as saved. A delayed read begun before a successful save cannot restore the older saved-time claim. New offline roadmap entry discloses today's detail and region from the accepted cached snapshot, while clearly identifying that snapshot as cached; when no day matches its `today`, do not invent a current day. Reduced motion keeps disclosure and positioning understandable without introducing animation.

## 3. Components, data and copy
Reuse existing `RetroPanel`, `RetroButton`, `MapNode`, `CompanionSprite`, `StateBlock`, hub offline notice and native time field; no new props/events or kit additions are required by this amendment. Existing select/enter, retry, save and navigation actions remain.

Settings reads `GET /api/v1/settings` fields `notification_time`, `timezone`, `next_reminder_at`, `push_configured`; explicit save uses the existing `POST /api/v1/settings/notifications` contract and confirmed `notification_time`/`next_reminder_at`. Roadmap uses the existing owned `GET /api/v1/roadmap` response, including `roadmap_id`, `today`, `modules[].days[].date`, measured progress and task metadata. No new API or persistence contract.

Reuse existing Vietnamese status/error copy. Roadmap offline: “Đang ngoại tuyến — bản đồ là bản đã lưu.” Settings offline: “Cần mạng để lưu cài đặt.” Preserve the existing hub offline notice verbatim and once; no invented success/delivery claim. No external OAuth consent or push send is part of this amendment or its proof.

## 4. Designer acceptance
1. **I1 — reminder truth:** On settings re-entry with an older read pending, editing 07:30 to 09:15 and successfully saving leaves both input and saved-time status at the server-confirmed 09:15 after the older read resolves. A separate edit made before an entry refresh resolves remains in the input, distinct from accepted saved status; a later genuine refresh can still update saved preferences without silently discarding an edit or newer confirmed save.
2. **I2 — common column:** At 375px and 1280px viewport widths, the map occupies the centered column capped at 28rem with 16px inner side gutters. Headers, panels and focus outlines have breathing room; at 200% zoom, text and existing controls remain readable, reachable and unclipped. Actual geometry demonstrates the column/gutters, not merely lack of horizontal overflow.
3. **M1 — entry versus refresh:** With an accepted cached map containing today in a later region, a fresh online or offline route entry shows today's detail and positions that region in view. Offline entry labels the cached map. Selecting a historical day and refreshing the unchanged mounted map retains that selection and viewport; a journey without a matching today gains no fabricated current-day marker.
4. **M2 — accurate design handoff:** The CODEMAP shell overview describes the current owned per-day roadmap read and retro hub presentation consistently with its newer route paragraphs. It makes no current claim that completion is inferred solely from `day_number`, that no per-day endpoint exists, or that removed hub visuals are still the active design.
5. **M3 — observable offline proof:** The production-browser hub offline case demonstrates retained cached tasks together with exactly one visible existing offline notice before capture. Evidence would fail if the notice vanished; retained task content alone does not establish this state.

## 5. Self-critique
This amendment restores existing intent rather than adding conflict-resolution controls or broader offline support. The chief traps are suppressing legitimate preference refreshes to hide the save race, moving the learner away from history on every refresh, and passing width-only checks while leaving the map uncentered. CODEMAP accuracy and the hub assertion are handoff/proof corrections, not new learner-facing interactions. The controller owns plan links, metadata and commits; this document does not approve merge or release.
