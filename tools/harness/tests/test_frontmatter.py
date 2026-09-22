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
