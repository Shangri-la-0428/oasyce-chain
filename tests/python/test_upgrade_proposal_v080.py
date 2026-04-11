import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock


def load_script_module(name, relative_path):
    script_path = Path(__file__).resolve().parents[2] / relative_path
    spec = importlib.util.spec_from_file_location(name, script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


proposal_tool = load_script_module("upgrade_proposal_v080_under_test", "scripts/upgrade_proposal_v080.py")


class UpgradeProposalV080Tests(unittest.TestCase):
    def test_render_artifacts_writes_expected_shape(self):
        with tempfile.TemporaryDirectory(prefix="proposal-v080-test-") as tmpdir:
            tmp = Path(tmpdir)
            proposal_path = tmp / "proposal.json"
            metadata_path = tmp / "metadata.json"

            result = proposal_tool.render_artifacts(
                height=123456,
                title="Upgrade Sigil",
                summary="Apply the v0.8.0 migration",
                deposit="123456789uoas",
                forum_url="https://forum.oasyce.com/t/v080",
                metadata_ref="",
                proposal_output=proposal_path,
                metadata_output=metadata_path,
            )

            self.assertEqual(result["status"], "ok")
            self.assertEqual(result["proposal"]["messages"][0]["@type"], proposal_tool.EXPECTED_MSG_TYPE)
            self.assertEqual(result["proposal"]["messages"][0]["plan"]["name"], proposal_tool.DEFAULT_PLAN_NAME)
            self.assertEqual(result["proposal"]["messages"][0]["plan"]["height"], 123456)
            self.assertEqual(result["proposal"]["deposit"], "123456789uoas")
            self.assertEqual(result["proposal"]["title"], "Upgrade Sigil")
            self.assertEqual(result["proposal"]["summary"], "Apply the v0.8.0 migration")
            self.assertEqual(result["proposal"]["metadata"], "https://forum.oasyce.com/t/v080")
            self.assertEqual(result["metadata"]["proposal_forum_url"], "https://forum.oasyce.com/t/v080")
            self.assertTrue(proposal_path.exists())
            self.assertTrue(metadata_path.exists())

    def test_validate_file_rejects_unrendered_height_placeholder(self):
        with tempfile.TemporaryDirectory(prefix="proposal-v080-invalid-") as tmpdir:
            proposal_path = Path(tmpdir) / "proposal.json"
            proposal_path.write_text(
                json.dumps(
                    {
                        "messages": [
                            {
                                "@type": proposal_tool.EXPECTED_MSG_TYPE,
                                "authority": proposal_tool.EXPECTED_AUTHORITY,
                                "plan": {
                                    "name": proposal_tool.DEFAULT_PLAN_NAME,
                                    "height": proposal_tool.HEIGHT_PLACEHOLDER,
                                    "info": "sigil v1 -> v2 effective activity height migration; state migration only; no new stores",
                                },
                            }
                        ],
                        "metadata": proposal_tool.METADATA_REF_PLACEHOLDER,
                        "deposit": proposal_tool.DEFAULT_DEPOSIT,
                        "title": "Upgrade",
                        "summary": "Summary",
                        "expedited": False,
                    },
                    indent=2,
                ),
                encoding="utf-8",
            )

            result = proposal_tool.validate_file(proposal_path)

            self.assertEqual(result["status"], "error")
            self.assertIn("messages[0].plan.height must be a positive integer", result["errors"])
            self.assertIn("metadata must be rendered", result["errors"])

    def test_build_submit_command_includes_dry_run_flags(self):
        cmd = proposal_tool.build_submit_command(
            Path("/tmp/proposal.json"),
            binary=Path("/tmp/oasyced"),
            from_name="validator",
            chain_id="oasyce-live-gate-1",
            fees="20000uoas",
            home="/tmp/home",
            keyring_backend="test",
            node="tcp://127.0.0.1:26657",
            dry_run=True,
        )

        self.assertEqual(cmd[:4], ["/tmp/oasyced", "tx", "gov", "submit-proposal"])
        self.assertIn("--dry-run", cmd)
        self.assertNotIn("--yes", cmd)
        self.assertIn("--home", cmd)
        self.assertIn("/tmp/home", cmd)
        self.assertIn("--node", cmd)
        self.assertIn("tcp://127.0.0.1:26657", cmd)

    def test_resolve_from_address_uses_keyring_lookup_for_names(self):
        completed = mock.Mock(returncode=0, stdout="oasyce1validatorqqqqqqqqqqqqqqqqqqqqqqqq6y3h2m\n", stderr="")

        with mock.patch.object(proposal_tool.live_gate, "run_cmd", return_value=completed) as run_cmd:
            address = proposal_tool.resolve_from_address(
                "validator",
                binary=Path("/tmp/oasyced"),
                home="/tmp/home",
                keyring_backend="test",
            )

        self.assertEqual(address, "oasyce1validatorqqqqqqqqqqqqqqqqqqqqqqqq6y3h2m")
        run_cmd.assert_called_once()

    def test_run_gov_dry_run_uses_resolved_address(self):
        proposal_path = Path("/tmp/proposal.json")

        with mock.patch.object(
            proposal_tool,
            "resolve_from_address",
            return_value="oasyce1validatorqqqqqqqqqqqqqqqqqqqqqqqq6y3h2m",
        ) as resolve_from_address, mock.patch.object(
            proposal_tool.live_gate,
            "run_cmd",
            return_value=mock.Mock(returncode=0, stdout='{"code":0}', stderr=""),
        ) as run_cmd:
            result = proposal_tool.run_gov_dry_run(
                proposal_path,
                binary=Path("/tmp/oasyced"),
                home="/tmp/home",
                chain_id="oasyce-live-gate-1",
                keyring_backend="test",
                rpc_url="http://127.0.0.1:26657",
                fees="20000uoas",
            )

        self.assertEqual(result["status"], "ok")
        resolve_from_address.assert_called_once()
        cmd = run_cmd.call_args.args[0]
        from_index = cmd.index("--from")
        self.assertEqual(cmd[from_index + 1], "oasyce1validatorqqqqqqqqqqqqqqqqqqqqqqqq6y3h2m")

    def test_dry_run_current_requires_home(self):
        args = proposal_tool.proposal_parser().parse_args(
            [
                "dry-run",
                "--height",
                "123456",
                "--network",
                "current",
            ]
        )

        result = proposal_tool.dry_run_command(args)

        self.assertEqual(result["status"], "error")
        self.assertEqual(result["error"], "--home is required for --network current")

    def test_dry_run_current_uses_submit_proposal_dry_run(self):
        with tempfile.TemporaryDirectory(prefix="proposal-v080-dry-run-") as tmpdir:
            proposal_path = Path(tmpdir) / "proposal.json"
            metadata_path = Path(tmpdir) / "metadata.json"
            args = proposal_tool.proposal_parser().parse_args(
                [
                    "dry-run",
                    "--height",
                    "123456",
                    "--network",
                    "current",
                    "--home",
                    "/tmp/home",
                    "--proposal-output",
                    str(proposal_path),
                    "--metadata-output",
                    str(metadata_path),
                ]
            )

            with mock.patch.object(
                proposal_tool,
                "run_gov_dry_run",
                return_value={"status": "ok", "command": "oasyced tx gov submit-proposal", "stdout": "{}", "stderr": "", "returncode": 0},
            ) as dry_run:
                result = proposal_tool.dry_run_command(args)

            self.assertEqual(result["status"], "ok")
            self.assertEqual(result["network"], "current")
            self.assertEqual(result["validation"]["status"], "ok")
            self.assertEqual(result["rendered"]["proposal"]["messages"][0]["plan"]["height"], 123456)
            dry_run.assert_called_once()

    def test_validate_file_rejects_too_long_metadata_reference(self):
        with tempfile.TemporaryDirectory(prefix="proposal-v080-invalid-metadata-") as tmpdir:
            proposal_path = Path(tmpdir) / "proposal.json"
            proposal_path.write_text(
                json.dumps(
                    {
                        "messages": [
                            {
                                "@type": proposal_tool.EXPECTED_MSG_TYPE,
                                "authority": proposal_tool.EXPECTED_AUTHORITY,
                                "plan": {
                                    "name": proposal_tool.DEFAULT_PLAN_NAME,
                                    "height": 123456,
                                    "info": "sigil v1 -> v2 effective activity height migration; state migration only; no new stores",
                                },
                            }
                        ],
                        "metadata": "x" * 256,
                        "deposit": proposal_tool.DEFAULT_DEPOSIT,
                        "title": "Upgrade",
                        "summary": "Summary",
                        "expedited": False,
                    },
                    indent=2,
                ),
                encoding="utf-8",
            )

            result = proposal_tool.validate_file(proposal_path)

            self.assertEqual(result["status"], "error")
            self.assertIn("metadata must be at most 255 bytes", result["errors"])


if __name__ == "__main__":
    unittest.main()
