import importlib.util
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock


def load_script_module(name, relative_path):
    script_path = Path(__file__).resolve().parents[2] / relative_path
    spec = importlib.util.spec_from_file_location(name, script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


provider_agent = load_script_module("provider_agent_under_test", "scripts/provider_agent.py")


class ProviderAgentTests(unittest.TestCase):
    def setUp(self):
        self.tempdir = tempfile.TemporaryDirectory()
        provider_agent.CAPABILITY_ID = "CAP_TEST"
        provider_agent.ALERT_LOG = str(Path(self.tempdir.name) / "alert.log")
        provider_agent.ALERT_STATE_DIR = str(Path(self.tempdir.name) / "alerts")
        provider_agent._RUNTIME = None
        provider_agent._upstream_ok = True
        provider_agent._upstream_known = False
        provider_agent._upstream_error = ""
        provider_agent._upstream_check_ts = 0
        provider_agent._capability_ok = True
        provider_agent._capability_check_ts = 0
        provider_agent._capability_error = ""
        provider_agent._deactivated = False
        provider_agent._buyer_failure_streak = 0
        provider_agent.AUTO_DEACTIVATE_FAILURE_THRESHOLD = 3

    def tearDown(self):
        self.tempdir.cleanup()

    def test_health_without_probe_does_not_touch_upstream(self):
        with mock.patch.object(provider_agent, "_check_capability_cached", return_value=(True, "")), \
             mock.patch.object(provider_agent, "probe_upstream") as mock_probe:
            code, payload = provider_agent.build_health_status(probe=False)

        self.assertEqual(code, 200)
        self.assertEqual(payload["status"], "ok")
        self.assertIsNone(payload["upstream_ok"])
        mock_probe.assert_not_called()

    def test_health_probe_failure_reports_degraded_without_disabling_capability(self):
        with mock.patch.object(provider_agent, "_check_capability_cached", return_value=(True, "")), \
             mock.patch.object(provider_agent, "probe_upstream", return_value=(False, "No available accounts")), \
             mock.patch.object(provider_agent, "activate_alert_once") as mock_alert, \
             mock.patch.object(provider_agent, "submit_single") as mock_submit:
            code, payload = provider_agent.build_health_status(probe=True)

        self.assertEqual(code, 503)
        self.assertEqual(payload["status"], "degraded")
        self.assertFalse(provider_agent._deactivated)
        mock_alert.assert_not_called()
        mock_submit.assert_not_called()

    def test_buyer_path_failure_fails_invocation_and_deactivates(self):
        runtime = SimpleNamespace(actor_address="oasyce1provider")
        with mock.patch.object(provider_agent, "activate_alert_once") as mock_alert, \
             mock.patch.object(provider_agent, "get_runtime", return_value=runtime), \
             mock.patch.object(provider_agent, "submit_single", side_effect=["fail-tx", "deactivate-tx"]) as mock_submit, \
             mock.patch.object(provider_agent, "tx_status", side_effect=[(True, "failed"), (True, "deactivated")]):
            provider_agent.AUTO_DEACTIVATE_FAILURE_THRESHOLD = 1
            provider_agent.handle_buyer_path_failure("INV_TEST", "upstream HTTP 503")

        self.assertFalse(provider_agent._upstream_ok)
        self.assertTrue(provider_agent._upstream_known)
        self.assertEqual(provider_agent._upstream_error, "upstream HTTP 503")
        self.assertTrue(provider_agent._deactivated)
        self.assertEqual(mock_submit.call_count, 2)
        self.assertEqual(mock_submit.call_args_list[0].args[1], "/oasyce.capability.v1.MsgFailInvocation")
        self.assertEqual(
            mock_submit.call_args_list[0].args[2],
            {"creator": "oasyce1provider", "invocation_id": "INV_TEST"},
        )
        self.assertEqual(mock_submit.call_args_list[1].args[1], "/oasyce.capability.v1.MsgDeactivateCapability")
        self.assertEqual(
            mock_submit.call_args_list[1].args[2],
            {"creator": "oasyce1provider", "capability_id": "CAP_TEST"},
        )
        mock_alert.assert_called_once()

    def test_buyer_path_failure_below_threshold_only_fails_invocation(self):
        runtime = SimpleNamespace(actor_address="oasyce1provider")
        with mock.patch.object(provider_agent, "activate_alert_once") as mock_alert, \
             mock.patch.object(provider_agent, "get_runtime", return_value=runtime), \
             mock.patch.object(provider_agent, "submit_single", return_value="fail-tx") as mock_submit, \
             mock.patch.object(provider_agent, "tx_status", return_value=(True, "failed")):
            provider_agent.handle_buyer_path_failure("INV_TEST", "upstream HTTP 503")

        self.assertEqual(provider_agent._buyer_failure_streak, 1)
        self.assertFalse(provider_agent._deactivated)
        mock_submit.assert_called_once()
        self.assertEqual(mock_submit.call_args.args[1], "/oasyce.capability.v1.MsgFailInvocation")
        mock_alert.assert_not_called()

    def test_register_refuses_unreachable_upstream(self):
        with mock.patch.object(provider_agent, "probe_upstream", return_value=(False, "upstream down")), \
             mock.patch.object(provider_agent, "get_runtime") as mock_runtime:
            with self.assertRaises(SystemExit):
                provider_agent.register_capability("Test Capability", 1000)

        mock_runtime.assert_not_called()


if __name__ == "__main__":
    unittest.main()
