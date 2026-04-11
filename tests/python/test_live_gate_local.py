import importlib.util
import json
import sys
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


live_gate = load_script_module("live_gate_under_test", "scripts/live_gate_local.py")


class LiveGateLocalTests(unittest.TestCase):
    def make_base_report(self):
        return {
            "build": {"status": "ok"},
            "localnet": {"status": "ok"},
            "sdk_surface": {"status": "warn", "warnings": ["distribution metadata drift: installed dist version=0.10.6, source pyproject=0.12.0"]},
            "pulse_compat": {
                "chain": {
                    "source": {"status": "chain-ready"},
                    "cli_live_tx": {"status": "ok"},
                    "sdk_live_tx": {"status": "ok"},
                },
                "thronglets": {"status": "thronglets-ready"},
                "sdk": {"status": "sdk-ready"},
                "sdk_surface": {"warnings": ["distribution metadata drift: installed dist version=0.10.6, source pyproject=0.12.0"]},
            },
            "autonomy": {"status": "ok"},
        }

    def test_extract_last_json_returns_final_payload(self):
        text = "\n".join(
            [
                "[live-gate] step one",
                json.dumps({"status": "ignore"}),
                "[live-gate] done",
                json.dumps({"status": "ok", "value": 1}),
            ]
        )
        payload = live_gate.extract_last_json(text)
        self.assertEqual(payload, {"status": "ok", "value": 1})

    def test_finalize_report_allows_only_distribution_metadata_drift(self):
        report = self.make_base_report()
        finalized = live_gate.finalize_report(report)
        self.assertEqual(finalized["status"], "ok")
        self.assertEqual(len(finalized["warnings"]), 1)

    def test_finalize_report_fails_on_non_preflight_warning(self):
        report = self.make_base_report()
        report["sdk_surface"] = {"status": "warn", "warnings": ["SDK has no explicit pulse_sigil helper on NativeSigner"]}
        finalized = live_gate.finalize_report(report)
        self.assertEqual(finalized["status"], "error")

    def test_finalize_report_fails_when_build_fails(self):
        report = self.make_base_report()
        report["build"]["status"] = "error"
        finalized = live_gate.finalize_report(report)
        self.assertEqual(finalized["status"], "error")

    def test_finalize_report_fails_when_localnet_fails(self):
        report = self.make_base_report()
        report["localnet"]["status"] = "error"
        finalized = live_gate.finalize_report(report)
        self.assertEqual(finalized["status"], "error")

    def test_finalize_report_fails_when_pulse_fails(self):
        report = self.make_base_report()
        report["pulse_compat"]["chain"]["sdk_live_tx"]["status"] = "error"
        finalized = live_gate.finalize_report(report)
        self.assertEqual(finalized["status"], "error")

    def test_finalize_report_fails_when_autonomy_fails(self):
        report = self.make_base_report()
        report["autonomy"]["status"] = "error"
        finalized = live_gate.finalize_report(report)
        self.assertEqual(finalized["status"], "error")


if __name__ == "__main__":
    unittest.main()
