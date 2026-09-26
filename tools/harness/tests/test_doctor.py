# tools/harness/tests/test_doctor.py
import io, os, json, shutil, pathlib, tempfile, unittest, contextlib
from unittest import mock
from tools.harness import cli

REPO = pathlib.Path(__file__).resolve().parents[3]
COPY = [".agents/toolchain.json", ".agents/roles", ".agents/skills", ".claude/agents", ".claude/settings.json",
        ".mcp.json", ".codex/config.toml", ".codex/hooks.json", ".gemini/settings.json"]


class DoctorTests(unittest.TestCase):
    """doctor runs against a copy of the real repo's toolchain files, so the test also pins that they agree."""
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.old = os.getcwd()
        for rel in COPY:
            src, dst = REPO / rel, pathlib.Path(self.tmp.name, rel)
            dst.parent.mkdir(parents=True, exist_ok=True)
            (shutil.copytree(src, dst, symlinks=True) if src.is_dir() else shutil.copy(src, dst))
        os.chdir(self.tmp.name)
    def tearDown(self):
        os.chdir(self.old); self.tmp.cleanup()

    def run_doctor(self, present=None):
        present = present if present is not None else {"git", "gh", "python3", "go", "node", "npm", "docker"}
        buf = io.StringIO()
        with mock.patch.object(cli.shutil, "which", side_effect=lambda c: "/bin/" + c if c in present else None), \
             contextlib.redirect_stdout(buf):
            code = cli.main(["doctor"])
        return code, buf.getvalue()

    def test_repo_toolchain_passes_with_required_clis(self):
        code, out = self.run_doctor()
        self.assertEqual(code, 0, out)
        self.assertNotIn("FAIL", out)
        self.assertIn("WARN cli railway", out)          # optional CLI missing is only a warning

    def test_missing_required_cli_fails(self):
        code, out = self.run_doctor(present={"git", "python3"})
        self.assertEqual(code, 1)
        self.assertIn("FAIL cli gh", out)

    def test_required_mcp_missing_from_an_adapter_fails(self):
        g = json.loads(pathlib.Path(".gemini/settings.json").read_text()); del g["mcpServers"]["playwright"]
        pathlib.Path(".gemini/settings.json").write_text(json.dumps(g))
        code, out = self.run_doctor()
        self.assertEqual(code, 1)
        self.assertIn("FAIL mcp playwright: not declared for gemini-cli", out)

    def test_required_skill_pack_not_enabled_for_claude_fails(self):
        s = json.loads(pathlib.Path(".claude/settings.json").read_text()); s["enabledPlugins"].pop("superpowers@superpowers-marketplace")
        pathlib.Path(".claude/settings.json").write_text(json.dumps(s))
        code, out = self.run_doctor()
        self.assertEqual(code, 1)
        self.assertIn("FAIL skill-pack superpowers", out)

    def test_role_without_claude_adapter_fails(self):
        pathlib.Path(".claude/agents/harness-designer.md").unlink()
        code, out = self.run_doctor()
        self.assertEqual(code, 1)
        self.assertIn("FAIL role designer", out)


if __name__ == "__main__":
    unittest.main()
