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

    def test_blocker_refuses_merge_until_fixed(self):
        _, run = self.run_cli("new-run")
        # the slice that was built and reviewed
        _, idea = self.run_cli("new-idea", "--run", run, "--title", "Slice", "--type", "mvp-slice", "--source", "ideator", "--order", "1")
        self.run_cli("set", idea, "status=selected", "priority=high")
        _, plan = self.run_cli("new-plan", "--idea", idea)
        for st in ["approved", "executing", "done"]:
            self.run_cli("set", plan, f"status={st}")
        # reviewer files a merge-blocking bug against it
        _, bug = self.run_cli("new-idea", "--run", run, "--title", "Drops tables", "--type", "bug", "--source", "reviewer", "--priority", "high")
        self.assertEqual(self.run_cli("set", bug, f"blocks={plan}")[0], 0)
        code, out = self.run_cli("blockers", "--plan", plan)
        self.assertEqual(code, 1)
        self.assertIn(bug, out)
        # merge is refused while the blocker is unfixed
        code, out = self.run_cli("set", plan, "merged=true")
        self.assertEqual(code, 1)
        self.assertIn("unresolved blockers", out)
        self.assertIs(read_fm(plan)["merged"], False)
        # fixing it: the blocker gets its own plan, executed to done
        self.run_cli("set", bug, "status=selected")
        _, fix = self.run_cli("new-plan", "--idea", bug)
        for st in ["approved", "executing", "done"]:
            self.run_cli("set", fix, f"status={st}")
        self.assertEqual(self.run_cli("blockers", "--plan", plan)[0], 0)
        self.assertEqual(self.run_cli("set", plan, "merged=true")[0], 0)
        self.assertIs(read_fm(plan)["merged"], True)

    def test_blocker_must_be_high_priority_bug(self):
        _, run = self.run_cli("new-run")
        _, feat = self.run_cli("new-idea", "--run", run, "--title", "Feat", "--type", "feature", "--source", "ideator", "--priority", "high")
        self.assertEqual(self.run_cli("set", feat, "blocks=harness/plans/x.md")[0], 1)
        _, low = self.run_cli("new-idea", "--run", run, "--title", "Low bug", "--type", "bug", "--source", "reviewer", "--priority", "low")
        self.assertEqual(self.run_cli("set", low, "blocks=harness/plans/x.md")[0], 1)

    def test_state_writes_file(self):
        self.run_cli("state")
        self.assertIn("# Harness state", pathlib.Path("harness/STATE.md").read_text())

if __name__ == "__main__":
    unittest.main()
