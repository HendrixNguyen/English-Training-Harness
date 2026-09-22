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
