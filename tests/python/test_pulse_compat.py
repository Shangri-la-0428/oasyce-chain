import importlib.util
import sys
import types
import unittest
from unittest import mock
from pathlib import Path


def load_script_module(name, relative_path):
    script_path = Path(__file__).resolve().parents[2] / relative_path
    spec = importlib.util.spec_from_file_location(name, script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


pulse_compat = load_script_module("pulse_compat_under_test", "scripts/check_pulse_compat.py")


class PulseCompatTests(unittest.TestCase):
    def test_sdk_pulse_state_requires_helper_schema_and_create_sigil(self):
        ready_surface = {
            "pulse": {
                "helper_names": ["pulse_sigil"],
                "schema_present": True,
                "schema_has_dimensions": True,
            },
            "signer_methods": {"create_sigil": True},
        }
        gap_surface = {
            "pulse": {
                "helper_names": [],
                "schema_present": True,
                "schema_has_dimensions": False,
            },
            "signer_methods": {"create_sigil": False},
        }

        ready = pulse_compat.sdk_pulse_state(ready_surface)
        gap = pulse_compat.sdk_pulse_state(gap_surface)

        self.assertEqual(ready["status"], "sdk-ready")
        self.assertEqual(gap["status"], "sdk-gap")

    def test_report_ok_requires_both_cli_and_sdk_live_pulse(self):
        report = {
            "chain": {
                "source": {"status": "chain-ready"},
                "cli_live_tx": {"status": "ok"},
                "sdk_live_tx": {"status": "ok"},
            },
            "thronglets": {"status": "thronglets-ready"},
            "sdk": {"status": "sdk-ready"},
            "sdk_surface": {"status": "warn"},
        }
        self.assertTrue(pulse_compat.report_ok(report))

        report["chain"]["sdk_live_tx"]["status"] = "error"
        self.assertFalse(pulse_compat.report_ok(report))

    def test_try_cli_live_pulse_reports_success(self):
        identity = {"address": "oasyce1validator", "pubkey_hex": "aa" * 32, "sigil_id": "SIG_cli"}
        with mock.patch.object(pulse_compat, "local_chain_state", return_value={"reachable": True, "rest_reachable": True}), \
            mock.patch.object(pulse_compat, "chain_binary_state", return_value={"has_pulse_cmd": True}), \
            mock.patch.object(pulse_compat, "validator_identity", return_value=identity), \
            mock.patch.object(pulse_compat, "ensure_local_sigil_cli"), \
            mock.patch.object(pulse_compat, "run_cmd", return_value=(0, '{"txhash":"ABC"}', "")), \
            mock.patch.object(
                pulse_compat,
                "wait_for_dimensions",
                return_value={"dimensions": {"chain": 1, "thronglets": 2}},
            ):
            result = pulse_compat.try_cli_live_pulse(
                Path("/tmp/oasyced"),
                "http://127.0.0.1:26657",
                "http://127.0.0.1:1317",
                "oasyce-live-gate-1",
                "test",
                "/tmp/home",
            )

        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["txhash"], "ABC")
        self.assertEqual(result["sigil_id"], "SIG_cli")

    def test_try_sdk_live_pulse_reports_success(self):
        wallet = types.SimpleNamespace(address="oasyce1sdk", public_key_bytes=bytes.fromhex("44" * 32))

        class FakeWallet:
            @staticmethod
            def from_private_key(_value):
                return wallet

        class FakeTxResult:
            success = True
            raw_log = ""
            code = 0
            tx_hash = "SDKTX"

        class FakeSigner:
            def __init__(self, wallet_obj, client, chain_id):
                self.wallet_obj = wallet_obj
                self.client = client
                self.chain_id = chain_id

            def create_sigil(self, _pubkey_hex):
                return FakeTxResult()

            def pulse_sigil(self, _sigil_id, dimensions):
                self.dimensions = dimensions
                return FakeTxResult()

        fake_sdk = types.ModuleType("oasyce_sdk")
        fake_sdk.OasyceClient = lambda rest: {"rest": rest}
        fake_crypto = types.ModuleType("oasyce_sdk.crypto")
        fake_crypto.NativeSigner = FakeSigner
        fake_crypto.Wallet = FakeWallet

        with mock.patch.dict(sys.modules, {"oasyce_sdk": fake_sdk, "oasyce_sdk.crypto": fake_crypto}, clear=False), \
            mock.patch.object(pulse_compat, "local_chain_state", return_value={"reachable": True, "rest_reachable": True}), \
            mock.patch.object(pulse_compat, "_ensure_sdk_importable"), \
            mock.patch.object(pulse_compat, "send_tokens"), \
            mock.patch.object(pulse_compat, "wait_for_balance"), \
            mock.patch.object(pulse_compat, "query_sigil", side_effect=RuntimeError("missing")), \
            mock.patch.object(
                pulse_compat,
                "wait_for_dimensions",
                return_value={"dimensions": {"sdk": 3, "chain": 4}},
            ):
            result = pulse_compat.try_sdk_live_pulse(
                Path("/tmp/oasyced"),
                "http://127.0.0.1:26657",
                "http://127.0.0.1:1317",
                "oasyce-live-gate-1",
                "test",
                "/tmp/home",
                "source",
            )

        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["txhash"], "SDKTX")
        self.assertEqual(result["address"], wallet.address)


if __name__ == "__main__":
    unittest.main()
