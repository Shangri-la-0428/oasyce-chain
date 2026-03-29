import importlib.util
import json
import subprocess
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


faucet_server = load_script_module("faucet_server_under_test", "scripts/faucet_server.py")


class FaucetServerTests(unittest.TestCase):
    def test_run_faucet_send_retries_sequence_mismatch(self):
        first = subprocess.CompletedProcess(
            args=[],
            returncode=0,
            stdout=json.dumps({"code": 19, "raw_log": "sequence mismatch"}),
            stderr="",
        )
        second = subprocess.CompletedProcess(
            args=[],
            returncode=0,
            stdout=json.dumps({"code": 0, "txhash": "ABC123"}),
            stderr="",
        )
        with mock.patch.object(faucet_server.subprocess, "check_output", return_value="oasyce1faucet"), \
             mock.patch.object(faucet_server.subprocess, "run", side_effect=[first, second]), \
             mock.patch.object(faucet_server.time, "sleep", return_value=None):
            ok, payload = faucet_server.run_faucet_send("oasyce1target", retries=2)

        self.assertTrue(ok)
        self.assertEqual(payload["txhash"], "ABC123")

    def test_run_faucet_send_rejects_checktx_error(self):
        failed = subprocess.CompletedProcess(
            args=[],
            returncode=0,
            stdout=json.dumps({"code": 5, "raw_log": "insufficient funds"}),
            stderr="",
        )
        with mock.patch.object(faucet_server.subprocess, "check_output", return_value="oasyce1faucet"), \
             mock.patch.object(faucet_server.subprocess, "run", return_value=failed):
            ok, payload = faucet_server.run_faucet_send("oasyce1target", retries=1)

        self.assertFalse(ok)
        self.assertEqual(payload, "insufficient funds")


if __name__ == "__main__":
    unittest.main()
