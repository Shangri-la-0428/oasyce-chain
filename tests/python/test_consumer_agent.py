import importlib.util
import json
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


consumer_agent = load_script_module("consumer_agent_under_test", "scripts/consumer_agent.py")


class ConsumerAgentTests(unittest.TestCase):
    def setUp(self):
        self.tempdir = tempfile.TemporaryDirectory()
        consumer_agent.STATE_FILE = str(Path(self.tempdir.name) / "consumer_state.json")

    def tearDown(self):
        self.tempdir.cleanup()

    def load_state(self):
        with open(consumer_agent.STATE_FILE) as f:
            return json.load(f)

    def test_discover_capability_prefers_provider_capability(self):
        preferred = {
            "id": "CAP_PROVIDER",
            "name": "Claude AI",
            "is_active": True,
            "price_per_call": "500000",
        }
        with mock.patch.object(consumer_agent, "get_capability", return_value=preferred):
            cap = consumer_agent.discover_capability("CAP_PROVIDER")
        self.assertEqual(cap["id"], "CAP_PROVIDER")

    def test_main_records_heartbeat_when_balance_is_too_low(self):
        cap = {
            "id": "CAP_PROVIDER",
            "name": "Claude AI",
            "is_active": True,
            "price_per_call": "500000",
        }
        fetch_health = mock.Mock(return_value=({"status": "ok", "capability_id": "CAP_PROVIDER"}, ""))
        with mock.patch.object(consumer_agent, "get_address", return_value="oasyce1consumer"), \
             mock.patch.object(consumer_agent, "ensure_consumer_key", side_effect=lambda addr: addr), \
             mock.patch.object(consumer_agent, "check_balance", side_effect=[0, 0]), \
             mock.patch.object(consumer_agent, "request_faucet", return_value=(False, 0, "HTTP 429")), \
             mock.patch.object(consumer_agent, "fetch_provider_health", fetch_health), \
             mock.patch.object(consumer_agent, "discover_capability", return_value=cap):
            rc = consumer_agent.main()

        self.assertEqual(rc, 0)
        state = self.load_state()
        self.assertEqual(state["last_status"], "insufficient_funds")
        self.assertEqual(state["last_error"], "balance 0uoas below required invoke budget 510000uoas for CAP_PROVIDER")
        self.assertTrue(state["last_run"])
        self.assertEqual(state["last_success"], "")
        fetch_health.assert_called_once_with(probe=True)


if __name__ == "__main__":
    unittest.main()
