# tools/harness/tests/test_scan_state.py
import os, tempfile, unittest, pathlib
from tools.harness.scan import scan
from tools.harness.state import render_state, render_context

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
            pass
        for h in ["## Invalid", "## Inbox (reviewer bugs awaiting evaluation)", "## Proposed", "## Approved", "## Done", "## Failed"]:
            self.assertIn(h, s)
        self.assertIn("broken.md", s.split("## Invalid")[1].split("##")[0])
        self.assertIn("bugone.md", s.split("## Inbox")[1].split("##")[0])
        self.assertIn("2026-09-22-alpha.md", s.split("## Approved")[1].split("##")[0])

    def test_done_without_review_flagged(self):
        s = render_state(scan(self.root))
        self.assertIn("beta.md", s)
        self.assertIn("(unreviewed)", s)
    def test_render_context_keeps_actionable_sections_and_collapses_history(self):
        codemap = "# CODEMAP\n\n## Backend\n\n- **store** — Postgres + Redis.\n- plain bullet\n"
        s = render_context(scan(self.root), codemap, "Branch: main")
        self.assertIn("## Git\nBranch: main", s)
        self.assertIn("## Backend\n  - store\n", s); self.assertNotIn("plain bullet", s)
        self.assertIn("bugone.md", s.split("## Inbox")[1].split("##")[0])
        self.assertIn("2026-09-22-alpha.md", s.split("## Approved")[1].split("##")[0])
        self.assertIn("## Proposed: 1 item(s)", s); self.assertNotIn("r1/alpha.md", s)
        self.assertIn("## Done (last 10): 1 item(s)", s); self.assertNotIn("beta.md", s)
        self.assertNotIn("_Generated", s)

    def test_render_context_without_git_or_codemap(self):
        s = render_context(scan(self.root))
        self.assertNotIn("## Git", s); self.assertNotIn("Code map", s)
        self.assertIn("## Harness state", s)

if __name__ == "__main__":
    unittest.main()
