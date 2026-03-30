import importlib.util
import tempfile
import unittest
from pathlib import Path
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
        provider_agent._upstream_ok = True
        provider_agent._upstream_known = False
        provider_agent._upstream_error = ""
        provider_agent._upstream_check_ts = 0
        provider_agent._capability_ok = True
        provider_agent._capability_check_ts = 0
        provider_agent._capability_error = ""
        provider_agent._deactivated = False

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
             mock.patch.object(provider_agent, "oasyced_tx", return_value=(True, "txhash")) as mock_tx:
            code, payload = provider_agent.build_health_status(probe=True)

        self.assertEqual(code, 503)
        self.assertEqual(payload["status"], "degraded")
        self.assertFalse(provider_agent._deactivated)
        mock_alert.assert_not_called()
        mock_tx.assert_not_called()

    def test_buyer_path_failure_fails_invocation_and_deactivates(self):
        with mock.patch.object(provider_agent, "activate_alert_once") as mock_alert, \
             mock.patch.object(provider_agent, "oasyced_tx", side_effect=[(True, "failed"), (True, "deactivated")]) as mock_tx:
            provider_agent.handle_buyer_path_failure("INV_TEST", "upstream HTTP 503")

        self.assertFalse(provider_agent._upstream_ok)
        self.assertTrue(provider_agent._upstream_known)
        self.assertEqual(provider_agent._upstream_error, "upstream HTTP 503")
        self.assertTrue(provider_agent._deactivated)
        self.assertEqual(
            mock_tx.call_args_list,
            [
                mock.call(["oasyce_capability", "fail-invocation", "INV_TEST"]),
                mock.call(["oasyce_capability", "deactivate", "CAP_TEST"]),
            ],
        )
        mock_alert.assert_called_once()

    def test_register_refuses_unreachable_upstream(self):
        with mock.patch.object(provider_agent, "probe_upstream", return_value=(False, "upstream down")), \
             mock.patch.object(provider_agent, "oasyced_tx") as mock_tx:
            with self.assertRaises(SystemExit):
                provider_agent.register_capability("Test Capability", 1000)

        mock_tx.assert_not_called()


if __name__ == "__main__":
    unittest.main()
