# Agent Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the four-role, file-driven agent harness described in `docs/superpowers/specs/2026-09-22-agent-harness-design.md`, so `/ideate --mvp` can start building the English-learning app.

**Architecture:** Tool-neutral role prompts and skills live in `.agents/`; runtime state is markdown-with-frontmatter under `harness/`; a small stdlib-only Python CLI (`tools/harness/`) owns every state mutation (validate, scan, STATE.md, status changes, locks) so agents run commands instead of hand-editing YAML. `.claude/` is a thin adapter: agent files point at roles, commands invoke skills.

**Tech Stack:** Python 3 stdlib (`unittest`, no pip deps), git worktrees, `gh` CLI, Claude Code agents/skills/commands.

**Spec addendum (decided while planning):** plan frontmatter also carries `priority:` and `order:` copied from the idea, so `next execute` can rank plans without opening idea files.

---

## File structure

```
AGENTS.md                              canonical instructions (tool-neutral)
CLAUDE.md                              existing; add @AGENTS.md import line
.gitignore                             add .worktrees/, harness/.lock, __pycache__/
tools/harness/
  __init__.py
  frontmatter.py                       parse/serialize YAML subset
  schema.py                            per-type required keys + allowed values
  scan.py                              walk harness/, load artifacts, validate
  state.py                             render STATE.md from scan
  cli.py                               argparse entry: validate|state|new-run|new-idea|new-plan|new-review|set|next|lock|unlock|slug|stale-worktrees
  tests/test_frontmatter.py
  tests/test_schema.py
  tests/test_scan_state.py
  tests/test_cli.py
.agents/
  templates/idea.md, plan.md, review.md, run.md
  roles/ideator.md, evaluator.md, executor.md, reviewer.md
  skills/harness-ideate/SKILL.md
  skills/harness-evaluate/SKILL.md
  skills/harness-execute/SKILL.md
  skills/harness-review/SKILL.md
  skills/harness-orchestrate/SKILL.md
.claude/
  skills -> ../.agents/skills
  agents/harness-ideator.md, harness-evaluator.md, harness-executor.md, harness-reviewer.md
  commands/ideate.md, idea.md, evaluate.md, approve.md, execute.md, review.md, harness.md
harness/
  ideas/_inbox/.gitkeep  plans/.gitkeep  designs/.gitkeep  reviews/.gitkeep  runs/.gitkeep
  CODEMAP.md
  STATE.md                             generated
```

Run all tests at any time with: `python3 -m unittest discover -s tools/harness/tests -v`

---

### Task 1: Repository skeleton and AGENTS.md

**Files:**
- Create: `AGENTS.md`
- Modify: `CLAUDE.md` (prepend import)
- Modify: `.gitignore`
- Create: `harness/ideas/_inbox/.gitkeep`, `harness/plans/.gitkeep`, `harness/designs/.gitkeep`, `harness/reviews/.gitkeep`, `harness/runs/.gitkeep`
- Create: `tools/harness/__init__.py`, `tools/harness/tests/__init__.py`
- Create symlink: `.claude/skills -> ../.agents/skills`

- [ ] **Step 1: Create folders**

```bash
mkdir -p harness/ideas/_inbox harness/plans harness/designs harness/reviews harness/runs \
         tools/harness/tests .agents/roles .agents/templates .agents/skills .claude/agents .claude/commands
touch harness/ideas/_inbox/.gitkeep harness/plans/.gitkeep harness/designs/.gitkeep harness/reviews/.gitkeep harness/runs/.gitkeep
touch tools/harness/__init__.py tools/harness/tests/__init__.py
ln -s ../.agents/skills .claude/skills
```

- [ ] **Step 2: Write AGENTS.md**

