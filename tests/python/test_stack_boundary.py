import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path


def load_script_module(name, relative_path):
    script_path = Path(__file__).resolve().parents[2] / relative_path
    spec = importlib.util.spec_from_file_location(name, script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


stack_boundary = load_script_module("stack_boundary_under_test", "scripts/check_stack_boundary.py")


class StackBoundaryTests(unittest.TestCase):
    def test_scan_repo_flags_blacklisted_token_in_target_tree(self):
        with tempfile.TemporaryDirectory(prefix="stack-boundary-") as tmpdir:
            root = Path(tmpdir)
            (root / "app").mkdir()
            (root / "app" / "bad.go").write_text('const x = "presence_ping"\n', encoding="utf-8")

            findings = stack_boundary.scan_repo(root)

            self.assertEqual(len(findings), 1)
            self.assertEqual(findings[0].token, "presence_ping")

    def test_scan_repo_allows_explicit_inline_exemption(self):
        with tempfile.TemporaryDirectory(prefix="stack-boundary-allow-") as tmpdir:
            root = Path(tmpdir)
            (root / "x").mkdir()
            (root / "x" / "allowed.go").write_text(
                'const x = "presence_ping" // stack-boundary: allow fixture keyword\n',
                encoding="utf-8",
            )

            findings = stack_boundary.scan_repo(root)

            self.assertEqual(findings, [])

    def test_scan_repo_ignores_non_target_directories(self):
        with tempfile.TemporaryDirectory(prefix="stack-boundary-ignore-") as tmpdir:
            root = Path(tmpdir)
            (root / "scripts").mkdir()
            (root / "scripts" / "ignored.py").write_text("presence_ping = True\n", encoding="utf-8")

            findings = stack_boundary.scan_repo(root)

            self.assertEqual(findings, [])


if __name__ == "__main__":
    unittest.main()
