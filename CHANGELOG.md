# Changelog

What shipped to production, newest first. One section per release; the
section is written by the release PR into `production`
(`.agents/routines/daily-ship.md`, or the owner for a hotfix), and a green
`Deploy` run tags that commit `v<version>` with a GitHub Release carrying the
same text. Rolling back = redeploying an older tag (`deploy/README.md`
"Versions and rollback").

Versions follow [Semantic Versioning](https://semver.org/): **minor** when a
release carries a feature or MVP slice, **patch** when it carries only bug
fixes, docs or tooling, **major** only by owner decision.

## [0.3.0] - 2026-10-13

### Added
- Complete one reading task with Vietnamese glossary and saved progress (`harness/plans/2026-10-11-reading-flow.md`).

### Fixed
- Learning sessions save graded answers and explain each question (`harness/plans/2026-10-12-learning-sessions-discard-answers-and-hide-feedback-while-un.md`).

### Changed
- Restore the missing v0.2.0 changelog record while preserving existing release tags.

## [0.2.0] - 2026-09-29

Historical release at `2d9cd6393377578e18281aa74fbb330b2711597d`, tagged and published on 2026-09-29. Its changelog section was missing; this record restores the released version without changing the tag. The original GitHub Release has no release notes.

## [0.1.0] - 2026-09-25

Baseline: what was on `production` (`61224f6`) before releases were
versioned. Not tagged.

### Added
- The nine MVP slices and every fix merged to `main` up to `61224f6` (the
  `type: mvp-slice`, `feature` and `bug` plans in `harness/plans/` marked
  done by then).
- Shipping from the `production` branch (`deploy.yml` + nightly ship routine).
