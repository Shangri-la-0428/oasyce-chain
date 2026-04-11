import importlib.util
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
    spec.loader.exec_module(module)
    return module


data_agent = load_script_module("data_agent_under_test", "scripts/data_agent.py")


class DataAgentTests(unittest.TestCase):
    def setUp(self):
        self.tempdir = tempfile.TemporaryDirectory()
        self.watch_dir = Path(self.tempdir.name) / "watch"
        self.watch_dir.mkdir(parents=True, exist_ok=True)
        self.sample_path = self.watch_dir / "safe.txt"
        self.sample_path.write_text("safe fixture\n", encoding="utf-8")
        data_agent.STATE_FILE = str(Path(self.tempdir.name) / "state.json")
        data_agent.WATCH_DIRS = []
        data_agent._RUNTIME = None
        data_agent._registered_assets = {}
        data_agent._last_cycle_stats = {}
        data_agent._last_cycle_time = ""
        data_agent._total_registered = 0
        data_agent._total_scanned = 0
        data_agent._total_cycles = 0

    def tearDown(self):
        self.tempdir.cleanup()

    def test_main_dry_run_does_not_require_signer(self):
        argv = [
            "data_agent.py",
            "--once",
            "--dry-run",
            "--watch-dirs",
            str(self.watch_dir),
        ]
        with mock.patch.object(data_agent, "get_sdk_agent_scanner", return_value=object()), \
             mock.patch.object(data_agent, "load_state"), \
             mock.patch.object(data_agent, "scan_and_register_cycle", return_value={"errors": 0}) as mock_cycle, \
             mock.patch.object(data_agent, "get_agent_address") as mock_get_address, \
             mock.patch.object(sys, "argv", argv):
            with self.assertRaises(SystemExit) as ctx:
                data_agent.main()

        self.assertEqual(ctx.exception.code, 0)
        mock_get_address.assert_not_called()
        mock_cycle.assert_called_once_with(dry_run=True)

    def test_scan_and_register_cycle_live_registers_and_tracks_asset(self):
        data_agent.WATCH_DIRS = [str(self.watch_dir)]
        file_info = {
            "path": str(self.sample_path),
            "hash": "abc123",
            "category": "text",
            "ext": ".txt",
            "privacy_risk": "safe",
        }
        with mock.patch.object(data_agent, "scan_and_filter", return_value=[file_info]), \
             mock.patch.object(data_agent, "is_already_registered", return_value=False), \
             mock.patch.object(data_agent, "register_on_chain", return_value=(True, "DATA_ABC123")) as mock_register, \
             mock.patch.object(data_agent, "save_state") as mock_save:
            stats = data_agent.scan_and_register_cycle(dry_run=False)

        self.assertEqual(stats["registered"], 1)
        self.assertEqual(stats["errors"], 0)
        mock_register.assert_called_once_with("safe", "abc123", "text,txt", "")
        self.assertEqual(
            data_agent._registered_assets["abc123"],
            {"path": str(self.sample_path), "asset_id": "DATA_ABC123"},
        )
        mock_save.assert_called_once()


if __name__ == "__main__":
    unittest.main()