```markdown
# AGENTS.md

Canonical instructions for any coding agent (Claude Code, Codex, Gemini CLI, …) working in this repository. Tool-specific files (`CLAUDE.md`, `GEMINI.md`) import this file.

## What this repo is

An adaptive English-learning PWA (spec: `project-base/1st-thinking-architecture-doc.md`) built and evolved by an **agent harness** (design: `docs/superpowers/specs/2026-09-22-agent-harness-design.md`). App code lives in `frontend/` (Nuxt 3) and `backend/` (Go modular monolith) once the MVP slices land.

## The harness in one paragraph

Four roles in `.agents/roles/` — ideator, evaluator, executor, reviewer — pass markdown artifacts through `harness/`. Status lives in each file's frontmatter; `harness/STATE.md` is a generated dashboard. All state changes go through `python3 tools/harness/cli.py` — never hand-edit frontmatter. Read `harness/CODEMAP.md` before exploring code.

## Rules every role follows

- Never modify the main checkout's app code; executors work in `.worktrees/<slug>` on `harness/*` branches.
- Never merge to `main`. Only a human runs `/harness merge`.
- Pushing `harness/*` branches and opening Draft PRs is allowed without asking. Pushing `main` is not.
- Token discipline: CODEMAP → `rg` with tight patterns → read only matched ranges. Never `cat` a directory.
- Files under `.agents/` and `harness/` are tool-neutral: say "spawn the evaluator role" or "load skill harness-evaluate", never name a specific tool.
- This project's remote is **GitHub** (`gh`, pull requests).

## Commands (Claude Code adapter)

`/ideate [--mvp]`, `/idea "<text>"`, `/evaluate [<idea>|--run <run>]`, `/approve <plan>`, `/execute [<plan>]`, `/review [<plan>]`, `/harness status|run|merge|prune`.

## Harness CLI quick reference

```
python3 tools/harness/cli.py validate                 # exit 1 on malformed artifacts
python3 tools/harness/cli.py state                    # regenerate + print STATE.md
python3 tools/harness/cli.py new-run [--mvp]          # -> harness/ideas/<date>-run-NN/
python3 tools/harness/cli.py new-idea --run DIR --title T --type feature|bug|mvp-slice --source ideator|human|reviewer [--order N]
python3 tools/harness/cli.py new-plan --idea FILE     # -> harness/plans/<date>-<slug>.md (status draft)
python3 tools/harness/cli.py new-review --plan FILE --verdict pass|pass-with-bugs|fail
python3 tools/harness/cli.py set FILE key=value ...   # frontmatter update with validation
python3 tools/harness/cli.py next --stage evaluate|execute|review
python3 tools/harness/cli.py lock PLAN | unlock
python3 tools/harness/cli.py slug "Title"
python3 tools/harness/cli.py stale-worktrees
```
```

- [ ] **Step 3: Prepend import to CLAUDE.md and extend .gitignore**

Add as the first line after the `# CLAUDE.md` heading block in `CLAUDE.md`:

```markdown
@AGENTS.md
```

Append to `.gitignore`:

```
__pycache__/
*.pyc
```

(`.worktrees/` and `harness/.lock` are already present.)

- [ ] **Step 4: Verify and commit**

Run: `ls -la .claude/skills && git status --short`
Expected: symlink resolves to `../.agents/skills`; new files listed.

```bash
git add -A
git commit -m "chore: harness skeleton, AGENTS.md, tool-neutral layout"
```

---

### Task 2: Frontmatter parser

**Files:**
- Create: `tools/harness/frontmatter.py`
- Test: `tools/harness/tests/test_frontmatter.py`

Supports the YAML subset we emit: `key: scalar`, `key: "quoted"`, `key: [a, b]`, `key: true|false`, ints, and `key:` (null). Preserves key order. No PyYAML.

- [ ] **Step 1: Write the failing tests**

```python
# tools/harness/tests/test_frontmatter.py
import unittest
from tools.harness.frontmatter import parse, serialize, split_document, join_document

DOC = """---
type: feature
status: proposed
priority: high
order: 3
merged: false
bugs: [a.md, b.md]
title: "Spaced: repetition"
empty:
---
# Body

text
"""

class ParseTests(unittest.TestCase):
    def test_split_returns_frontmatter_and_body(self):
        fm, body = split_document(DOC)
        self.assertTrue(fm.startswith("type:"))
        self.assertEqual(body, "# Body\n\ntext\n")

    def test_parse_types(self):
        fm, _ = split_document(DOC)
        d = parse(fm)
        self.assertEqual(d["type"], "feature")
        self.assertEqual(d["order"], 3)
        self.assertIs(d["merged"], False)
        self.assertEqual(d["bugs"], ["a.md", "b.md"])
        self.assertEqual(d["title"], "Spaced: repetition")
        self.assertIsNone(d["empty"])

    def test_parse_preserves_order(self):
        fm, _ = split_document(DOC)
        self.assertEqual(list(parse(fm).keys())[:3], ["type", "status", "priority"])

    def test_no_frontmatter(self):
        fm, body = split_document("# just body\n")
        self.assertEqual(fm, "")
        self.assertEqual(body, "# just body\n")

class SerializeTests(unittest.TestCase):
    def test_roundtrip(self):
        fm, body = split_document(DOC)
        d = parse(fm)
        again = parse(serialize(d))
        self.assertEqual(d, again)

    def test_quotes_values_with_colon(self):
        self.assertIn('title: "a: b"', serialize({"title": "a: b"}))

    def test_join_document(self):
        out = join_document({"a": 1}, "body\n")
        self.assertEqual(out, "---\na: 1\n---\nbody\n")

if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run to verify failure**

Run: `python3 -m unittest tools.harness.tests.test_frontmatter -v`
Expected: `ModuleNotFoundError: No module named 'tools.harness.frontmatter'`

- [ ] **Step 3: Implement**

```python
# tools/harness/frontmatter.py
"""Minimal YAML-subset frontmatter: scalars, quoted strings, inline lists, bools, ints, null."""
import re

_FENCE = "---"


def split_document(text):
    """Return (frontmatter_text, body). frontmatter_text is '' when absent."""
    if not text.startswith(_FENCE + "\n"):
        return "", text
    end = text.find("\n" + _FENCE + "\n", len(_FENCE))
    if end == -1:
        return "", text
    fm = text[len(_FENCE) + 1:end + 1]
    body = text[end + len(_FENCE) + 2:]
    return fm, body


def _scalar(raw):
    raw = raw.strip()
    if raw == "":
        return None
    if raw.startswith('"') and raw.endswith('"') and len(raw) >= 2:
        return raw[1:-1].replace('\\"', '"')
    if raw.startswith("[") and raw.endswith("]"):
        inner = raw[1:-1].strip()
        return [] if not inner else [_scalar(x) for x in inner.split(",")]
    if raw == "true":
        return True
    if raw == "false":
        return False
    if re.fullmatch(r"-?\d+", raw):
        return int(raw)
    return raw


def parse(fm_text):
    out = {}
    for line in fm_text.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        key, sep, val = line.partition(":")
        if not sep:
            continue
        out[key.strip()] = _scalar(val)
    return out


def _emit(v):
    if v is None:
        return ""
    if isinstance(v, bool):
        return "true" if v else "false"
    if isinstance(v, int):
        return str(v)
    if isinstance(v, list):
        return "[" + ", ".join(_emit(x) for x in v) + "]"
    s = str(v)
    if s == "" or any(c in s for c in ':#[]",') or s in ("true", "false") or re.fullmatch(r"-?\d+", s):
        return '"' + s.replace('"', '\\"') + '"'
    return s


def serialize(d):
    return "".join(f"{k}: {_emit(v)}".rstrip() + "\n" for k, v in d.items())


def join_document(d, body):
    return f"{_FENCE}\n{serialize(d)}{_FENCE}\n{body}"
```

- [ ] **Step 4: Run tests**

Run: `python3 -m unittest tools.harness.tests.test_frontmatter -v`
Expected: 7 tests, `OK`

- [ ] **Step 5: Commit**

```bash
git add tools/harness/frontmatter.py tools/harness/tests/test_frontmatter.py
git commit -m "feat(harness): frontmatter parser/serializer"
```

---

### Task 3: Artifact schemas

**Files:**
- Create: `tools/harness/schema.py`
- Test: `tools/harness/tests/test_schema.py`

- [ ] **Step 1: Write the failing tests**

```python
# tools/harness/tests/test_schema.py
import unittest
from tools.harness.schema import validate, kind_of, TRANSITIONS

class SchemaTests(unittest.TestCase):
    def test_kind_from_path(self):
        self.assertEqual(kind_of("harness/ideas/2026-09-22-run-01/x.md"), "idea")
        self.assertEqual(kind_of("harness/ideas/_inbox/x.md"), "idea")
        self.assertEqual(kind_of("harness/plans/x.md"), "plan")
        self.assertEqual(kind_of("harness/reviews/x.md"), "review")
        self.assertIsNone(kind_of("harness/ideas/2026-09-22-run-01/_run.md"))
        self.assertIsNone(kind_of("harness/STATE.md"))

    def test_valid_idea(self):
        errs = validate("idea", {"type": "feature", "status": "proposed", "source": "ideator", "run": "2026-09-22-run-01"})
        self.assertEqual(errs, [])

    def test_idea_missing_and_bad_values(self):
        errs = validate("idea", {"type": "epic", "status": "proposed"})
        self.assertIn("type: 'epic' not in", " ".join(errs))
        self.assertIn("missing key: source", errs)
        self.assertIn("missing key: run", errs)

    def test_rejected_needs_reason(self):
        errs = validate("idea", {"type": "bug", "status": "rejected", "source": "reviewer", "run": "r"})
        self.assertIn("rejected idea needs rejected_reason", errs)

    def test_mvp_slice_needs_order(self):
        errs = validate("idea", {"type": "mvp-slice", "status": "proposed", "source": "ideator", "run": "r"})
        self.assertIn("mvp-slice idea needs integer order", errs)

    def test_valid_plan(self):
        self.assertEqual(validate("plan", {"idea": "harness/ideas/r/x.md", "status": "draft", "priority": "high", "merged": False}), [])

    def test_valid_review(self):
        self.assertEqual(validate("review", {"plan": "harness/plans/x.md", "verdict": "pass", "bugs": []}), [])
        self.assertIn("verdict: 'meh' not in", " ".join(validate("review", {"plan": "p", "verdict": "meh", "bugs": []})))

    def test_transitions(self):
        self.assertIn("approved", TRANSITIONS["plan"]["draft"])
        self.assertNotIn("done", TRANSITIONS["plan"]["draft"])
        self.assertIn("approved", TRANSITIONS["plan"]["failed"])

if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run to verify failure**

Run: `python3 -m unittest tools.harness.tests.test_schema -v`
Expected: `ModuleNotFoundError`

- [ ] **Step 3: Implement**

```python
# tools/harness/schema.py
"""Required keys, allowed values, and legal status transitions per artifact kind."""

ENUMS = {
    "idea": {
        "type": ["feature", "bug", "mvp-slice"],
        "status": ["proposed", "selected", "rejected", "planned"],
        "source": ["ideator", "human", "reviewer"],
        "priority": ["high", "medium", "low"],
    },
    "plan": {
        "status": ["draft", "approved", "executing", "done", "failed"],
        "priority": ["high", "medium", "low"],
    },
    "review": {
        "verdict": ["pass", "pass-with-bugs", "fail"],
    },
}

REQUIRED = {
    "idea": ["type", "status", "source", "run"],
    "plan": ["idea", "status", "priority", "merged"],
    "review": ["plan", "verdict", "bugs"],
}

TRANSITIONS = {
    "idea": {
        "proposed": ["selected", "rejected"],
        "selected": ["planned", "rejected"],
        "rejected": [],
        "planned": [],
    },
    "plan": {
        "draft": ["approved"],
        "approved": ["executing", "draft"],
        "executing": ["done", "failed"],
        "done": [],
        "failed": ["approved"],
    },
}


def kind_of(path):
    p = str(path).replace("\\", "/")
    name = p.rsplit("/", 1)[-1]
    if not name.endswith(".md") or name.startswith("_") or name.startswith("."):
        return None
    if "/harness/ideas/" in "/" + p or p.startswith("harness/ideas/"):
        return "idea"
    if "/harness/plans/" in "/" + p or p.startswith("harness/plans/"):
        return "plan"
    if "/harness/reviews/" in "/" + p or p.startswith("harness/reviews/"):
        return "review"
    return None


def validate(kind, fm):
    errs = []
    for k in REQUIRED[kind]:
        if k not in fm or fm[k] is None:
            errs.append(f"missing key: {k}")
    for k, allowed in ENUMS.get(kind, {}).items():
        if k in fm and fm[k] is not None and fm[k] not in allowed:
            errs.append(f"{k}: '{fm[k]}' not in {allowed}")
    if kind == "idea":
        if fm.get("status") == "rejected" and not fm.get("rejected_reason"):
            errs.append("rejected idea needs rejected_reason")
        if fm.get("type") == "mvp-slice" and not isinstance(fm.get("order"), int):
            errs.append("mvp-slice idea needs integer order")
        if fm.get("status") == "planned" and not fm.get("plan"):
            errs.append("planned idea needs plan path")
    if kind == "plan" and not isinstance(fm.get("merged"), bool):
        errs.append("merged must be true/false")
    if kind == "review" and not isinstance(fm.get("bugs"), list):
        errs.append("bugs must be a list")
    return errs
```

- [ ] **Step 4: Run tests**

Run: `python3 -m unittest tools.harness.tests.test_schema -v`
Expected: 8 tests, `OK`

- [ ] **Step 5: Commit**

```bash
git add tools/harness/schema.py tools/harness/tests/test_schema.py
git commit -m "feat(harness): artifact schemas and transitions"
```

---

### Task 4: Scanner and STATE.md renderer

**Files:**
- Create: `tools/harness/scan.py`
- Create: `tools/harness/state.py`
- Test: `tools/harness/tests/test_scan_state.py`

- [ ] **Step 1: Write the failing tests**

```python
# tools/harness/tests/test_scan_state.py
import os, tempfile, unittest, pathlib
from tools.harness.scan import scan
from tools.harness.state import render_state

def w(root, rel, text):
    p = pathlib.Path(root, rel); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(text); return p

IDEA = "---\ntype: feature\nstatus: {status}\nsource: ideator\nrun: r1\n{extra}---\n# {title}\n\n## Why\nx\n"
PLAN = "---\nidea: harness/ideas/r1/{slug}.md\nstatus: {status}\npriority: {prio}\nmerged: false\n{extra}---\n# Plan {slug}\n"

class ScanTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.root = self.tmp.name
        w(self.root, "harness/ideas/r1/_run.md", "# run\n")
        w(self.root, "harness/ideas/r1/alpha.md", IDEA.format(status="proposed", extra="", title="Alpha"))
        w(self.root, "harness/ideas/_inbox/bugone.md", IDEA.format(status="proposed", extra="", title="Bug one").replace("feature", "bug").replace("ideator", "reviewer"))
        w(self.root, "harness/ideas/r1/broken.md", "---\ntype: nope\n---\n# Broken\n")
        w(self.root, "harness/plans/2026-09-22-alpha.md", PLAN.format(slug="alpha", status="approved", prio="high", extra=""))
        w(self.root, "harness/plans/2026-09-22-beta.md", PLAN.format(slug="beta", status="done", prio="low", extra=""))
    def tearDown(self): self.tmp.cleanup()

    def test_scan_groups_and_titles(self):
        r = scan(self.root)
        self.assertEqual({a.rel for a in r.ideas}, {"harness/ideas/r1/alpha.md", "harness/ideas/_inbox/bugone.md"})
        self.assertEqual([a.rel for a in r.invalid], ["harness/ideas/r1/broken.md"])
        self.assertEqual(next(a for a in r.ideas if a.rel.endswith("alpha.md")).title, "Alpha")
        self.assertEqual(len(r.plans), 2)

    def test_render_state_sections(self):
        s = render_state(scan(self.root))
        for h in ["## Invalid", "## Inbox", "## Proposed", "## Approved", "## Done", "## Failed"]:
            self.assertIn(h, s)
        self.assertIn("broken.md", s.split("## Invalid")[1].split("##")[0])
        self.assertIn("bugone.md", s.split("## Inbox")[1].split("##")[0])
        self.assertIn("2026-09-22-alpha.md", s.split("## Approved")[1].split("##")[0])

    def test_done_without_review_flagged(self):
        s = render_state(scan(self.root))
        self.assertIn("beta.md", s)
        self.assertIn("(unreviewed)", s)

if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run to verify failure**

Run: `python3 -m unittest tools.harness.tests.test_scan_state -v`
Expected: `ModuleNotFoundError`

- [ ] **Step 3: Implement scan.py**

```python
# tools/harness/scan.py
"""Walk harness/, parse every artifact, validate, return a ScanResult."""
import pathlib
from dataclasses import dataclass, field
from .frontmatter import split_document, parse
from .schema import kind_of, validate


@dataclass
class Artifact:
    rel: str
    kind: str
    fm: dict
    body: str
    title: str
    errors: list = field(default_factory=list)


@dataclass
class ScanResult:
    root: str
    ideas: list = field(default_factory=list)
    plans: list = field(default_factory=list)
    reviews: list = field(default_factory=list)
    invalid: list = field(default_factory=list)

    def reviews_for(self, plan_rel):
        return [r for r in self.reviews if r.fm.get("plan") == plan_rel]


def _title(body):
    for line in body.splitlines():
        if line.startswith("# "):
            return line[2:].strip()
    return "(untitled)"


def load(root, rel):
    text = pathlib.Path(root, rel).read_text()
    fm_text, body = split_document(text)
    fm = parse(fm_text)
    kind = kind_of(rel)
    errs = validate(kind, fm) if kind else ["unknown artifact kind"]
    if not fm_text:
        errs.insert(0, "no frontmatter")
    return Artifact(rel=rel, kind=kind or "?", fm=fm, body=body, title=_title(body), errors=errs)


def scan(root="."):
    res = ScanResult(root=root)
    base = pathlib.Path(root, "harness")
    for p in sorted(base.rglob("*.md")):
        rel = p.relative_to(root).as_posix()
        if kind_of(rel) is None:
            continue
        art = load(root, rel)
        if art.errors:
            res.invalid.append(art)
        else:
            getattr(res, art.kind + "s").append(art)
    return res
```

- [ ] **Step 4: Implement state.py**

```python
# tools/harness/state.py
"""Render harness/STATE.md from a ScanResult. Never hand-edited."""
import datetime

PRIO_RANK = {"high": 0, "medium": 1, "low": 2, None: 3}


def _line(a, extra=""):
    p = a.fm.get("priority")
    tag = f" [{p}]" if p else ""
    return f"- `{a.rel}` — {a.title}{tag}{extra}"


def plan_sort_key(a):
    return (PRIO_RANK.get(a.fm.get("priority"), 3), a.fm.get("order") or 999, a.rel)


def render_state(res):
    now = datetime.datetime.now().strftime("%Y-%m-%d %H:%M")
    out = [f"# Harness state\n", f"_Generated {now}. Do not edit — run `python3 tools/harness/cli.py state`._\n"]

    def section(title, items, empty="_none_"):
        out.append(f"\n## {title}\n")
        out.extend(items if items else [empty])

    section("Invalid", [f"- `{a.rel}` — " + "; ".join(a.errors) for a in res.invalid])
    section("Inbox", [_line(a) for a in res.ideas if "/_inbox/" in a.rel])
    section("Proposed", [_line(a) for a in res.ideas if a.fm["status"] == "proposed" and "/_inbox/" not in a.rel])
    section("Selected", [_line(a) for a in res.ideas if a.fm["status"] == "selected"])
    section("Planned (awaiting approval)", [_line(a) for a in sorted(res.plans, key=plan_sort_key) if a.fm["status"] == "draft"])
    section("Approved", [_line(a) for a in sorted(res.plans, key=plan_sort_key) if a.fm["status"] == "approved"])
    section("Executing", [_line(a) for a in res.plans if a.fm["status"] == "executing"])
    done = [a for a in res.plans if a.fm["status"] == "done"]
    done_lines = []
    for a in sorted(done, key=lambda x: x.rel, reverse=True)[:10]:
        revs = res.reviews_for(a.rel)
        note = " (unreviewed)" if not revs else f" (review: {revs[-1].fm['verdict']})"
        note += " (merged)" if a.fm.get("merged") else ""
        done_lines.append(_line(a, note))
    section("Done (last 10)", done_lines)
    section("Failed", [_line(a) for a in res.plans if a.fm["status"] == "failed"])
    return "\n".join(out) + "\n"
```

- [ ] **Step 5: Run tests**

Run: `python3 -m unittest tools.harness.tests.test_scan_state -v`
Expected: 3 tests, `OK`

- [ ] **Step 6: Commit**

```bash
git add tools/harness/scan.py tools/harness/state.py tools/harness/tests/test_scan_state.py
git commit -m "feat(harness): artifact scanner and STATE.md renderer"
```

---

### Task 5: Templates

**Files:**
- Create: `.agents/templates/idea.md`, `.agents/templates/plan.md`, `.agents/templates/review.md`, `.agents/templates/run.md`

Templates use `{placeholder}` fields filled by the CLI. Frontmatter is emitted by the CLI, so templates hold bodies only.

- [ ] **Step 1: Write the four templates**

`.agents/templates/idea.md`:
```markdown
# {title}

## Why
<!-- Business rationale. Tie to spec goals: retention (30 min/day), CEFR progression, gamified pet engine. -->

## Expected output
<!-- What exists when done. User-visible behaviour first, then technical (tables, endpoints, packages). -->

## Evidence
<!-- Spec sections, research links, prior runs or reviews this idea rests on. -->
```

`.agents/templates/plan.md`:
```markdown
# {title} — Plan

**Idea:** `{idea}`
**Goal:** <!-- one sentence -->

## Tasks
<!-- writing-plans format: bite-sized tasks, each with files, steps, verification, commit. -->

### Task 1: …

## Verification
<!-- Commands that prove the whole plan is done. -->
```

`.agents/templates/review.md`:
```markdown
# Review — {title}

**Plan:** `{plan}`
**Branch/worktree:** `{branch}` / `{worktree}`
**Diff:** `git diff main...{branch} --stat`

## Plan vs idea
<!-- Does the plan actually deliver the idea's Expected output? -->

## Code vs plan
<!-- Each task: followed / deviated (justified?) / missing. -->

## Quality
<!-- Tests, boundaries (packages only talk via interfaces), CODEMAP accuracy. -->

## Bugs filed
<!-- One line per bug idea created in harness/ideas/_inbox/. -->

## Verdict
```

`.agents/templates/run.md`:
```markdown
# Ideation run {run}

**Mode:** {mode}
**Read:** <!-- spec sections, CODEMAP, inbox files, prior runs -->
**Inbox swept:** <!-- list bug files moved into this run -->

## Proposed
<!-- one line per idea file -->

## Notes
```

- [ ] **Step 2: Commit**

```bash
git add .agents/templates
git commit -m "feat(harness): artifact body templates"
```

---

### Task 6: CLI

**Files:**
- Create: `tools/harness/cli.py`
- Test: `tools/harness/tests/test_cli.py`

- [ ] **Step 1: Write the failing tests**

```python
# tools/harness/tests/test_cli.py
import os, io, sys, shutil, tempfile, unittest, pathlib, contextlib, time
from tools.harness import cli
from tools.harness.frontmatter import split_document, parse

def read_fm(path):
    fm, _ = split_document(pathlib.Path(path).read_text()); return parse(fm)

class CliTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.root = self.tmp.name
        for d in ["harness/ideas/_inbox", "harness/plans", "harness/reviews", "harness/runs", "harness/designs"]:
            pathlib.Path(self.root, d).mkdir(parents=True)
        src = pathlib.Path(__file__).resolve().parents[3] / ".agents" / "templates"
        shutil.copytree(src, pathlib.Path(self.root, ".agents", "templates"))
        self.old = os.getcwd(); os.chdir(self.root)
    def tearDown(self):
        os.chdir(self.old); self.tmp.cleanup()

    def run_cli(self, *args):
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            code = cli.main(list(args))
        return code, buf.getvalue().strip()

    def test_slug(self):
        self.assertEqual(self.run_cli("slug", "Spaced Repetition: Vocab!")[1], "spaced-repetition-vocab")

    def test_new_run_increments(self):
        _, r1 = self.run_cli("new-run")
        _, r2 = self.run_cli("new-run")
        self.assertTrue(r1.endswith("-run-01") and r2.endswith("-run-02"))
        self.assertTrue(pathlib.Path(r1, "_run.md").exists())

    def test_new_idea_and_validate(self):
        _, run = self.run_cli("new-run")
        code, path = self.run_cli("new-idea", "--run", run, "--title", "Alpha Beta", "--type", "feature", "--source", "human")
        self.assertEqual(code, 0)
        fm = read_fm(path)
        self.assertEqual(fm["status"], "proposed"); self.assertEqual(fm["source"], "human")
        self.assertEqual(fm["run"], pathlib.Path(run).name)
        self.assertEqual(self.run_cli("validate")[0], 0)

    def test_set_enforces_transition(self):
        _, run = self.run_cli("new-run")
        _, idea = self.run_cli("new-idea", "--run", run, "--title", "X", "--type", "feature", "--source", "ideator")
        self.assertEqual(self.run_cli("set", idea, "status=planned")[0], 1)          # proposed -> planned illegal
        self.assertEqual(self.run_cli("set", idea, "status=selected", "priority=high")[0], 0)
        self.assertEqual(read_fm(idea)["priority"], "high")

    def test_new_plan_links_idea_and_copies_priority(self):
        _, run = self.run_cli("new-run")
        _, idea = self.run_cli("new-idea", "--run", run, "--title", "Pet Unique", "--type", "bug", "--source", "reviewer")
        self.run_cli("set", idea, "status=selected", "priority=high")
        code, plan = self.run_cli("new-plan", "--idea", idea)
        self.assertEqual(code, 0)
        pf = read_fm(plan)
        self.assertEqual(pf["idea"], idea); self.assertEqual(pf["status"], "draft"); self.assertEqual(pf["priority"], "high")
        self.assertEqual(read_fm(idea)["status"], "planned"); self.assertEqual(read_fm(idea)["plan"], plan)

    def test_next_execute_ranks_priority_then_order(self):
        _, run = self.run_cli("new-run")
        plans = {}
        for title, typ, prio, order in [("Low", "feature", "low", None), ("HighTwo", "mvp-slice", "high", 2), ("HighOne", "mvp-slice", "high", 1)]:
            args = ["new-idea", "--run", run, "--title", title, "--type", typ, "--source", "ideator"] + (["--order", str(order)] if order else [])
            _, idea = self.run_cli(*args)
            self.run_cli("set", idea, "status=selected", f"priority={prio}")
            _, plan = self.run_cli("new-plan", "--idea", idea); plans[title] = plan
            self.run_cli("set", plan, "status=approved")
        self.assertEqual(self.run_cli("next", "--stage", "execute")[1], plans["HighOne"])

    def test_next_review_and_evaluate(self):
        _, run = self.run_cli("new-run")
        _, idea = self.run_cli("new-idea", "--run", run, "--title", "R", "--type", "feature", "--source", "ideator")
        self.assertEqual(self.run_cli("next", "--stage", "evaluate")[1], idea)
        self.run_cli("set", idea, "status=selected", "priority=medium")
        _, plan = self.run_cli("new-plan", "--idea", idea)
        for st in ["approved", "executing", "done"]: self.run_cli("set", plan, f"status={st}")
        self.assertEqual(self.run_cli("next", "--stage", "review")[1], plan)
        _, rev = self.run_cli("new-review", "--plan", plan, "--verdict", "pass")
        self.assertEqual(read_fm(rev)["plan"], plan)
        self.assertEqual(self.run_cli("next", "--stage", "review")[1], "")

    def test_lock_unlock_and_stale(self):
        self.assertEqual(self.run_cli("lock", "harness/plans/a.md")[0], 0)
        self.assertEqual(self.run_cli("lock", "harness/plans/b.md")[0], 1)
        old = time.time() - 3 * 3600
        os.utime("harness/.lock", (old, old))
        self.assertEqual(self.run_cli("lock", "harness/plans/b.md")[0], 0)   # stale lock taken over
        self.assertEqual(self.run_cli("unlock")[0], 0)
        self.assertFalse(pathlib.Path("harness/.lock").exists())

    def test_state_writes_file(self):
        self.run_cli("state")
        self.assertIn("# Harness state", pathlib.Path("harness/STATE.md").read_text())

if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run to verify failure**

Run: `python3 -m unittest tools.harness.tests.test_cli -v`
Expected: `ModuleNotFoundError` / `AttributeError: module has no attribute 'main'`

- [ ] **Step 3: Implement cli.py**

```python
# tools/harness/cli.py
"""Single entry point for every harness state mutation. Agents call this; they never hand-edit frontmatter."""
import argparse, datetime, os, pathlib, re, sys, time, unicodedata

if __package__ in (None, ""):
    sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2]))
from tools.harness.frontmatter import split_document, parse, join_document
from tools.harness.schema import kind_of, validate, TRANSITIONS
from tools.harness.scan import scan, load
from tools.harness.state import render_state, plan_sort_key

TEMPLATES = pathlib.Path(".agents/templates")
LOCK = pathlib.Path("harness/.lock")
LOCK_STALE_SECONDS = 2 * 3600


def today():
    return datetime.date.today().isoformat()


def slugify(text):
    text = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode()
    return re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")[:60]


def read(path):
    fm, body = split_document(pathlib.Path(path).read_text())
    return parse(fm), body


def write(path, fm, body):
    kind = kind_of(path)
    errs = validate(kind, fm) if kind else []
    if errs:
        print("\n".join(f"error: {e}" for e in errs)); return 1
    pathlib.Path(path).write_text(join_document(fm, body))
    return 0


def template(name, **kw):
    return (TEMPLATES / f"{name}.md").read_text().format(**kw)


def refresh_state():
    text = render_state(scan("."))
    pathlib.Path("harness/STATE.md").write_text(text)
    return text


# ---- commands -------------------------------------------------------------

def cmd_validate(a):
    res = scan(".")
    for art in res.invalid:
        print(f"{art.rel}: " + "; ".join(art.errors))
    return 1 if res.invalid else 0


def cmd_state(a):
    print(refresh_state()); return 0


def cmd_slug(a):
    print(slugify(a.title)); return 0


def cmd_new_run(a):
    base = pathlib.Path("harness/ideas")
    prefix = f"{today()}-run-"
    n = 1 + max([int(p.name[len(prefix):]) for p in base.glob(prefix + "*") if p.name[len(prefix):].isdigit()] or [0])
    run = base / f"{prefix}{n:02d}"
    run.mkdir(parents=True)
    (run / "_run.md").write_text(template("run", run=run.name, mode="mvp" if a.mvp else "features"))
    print(run.as_posix()); return 0


def cmd_new_idea(a):
    run = pathlib.Path(a.run)
    if not run.is_dir():
        print(f"error: run dir not found: {run}"); return 1
    slug = slugify(a.title)
    path = run / f"{slug}.md"
    if path.exists():
        print(f"error: exists: {path}"); return 1
    fm = {"type": a.type, "status": "proposed", "source": a.source, "run": run.name}
    if a.order is not None:
        fm["order"] = a.order
    if a.priority:
        fm["priority"] = a.priority
    code = write(path, fm, template("idea", title=a.title))
    if code == 0:
        print(path.as_posix()); refresh_state()
    return code


def cmd_new_plan(a):
    ifm, ibody = read(a.idea)
    if ifm.get("status") != "selected":
        print(f"error: idea must be 'selected' (is '{ifm.get('status')}')"); return 1
    title = next((l[2:].strip() for l in ibody.splitlines() if l.startswith("# ")), "untitled")
    slug = pathlib.Path(a.idea).stem
    path = pathlib.Path("harness/plans") / f"{today()}-{slug}.md"
    if path.exists():
        print(f"error: exists: {path}"); return 1
    fm = {"idea": a.idea, "status": "draft", "priority": ifm.get("priority") or "medium", "merged": False}
    if isinstance(ifm.get("order"), int):
        fm["order"] = ifm["order"]
    code = write(path, fm, template("plan", title=title, idea=a.idea))
    if code:
        return code
    ifm["status"] = "planned"; ifm["plan"] = path.as_posix()
    write(a.idea, ifm, ibody)
    print(path.as_posix()); refresh_state(); return 0


def cmd_new_review(a):
    pfm, pbody = read(a.plan)
    title = next((l[2:].strip() for l in pbody.splitlines() if l.startswith("# ")), "untitled").replace(" — Plan", "")
    slug = re.sub(r"^\d{4}-\d{2}-\d{2}-", "", pathlib.Path(a.plan).stem)
    path = pathlib.Path("harness/reviews") / f"{today()}-{slug}.md"
    fm = {"plan": a.plan, "verdict": a.verdict, "bugs": a.bugs or []}
    body = template("review", title=title, plan=a.plan, branch=pfm.get("branch") or "-", worktree=pfm.get("worktree") or "-")
    code = write(path, fm, body)
    if code == 0:
        print(path.as_posix()); refresh_state()
    return code


def _coerce(v):
    if v in ("true", "false"): return v == "true"
    if re.fullmatch(r"-?\d+", v): return int(v)
    if v.startswith("[") and v.endswith("]"): return [x.strip() for x in v[1:-1].split(",") if x.strip()]
    return v


def cmd_set(a):
    kind = kind_of(a.file)
    if not kind:
        print("error: not a harness artifact"); return 1
    fm, body = read(a.file)
    for kv in a.pairs:
        k, _, v = kv.partition("=")
        v = _coerce(v)
        if k == "status" and kind in TRANSITIONS:
            cur = fm.get("status")
            if v != cur and v not in TRANSITIONS[kind].get(cur, []):
                print(f"error: illegal transition {cur} -> {v} for {kind}"); return 1
        fm[k] = v
    code = write(a.file, fm, body)
    if code == 0:
        refresh_state()
    return code


def cmd_next(a):
    res = scan(".")
    if a.stage == "evaluate":
        items = [i.rel for i in res.ideas if i.fm["status"] == "proposed" and "/_inbox/" not in i.rel]
    elif a.stage == "execute":
        items = [p.rel for p in sorted(res.plans, key=plan_sort_key) if p.fm["status"] == "approved"]
    elif a.stage == "review":
        items = [p.rel for p in sorted(res.plans, key=lambda p: p.rel) if p.fm["status"] == "done" and not res.reviews_for(p.rel)]
    else:
        items = []
    if a.all:
        print("\n".join(items))
    else:
        print(items[0] if items else "")
    return 0


def cmd_lock(a):
    if LOCK.exists() and time.time() - LOCK.stat().st_mtime < LOCK_STALE_SECONDS:
        print(f"error: locked by {LOCK.read_text().strip()}"); return 1
    LOCK.write_text(a.plan + "\n"); print("locked"); return 0


def cmd_unlock(a):
    if LOCK.exists():
        LOCK.unlink()
    print("unlocked"); return 0


def cmd_stale_worktrees(a):
    res = scan(".")
    merged = {pathlib.Path(p.fm.get("worktree", "")).name for p in res.plans if p.fm.get("merged")}
    for wt in sorted(pathlib.Path(".worktrees").glob("*")) if pathlib.Path(".worktrees").exists() else []:
        if wt.name in merged:
            print(wt.as_posix())
    return 0


def main(argv=None):
    ap = argparse.ArgumentParser(prog="harness")
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("validate").set_defaults(fn=cmd_validate)
    sub.add_parser("state").set_defaults(fn=cmd_state)
    p = sub.add_parser("slug"); p.add_argument("title"); p.set_defaults(fn=cmd_slug)
    p = sub.add_parser("new-run"); p.add_argument("--mvp", action="store_true"); p.set_defaults(fn=cmd_new_run)
    p = sub.add_parser("new-idea")
    p.add_argument("--run", required=True); p.add_argument("--title", required=True)
    p.add_argument("--type", required=True, choices=["feature", "bug", "mvp-slice"])
    p.add_argument("--source", required=True, choices=["ideator", "human", "reviewer"])
    p.add_argument("--order", type=int); p.add_argument("--priority", choices=["high", "medium", "low"])
    p.set_defaults(fn=cmd_new_idea)
    p = sub.add_parser("new-plan"); p.add_argument("--idea", required=True); p.set_defaults(fn=cmd_new_plan)
    p = sub.add_parser("new-review"); p.add_argument("--plan", required=True)
    p.add_argument("--verdict", required=True, choices=["pass", "pass-with-bugs", "fail"]); p.add_argument("--bugs", nargs="*")
    p.set_defaults(fn=cmd_new_review)
    p = sub.add_parser("set"); p.add_argument("file"); p.add_argument("pairs", nargs="+"); p.set_defaults(fn=cmd_set)
    p = sub.add_parser("next"); p.add_argument("--stage", required=True, choices=["evaluate", "execute", "review"])
    p.add_argument("--all", action="store_true"); p.set_defaults(fn=cmd_next)
    p = sub.add_parser("lock"); p.add_argument("plan"); p.set_defaults(fn=cmd_lock)
    sub.add_parser("unlock").set_defaults(fn=cmd_unlock)
    sub.add_parser("stale-worktrees").set_defaults(fn=cmd_stale_worktrees)
    a = ap.parse_args(argv)
    return a.fn(a)


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 4: Run all tests**

Run: `python3 -m unittest discover -s tools/harness/tests -v`
Expected: all tests `OK` (7 + 8 + 3 + 9 = 27)

- [ ] **Step 5: Smoke-test from repo root and generate first STATE.md**

Run: `python3 tools/harness/cli.py state`
Expected: prints `# Harness state` with every section `_none_`; `harness/STATE.md` now exists.

- [ ] **Step 6: Commit**

```bash
git add tools/harness/cli.py tools/harness/tests/test_cli.py harness/STATE.md
git commit -m "feat(harness): cli for artifact creation, status transitions, next-work, locks"
```

---

### Task 7: CODEMAP.md seed

**Files:**
- Create: `harness/CODEMAP.md`

- [ ] **Step 1: Write the seed**

```markdown
# CODEMAP

One paragraph per package/module. Read this before exploring code. Executors update the paragraph for any package they change; reviewers correct it.

## Planned backend packages (`backend/internal/`) — none exist yet

- **store** — Postgres + Redis clients, migrations (DDL from spec §3.2). Everything else depends on it.
- **auth** — Google OAuth code exchange, refresh-token storage, JWT issue/verify. `POST /api/v1/auth/google`.
- **onboarding** — placement test (Redis `quiz:placement:*`), CEFR grading via airouter, roadmap kickoff. `POST /api/v1/onboarding/assessment`.
- **quests** — daily quests + progress. Redis `INCRBY daily:accumulated:*` first, then Postgres `daily_progress`. `GET /quests/daily`, `POST /quests/progress`.
- **pet** — health/streak/stage engine, decay on missed target, revive challenge. `GET /pet/status`, `POST /pet/revive`.
- **airouter** — LLM providers + task→provider strategies (spec §6.2). Rate limit `ratelimit:ai:*`.
- **google** — one-way Calendar + Tasks sync. `POST /integrations/google/sync`.
- **notify** — Web Push, `queue:webpush:delay` ZSET, in-process cron. `POST /settings/notifications`.

## Planned frontend areas (`frontend/`) — none exist yet

- **auth flow**, **onboarding wizard**, **daily quest screen**, **pet view**, **settings**, **PWA shell** (`@vite-pwa/nuxt`, service worker, push subscription).

## Harness tooling (`tools/harness/`)

- `frontmatter.py` YAML-subset parser · `schema.py` required keys/enums/transitions · `scan.py` walk + validate · `state.py` STATE.md renderer · `cli.py` all mutations. Tests: `python3 -m unittest discover -s tools/harness/tests`.
```

- [ ] **Step 2: Commit**

```bash
git add harness/CODEMAP.md
git commit -m "docs(harness): seed CODEMAP with planned packages"
```

---

### Task 8: Ideator role, skill, and commands

**Files:**
- Create: `.agents/roles/ideator.md`
- Create: `.agents/skills/harness-ideate/SKILL.md`
- Create: `.claude/agents/harness-ideator.md`
- Create: `.claude/commands/ideate.md`, `.claude/commands/idea.md`

- [ ] **Step 1: Write the role**

```markdown
# Role: Ideator

You are the product-minded researcher for the Adaptive English Learning Platform. You read the business intent (spec, CODEMAP, prior runs, reviewer-filed bugs) and propose concrete, well-argued ideas. You do not judge them — the evaluator does — but every idea you write must be worth judging.

## You must
- Ground every idea in the spec's goals: ≥30 min/day retention, CEFR progression, the pet/plant loop, Google Calendar/Tasks integration, PWA reach.
- Write `## Why` as a business argument a skeptical founder would accept, not a feature description.
- Write `## Expected output` so a reviewer could later check whether it was delivered.
- Cite evidence: spec section numbers, CODEMAP entries, research links, prior run or review files.
- Sweep `harness/ideas/_inbox/` into your run: move each bug file in, keeping its frontmatter, and list it in `_run.md`.
- Prefer fewer, sharper ideas over many vague ones.

## You must never
- Write plans, designs, or code.
- Edit frontmatter by hand — use the harness CLI.
- Set `priority` — that is the evaluator's call (except bugs from `_inbox/`, which keep what the reviewer set).
- Deduplicate against old runs. Re-proposing with fresh evidence is fine; history is append-only.

## MVP mode
When asked for MVP slices, propose exactly the packages in CODEMAP's "Planned backend packages" plus one `frontend-shell` slice, each as `type: mvp-slice` with `order:` in dependency order (store=1, auth=2, quests=3, pet=4, airouter=5, google=6, notify=7, frontend-shell=8). Each slice's Expected output lists the endpoints/tables/screens from the spec it must deliver.
```

- [ ] **Step 2: Write the skill**

```markdown
---
name: harness-ideate
description: Run one ideation cycle for the harness — create a run folder, sweep inbox bugs, research, write idea files and a run summary. Use when asked to ideate, find new features, or produce MVP slices.
---

# harness-ideate

Adopt the role in `.agents/roles/ideator.md`. Inputs: `mode` (`features` default, or `mvp`), `count` (default 5; ignored in mvp mode).

## Procedure

1. **Validate first.** Run `python3 tools/harness/cli.py validate`. If it exits 1, stop and report the invalid files — do not build on a broken state.
2. **Create the run.** `RUN=$(python3 tools/harness/cli.py new-run [--mvp])`.
3. **Sweep the inbox.** For every `harness/ideas/_inbox/*.md`: `git mv` it into `$RUN/`, then `python3 tools/harness/cli.py set $RUN/<file> run=<run-name>`. Record each in `_run.md` under *Inbox swept*.
4. **Read, in this order, and no more than needed:** `harness/CODEMAP.md`; the spec sections relevant to the mode (`project-base/1st-thinking-architecture-doc.md` §1, §5, §7 for features; §2–§4, §6, §7 for mvp); the last two `_run.md` files; if a `remembering-conversations` skill is available, query it for prior decisions about this project.
5. **Research (features mode only).** Look for 2–3 external references on retention mechanics in language-learning apps or on the specific gap you are targeting. Record URLs under `## Evidence`.
6. **Write ideas.** For each: `python3 tools/harness/cli.py new-idea --run $RUN --title "<Title>" --type <feature|bug|mvp-slice> --source ideator [--order N]`, then fill the three body sections of the created file (body only — leave frontmatter alone).
7. **Write `_run.md`.** Fill *Read*, *Inbox swept*, *Proposed* (one line per idea path + title), *Notes* (what you considered and dropped).
8. **Finish.** `python3 tools/harness/cli.py validate && python3 tools/harness/cli.py state`. Commit: `git add harness && git commit -m "harness: ideation run <run-name>"`.
9. **Report** the run path, the idea list with one-line summaries, and any inbox bugs swept. Stop — do not evaluate.
```

- [ ] **Step 3: Write the Claude agent adapter**

```markdown
---
name: harness-ideator
description: Harness ideation role. Spawn for /ideate and for the ideate stage of /harness run. Produces idea files in a new run folder; never plans or codes.
model: inherit
color: yellow
---

Load `.agents/roles/ideator.md` and adopt it fully. Then follow `.agents/skills/harness-ideate/SKILL.md` step by step with the mode and count you were given.

Tool mapping for this adapter: "load skill X" → use the Skill tool if X is registered, otherwise read `.agents/skills/X/SKILL.md`; "ask the user" → AskUserQuestion; prefer `rg` over reading whole files.
```

- [ ] **Step 4: Write the commands**

`.claude/commands/ideate.md`:
```markdown
---
description: Run one harness ideation cycle (new run folder, inbox sweep, idea files)
argument-hint: [--mvp] [--count N]
---

Spawn the `harness-ideator` agent with these arguments: `$ARGUMENTS`. Default mode is `features` with count 5; `--mvp` switches to MVP-slice mode. When it returns, print its report verbatim and then show `harness/STATE.md`.
```

`.claude/commands/idea.md`:
```markdown
---
description: Capture a human idea and evaluate it immediately
argument-hint: "<idea text>" [--type feature|bug]
---

1. Find or create today's manual run: `RUN=$(ls -d harness/ideas/$(date +%F)-run-* 2>/dev/null | tail -1)`; if empty, `RUN=$(python3 tools/harness/cli.py new-run)`.
2. Derive a short title (≤8 words) from `$ARGUMENTS`, then `python3 tools/harness/cli.py new-idea --run $RUN --title "<title>" --type <feature unless --type bug given> --source human`.
3. Fill the idea body from the user's text: their reasoning goes in `## Why`, anything they described as the result in `## Expected output`; write "human proposal, see conversation" in `## Evidence` if none given.
4. Spawn the `harness-evaluator` agent on that idea file and print its report.
```

- [ ] **Step 5: Commit**

```bash
git add .agents/roles/ideator.md .agents/skills/harness-ideate .claude/agents/harness-ideator.md .claude/commands/ideate.md .claude/commands/idea.md
git commit -m "feat(harness): ideator role, skill, and /ideate /idea commands"
```

---

### Task 9: Evaluator role, skill, and commands

**Files:**
- Create: `.agents/roles/evaluator.md`
- Create: `.agents/skills/harness-evaluate/SKILL.md`
- Create: `.claude/agents/harness-evaluator.md`
- Create: `.claude/commands/evaluate.md`, `.claude/commands/approve.md`

- [ ] **Step 1: Write the role**

```markdown
# Role: Evaluator

You are the technical lead who decides what gets built and writes the plan for it. You turn a proposed idea into either a rejection with a reason, or a `selected` idea with a priority and a `draft` plan (plus a design doc for UI work).

## Judging
Answer, in the idea file under a new `## Evaluation` section: Is the *Why* real for this product? Is the *Expected output* achievable in one plan (≤ ~1 day of agent work)? What does it depend on that doesn't exist yet? Then decide:
- **reject** — weak rationale, duplicates something done, or depends on unbuilt foundations that aren't themselves queued. Always give `rejected_reason`.
- **select** — set `priority`: `high` = blocks users or MVP order, or a `fail`-review bug; `medium` = clear retention/learning value; `low` = nice-to-have.

## Planning
For selected ideas write a plan in the `writing-plans` format: bite-sized tasks, exact paths, tests first, verification commands, commit per task. Respect the modular-monolith boundaries in CODEMAP — packages talk via interfaces, never each other's tables.
- **UI features:** first produce `harness/designs/<slug>.md` (layout, states, components, Tailwind tokens) using the frontend-design skill's guidance; the plan references it.
- **Bugs:** first find the root cause (systematic-debugging skill), then plan the smallest correct fix plus a regression test. Prioritise by user impact × frequency.

## You must never
- Write or edit app code.
- Hand-edit frontmatter — use the harness CLI.
- Approve your own plan. `draft → approved` is the human's (or orchestrator's) gate.
```

- [ ] **Step 2: Write the skill**

```markdown
---
name: harness-evaluate
description: Evaluate one idea or every proposed idea in a run — reject with reason or select with priority and write a draft plan (and design doc for UI). Use for /evaluate, /idea, and the evaluate stage of /harness run.
---

# harness-evaluate

Adopt `.agents/roles/evaluator.md`. Input: one idea path, or `--run <dir>` for all `proposed` ideas in it, or nothing (then use `python3 tools/harness/cli.py next --stage evaluate --all`).

## Per idea

1. `python3 tools/harness/cli.py validate` — stop on failure.
2. Read the idea file, `harness/CODEMAP.md`, and the spec sections it cites. Read nothing else unless the idea's evidence points there.
3. Apply the brainstorming skill's questioning to the *Why* — but answer the questions yourself from spec and evidence; do not ask the user unless the idea is a `source: human` idea and genuinely ambiguous.
4. Append `## Evaluation` to the idea body (verdict, reasoning, dependencies, priority rationale).
5. **Reject:** `python3 tools/harness/cli.py set <idea> status=rejected rejected_reason="<one sentence>"`. Done with this idea.
6. **Select:** `python3 tools/harness/cli.py set <idea> status=selected priority=<high|medium|low>`.
7. If the idea involves UI: write `harness/designs/<slug>.md` following the frontend-design skill. Keep it to layout, states, component list, and token choices — no code.
8. If `type: bug`: apply systematic-debugging to locate root cause in the worktree-free main checkout (read-only). Record it in `## Evaluation`.
9. `PLAN=$(python3 tools/harness/cli.py new-plan --idea <idea>)`. Fill the plan body using the writing-plans skill. If a design exists: `python3 tools/harness/cli.py set $PLAN design=harness/designs/<slug>.md`.
10. `python3 tools/harness/cli.py validate && python3 tools/harness/cli.py state`; commit `harness/` with message `harness: evaluate <slug>`.

## Report
List each idea → verdict (+ priority / plan path or rejected reason). Remind the user that drafts need `/approve <plan>`.
```

- [ ] **Step 3: Write the Claude agent adapter**

```markdown
---
name: harness-evaluator
description: Harness evaluation role. Spawn for /evaluate, /idea, and the evaluate stage of /harness run. Judges ideas, sets priority, writes draft plans and UI designs; never touches app code.
model: inherit
color: blue
---

Load `.agents/roles/evaluator.md` and adopt it fully. Follow `.agents/skills/harness-evaluate/SKILL.md` for the idea(s) you were given.

Tool mapping: "brainstorming / writing-plans / frontend-design / systematic-debugging skill" → invoke via the Skill tool (`brainstorming`, `writing-plans`, `frontend-design:frontend-design`, `systematic-debugging`); "ask the user" → AskUserQuestion, only for ambiguous human ideas.
```

- [ ] **Step 4: Write the commands**

`.claude/commands/evaluate.md`:
```markdown
---
description: Evaluate one idea, a whole run, or the next proposed idea
argument-hint: [<idea-file> | --run <run-dir>]
---

Spawn the `harness-evaluator` agent with: `$ARGUMENTS` (empty means "all proposed ideas"). Print its report, then `harness/STATE.md`.
```

`.claude/commands/approve.md`:
```markdown
---
description: Approve a draft plan for execution (human gate)
argument-hint: <plan-file>
---

Run `python3 tools/harness/cli.py set $ARGUMENTS status=approved`. If it fails, show the error. On success, `git add harness && git commit -m "harness: approve $(basename $ARGUMENTS .md)"` and show the Approved section of `harness/STATE.md`.
```

- [ ] **Step 5: Commit**

```bash
git add .agents/roles/evaluator.md .agents/skills/harness-evaluate .claude/agents/harness-evaluator.md .claude/commands/evaluate.md .claude/commands/approve.md
git commit -m "feat(harness): evaluator role, skill, /evaluate and /approve"
```

---

### Task 10: Executor role, skill, and command

**Files:**
- Create: `.agents/roles/executor.md`
- Create: `.agents/skills/harness-execute/SKILL.md`
- Create: `.claude/agents/harness-executor.md`
- Create: `.claude/commands/execute.md`

- [ ] **Step 1: Write the role**

```markdown
# Role: Executor

You implement one approved plan, exactly, in an isolated worktree, and leave a verifiable trail. You are a disciplined engineer, not a designer: the plan's intent is fixed.

## You must
- Work only inside `.worktrees/<slug>` on the plan's branch. Never edit files in the main checkout.
- Follow the plan task by task, tests first (test-driven-development), committing after each task.
- Verify before claiming done (verification-before-completion): run the plan's *Verification* commands and paste real output into the execution summary.
- Update `harness/CODEMAP.md` (in the worktree) for any package you create or change.
- Push the branch and open a **Draft** PR when done (see skill for naming).
- Log every deviation from the plan with a reason in `## Execution summary`.

## You must never
- Change what the plan is trying to achieve. If a task cannot be done as written, finish what you can, mark the plan `failed`, and explain in `## Failure`.
- Merge, push `main`, force-push, or delete branches/worktrees.
- Skip a failing test or weaken an assertion to get green.
- Hand-edit frontmatter — use the harness CLI (from the main checkout path, since `harness/` lives there).
```

- [ ] **Step 2: Write the skill**

```markdown
---
name: harness-execute
description: Execute one approved harness plan in a dedicated git worktree, push the branch, open a Draft PR, and record the outcome. Use for /execute and the execute stage of /harness run.
---

# harness-execute

Adopt `.agents/roles/executor.md`. Input: a plan path, or nothing (then `PLAN=$(python3 tools/harness/cli.py next --stage execute)`; if empty, report "nothing approved" and stop).

Let `ROOT` = main checkout (where you start). All `cli.py` calls run from `ROOT`.

## Procedure

1. `python3 tools/harness/cli.py validate` — stop on failure.
2. `python3 tools/harness/cli.py lock $PLAN` — if it fails, another executor is running; report and stop.
3. Read the plan, its idea, its design (if any), and `harness/CODEMAP.md`. Do not explore beyond what the plan names.
4. Derive names: `SLUG=$(basename $PLAN .md | sed 's/^[0-9-]*-//')`; `PRIO` = plan frontmatter `priority`; `DATE=$(date +%F)`; `BRANCH=harness/$DATE-$PRIO-$SLUG`; `WT=.worktrees/$SLUG`.
5. `git worktree add $WT -b $BRANCH main` (using-git-worktrees skill). Then `python3 tools/harness/cli.py set $PLAN status=executing branch=$BRANCH worktree=$WT`.
6. `cd $WT`. Execute the plan with the executing-plans skill: for each task — write the failing test, run it, implement, run, commit with the plan's message. Use `rg` to find code; read only matched ranges.
7. After the last task, run the plan's *Verification* section. Update `harness/CODEMAP.md` for touched packages and commit it.
8. **Record outcome** (back in `ROOT`):
   - Success: append `## Execution summary` to the plan (built / deviations + why / verification output), then `python3 tools/harness/cli.py set $PLAN status=done`.
   - Blocked: append `## Failure` (tried / blocker / suggested plan change), then `python3 tools/harness/cli.py set $PLAN status=failed`. Skip steps 9–10.
9. **Push + Draft PR** (skip with a note in the summary if `git remote get-url origin` fails or `gh auth status` fails):
   `cd $WT && git push -u origin $BRANCH`, then
   `gh pr create --draft --base main --head $BRANCH --title "[$DATE][P<n>] <Idea title>" --body-file <tmpfile> --label harness --label "type: <type>" --label "priority: $PRIO"`
   where `P1/P2/P3` = high/medium/low, and for `mvp-slice` use `[MVP-<order>]` instead of `[P<n>]`. Create missing labels with `gh label create`. Body = idea *Why* + *Expected output*, links to plan and idea paths, the execution summary, then the attribution line `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
   Then `python3 tools/harness/cli.py set $PLAN pr=<url>`.
10. `python3 tools/harness/cli.py unlock && python3 tools/harness/cli.py state`; in `ROOT`: `git add harness && git commit -m "harness: execute $SLUG ($STATUS)"`.
11. **Report:** status, branch, worktree, PR URL, verification result, deviations. Stop — do not review.
```

- [ ] **Step 3: Write the Claude agent adapter**

```markdown
---
name: harness-executor
description: Harness execution role. Spawn for /execute and the execute stage of /harness run. Implements one approved plan in a git worktree, pushes a harness/* branch, opens a Draft PR. Never merges.
model: inherit
color: green
---

Load `.agents/roles/executor.md` and adopt it fully. Follow `.agents/skills/harness-execute/SKILL.md` for the plan you were given (or the next approved one).

Tool mapping: executing-plans / test-driven-development / verification-before-completion / using-git-worktrees → Skill tool. Never use `--force`, `--no-verify`, or delete branches.
```

- [ ] **Step 4: Write the command**

`.claude/commands/execute.md`:
```markdown
---
description: Execute the next approved plan (or a given one) in a worktree and open a Draft PR
argument-hint: [<plan-file>]
---

Spawn the `harness-executor` agent with: `$ARGUMENTS`. Print its report, then `harness/STATE.md`.
```

- [ ] **Step 5: Commit**

```bash
git add .agents/roles/executor.md .agents/skills/harness-execute .claude/agents/harness-executor.md .claude/commands/execute.md
git commit -m "feat(harness): executor role, skill, and /execute"
```

---

### Task 11: Reviewer role, skill, and command

**Files:**
- Create: `.agents/roles/reviewer.md`
- Create: `.agents/skills/harness-review/SKILL.md`
- Create: `.claude/agents/harness-reviewer.md`
- Create: `.claude/commands/review.md`

- [ ] **Step 1: Write the role**

```markdown
# Role: Reviewer

You are the independent reviewer. You check two things: did the code deliver the plan, and did the plan deliver the idea. You report; you never fix.

## You must
- Work in the plan's worktree: build, run the tests, run the plan's Verification commands yourself. Never trust the execution summary without re-running.
- Diff `main...<branch>` and walk the plan task by task: followed / deviated (justified?) / missing.
- Check the idea's *Expected output* against what exists. A plan can be perfectly executed and still miss the idea.
- Enforce boundaries from CODEMAP: packages talk through interfaces; no cross-package table access; Redis-first ordering in the daily loop.
- Look for test gaps (the-validator style: boundaries, error paths, happy-path bias) and silent failures.
- File every bug as an idea in `harness/ideas/_inbox/` with `type: bug`, `source: reviewer`, a `priority`, and the plan path in *Evidence*.
- Correct `harness/CODEMAP.md` if the executor's update is wrong or missing (commit on the plan's branch).

## Verdicts
- `pass` — plan and idea delivered, no bugs filed.
- `pass-with-bugs` — delivered, but bugs filed (medium/low).
- `fail` — idea not delivered, tests failing, or a boundary violation. File at least one `priority: high` bug.

## You must never
- Edit app code or tests to "check something" and leave it changed.
- Merge or mark the PR ready on a `fail`.
```

- [ ] **Step 2: Write the skill**

```markdown
---
name: harness-review
description: Review a done harness plan — re-run verification in its worktree, compare code to plan and plan to idea, file bugs into the inbox, write a review file, comment on the PR. Use for /review and the review stage of /harness run.
---

# harness-review

Adopt `.agents/roles/reviewer.md`. Input: a plan path, or nothing (then `PLAN=$(python3 tools/harness/cli.py next --stage review)`; if empty, report "nothing to review" and stop).

## Procedure

1. `python3 tools/harness/cli.py validate` — stop on failure.
2. Read the plan (including its execution summary), its idea, its design if any, `harness/CODEMAP.md`. Note `branch`, `worktree`, `pr`.
3. `cd <worktree>`; `git diff main...<branch> --stat`; run the project's build/tests and the plan's *Verification* commands. Record real output.
4. Walk the diff against the plan tasks (code-review skill; typescript-review for Nuxt code). Walk the idea's Expected output against the result.
5. For each bug found: `RUN=harness/ideas/_inbox` — `python3 tools/harness/cli.py new-idea --run harness/ideas/_inbox --title "<bug title>" --type bug --source reviewer --priority <high|medium|low>`, then `python3 tools/harness/cli.py set <file> run=_inbox` and fill the body (*Why* = impact, *Expected output* = correct behaviour, *Evidence* = plan path, file:line, failing command).
6. Decide the verdict per the role. `REV=$(python3 tools/harness/cli.py new-review --plan $PLAN --verdict <v> --bugs <bug paths…>)`; fill the review body sections, paste verification output under *Code vs plan*.
7. If CODEMAP needs correction: edit it in the worktree and commit on the branch.
8. **PR** (skip with a note if the plan has no `pr`): `gh pr comment <pr> --body "<verdict + 3-line summary + path to review file>"`; on `pass` or `pass-with-bugs`: `gh pr ready <pr>`. On `fail` leave it Draft.
9. `python3 tools/harness/cli.py validate && python3 tools/harness/cli.py state`; commit `harness/` with `harness: review <slug> (<verdict>)`.
10. **Report:** verdict, bugs filed (paths), whether the PR is ready, and the merge command the human can run: `/harness merge $PLAN`.
```

- [ ] **Step 3: Write the Claude agent adapter**

```markdown
---
name: harness-reviewer
description: Harness review role. Spawn for /review and the review stage of /harness run. Re-verifies a done plan in its worktree, files bugs to the inbox, writes the review, comments on the PR. Never fixes code.
model: inherit
color: red
---

Load `.agents/roles/reviewer.md` and adopt it fully. Follow `.agents/skills/harness-review/SKILL.md` for the plan you were given (or the next unreviewed one).

Tool mapping: code-review → Skill `code-review`; typescript-review → Skill `typescript-review`; test-gap analysis → spawn the `the-validator` agent in read-only mode if the diff adds >200 lines of code.
```

- [ ] **Step 4: Write the command**

`.claude/commands/review.md`:
```markdown
---
description: Review the oldest unreviewed done plan (or a given one)
argument-hint: [<plan-file>]
---

Spawn the `harness-reviewer` agent with: `$ARGUMENTS`. Print its report, then `harness/STATE.md`.
```

- [ ] **Step 5: Commit**

```bash
git add .agents/roles/reviewer.md .agents/skills/harness-review .claude/agents/harness-reviewer.md .claude/commands/review.md
git commit -m "feat(harness): reviewer role, skill, and /review"
```

---

### Task 12: Orchestrator skill and /harness command

**Files:**
- Create: `.agents/skills/harness-orchestrate/SKILL.md`
- Create: `.claude/commands/harness.md`

- [ ] **Step 1: Write the skill**

```markdown
---
name: harness-orchestrate
description: Drive the harness end to end — status, one bounded autonomous run, human merge, worktree prune. Use for /harness and for scheduled unattended runs.
---

# harness-orchestrate

Subcommands: `status`, `run [--auto-approve] [--stages ideate,evaluate,execute,review]`, `merge <plan>`, `prune`.

## status
`python3 tools/harness/cli.py state`; then `python3 tools/harness/cli.py stale-worktrees` and append a "Stale worktrees" list. Print.

## run
Idempotent; safe to call on a schedule. `LOG=harness/runs/$(date +%Y%m%dT%H%M%S).log`. Every step appends one line to `$LOG`. Stages default to all four; `--stages` limits them.

1. `python3 tools/harness/cli.py validate` → if it fails, log the invalid files and **stop** (never build on broken state).
2. **ideate** if enabled and (`ls harness/ideas/_inbox/*.md` is non-empty **or** `next --stage evaluate` is empty): spawn the ideator role (features mode, count 5). Log the run path.
3. **evaluate** if enabled and `next --stage evaluate --all` is non-empty: spawn the evaluator role on all of them. Log verdicts.
4. **auto-approve** if `--auto-approve`: for each `draft` plan whose idea is `type: mvp-slice` **or** (`type: bug` and `priority: high`): `cli.py set <plan> status=approved`. Log each. Never auto-approve features.
5. **execute** if enabled and `next --stage execute` is non-empty: spawn the executor role on exactly that one plan. Log status, branch, PR.
6. **review** if enabled: for each path in `next --stage review --all`: spawn the reviewer role. Log verdicts and bugs filed.
7. `cli.py state`; append "Awaiting human: N draft plans, M passed reviews to merge" to `$LOG`; commit `harness/` with `harness: orchestrator run $(basename $LOG .log)`.
8. Print the log.

## merge <plan>  (human-invoked only)
Preconditions: plan `status=done`, latest review verdict `pass` or `pass-with-bugs`, `merged=false`. Refuse otherwise.
```
git checkout main && git merge --no-ff <branch> -m "Merge <branch>: <idea title>" && git push origin main
git worktree remove <worktree> && git branch -d <branch> && git push origin --delete <branch>
python3 tools/harness/cli.py set <plan> merged=true
```
If a `pr` exists, `gh pr merge <pr> --merge` may replace the local merge — pick one, never both. Commit `harness/`.

## prune
For each path from `cli.py stale-worktrees`: `git worktree remove <path>`. Print what was removed.
```

- [ ] **Step 2: Write the command**

`.claude/commands/harness.md`:
```markdown
---
description: Harness orchestrator — status | run [--auto-approve] [--stages …] | merge <plan> | prune
argument-hint: status | run [--auto-approve] | merge <plan-file> | prune
---

Load `.agents/skills/harness-orchestrate/SKILL.md` and perform the subcommand in `$ARGUMENTS`.

Role mapping for `run`: ideator → spawn agent `harness-ideator`; evaluator → `harness-evaluator`; executor → `harness-executor`; reviewer → `harness-reviewer`. Run stages sequentially — each depends on the previous one's files.

`merge` is human-only: if this command is being executed by a scheduler or another agent rather than the user, refuse and explain.
```

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/harness-orchestrate .claude/commands/harness.md
git commit -m "feat(harness): orchestrator skill and /harness command"
```

---

### Task 13: Acceptance tests 1–3 (manual, in a Claude Code session)

These exercise the real agents. Record outcomes in `harness/runs/acceptance.md`.

- [ ] **Step 1: Test 3 first (cheapest) — empty run**

Run: `/harness run`
Expected: log file created in `harness/runs/`; it says ideate ran (no proposed ideas existed) **or**, if you pass `--stages execute,review`, nothing runs and the log says "nothing to do"; `git status` shows only `harness/` changes.

- [ ] **Step 2: Test 1 — ideation against the spec**

Run: `/ideate --count 3`
Check: `harness/ideas/<today>-run-01/` has 3 idea files + `_run.md`; each `## Why` cites a spec goal; `python3 tools/harness/cli.py validate` exits 0; STATE.md lists them under Proposed.

- [ ] **Step 3: Test 2 — human idea end to end**

Run: `/idea "add UNIQUE(user_id) to pet_states in the DDL section of project-base/1st-thinking-architecture-doc.md"`
Expected: idea created with `source: human`; evaluator selects it (priority medium or high) and writes a draft plan.
Run: `/approve harness/plans/<today>-<slug>.md` → Approved.
Run: `/execute` → worktree `.worktrees/<slug>` exists; branch `harness/<today>-<prio>-<slug>`; plan `done` with execution summary; Draft PR opened if remote configured (title `[<today>][P2] …`).
Run: `/review` → review file with `verdict: pass`; PR marked ready.
Run: `/harness merge harness/plans/<today>-<slug>.md` → `main` contains the DDL change; worktree and branch gone; plan `merged: true`.

- [ ] **Step 4: Record and commit**

Write results (pass/fail per test, anything surprising) to `harness/runs/acceptance.md`.

```bash
git add harness
git commit -m "harness: acceptance test results"
```

- [ ] **Step 5: Kick off the real work**

Run: `/ideate --mvp`
Expected: 8 `mvp-slice` ideas with `order` 1–8. Then `/harness run --auto-approve --stages evaluate,execute` should plan them all and execute exactly `order: 1` (store).
