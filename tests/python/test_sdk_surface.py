import importlib.util
import sys
import types
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock


def load_script_module(name, relative_path):
    script_path = Path(__file__).resolve().parents[2] / relative_path
    spec = importlib.util.spec_from_file_location(name, script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


sdk_compat = load_script_module("sdk_surface_under_test", "scripts/_sdk_compat.py")


def make_fake_signer(*, include_pulse=True):
    attrs = {}
    for name in (
        "register_capability",
        "invoke_capability",
        "complete_invocation",
        "claim_invocation",
        "register_asset",
        "buy_shares",
        "submit_feedback",
        "set_delegate_policy",
        "enroll_delegate",
        "create_sigil",
    ):
        attrs[name] = lambda self, *args, **kwargs: None
    if include_pulse:
        attrs["pulse_sigil"] = lambda self, *args, **kwargs: None
    return type("FakeSigner", (), attrs)


def fake_msg_schema_module(include_dimensions: bool):
    module = types.ModuleType("oasyce_sdk.crypto.msg_schemas")
    pulse_fields = [("signer", 1, "string"), ("sigil_id", 2, "string")]
    if include_dimensions:
        pulse_fields.append(("dimensions", 3, "map_string_uint64"))
    module.MSG_SCHEMAS = {"/oasyce.sigil.v1.MsgPulse": pulse_fields}
    return module


class SDKSurfaceTests(unittest.TestCase):
    def test_inspect_sdk_surface_warns_only_on_distribution_drift(self):
        fake_source = Path("/tmp/oasyce-sdk")
        fake_sdk_module = SimpleNamespace(
            __file__=str(fake_source / "oasyce_sdk" / "__init__.py"),
            __version__="0.12.0",
        )
        fake_contract = {
            "sdk_module": fake_sdk_module,
            "OasyceClient": object,
            "NativeSigner": make_fake_signer(),
            "seam": "public",
        }
        fake_scanner = SimpleNamespace(__file__="/tmp/oasyce-sdk/oasyce_sdk/agent/scanner.py", scan=lambda **_: [])
        fake_crypto = types.ModuleType("oasyce_sdk.crypto")
        fake_modules = {
            "oasyce_sdk.crypto": fake_crypto,
            "oasyce_sdk.crypto.msg_schemas": fake_msg_schema_module(include_dimensions=True),
        }

        with (
            mock.patch.object(sdk_compat, "_load_sdk_contract", return_value=fake_contract),
            mock.patch.object(sdk_compat, "_source_checkout_path", return_value=fake_source),
            mock.patch.object(sdk_compat, "_read_pyproject_version", return_value="0.12.0"),
            mock.patch.object(sdk_compat, "_safe_dist_version", return_value="0.10.6"),
            mock.patch.object(sdk_compat, "import_module", return_value=fake_scanner),
            mock.patch.dict(sys.modules, fake_modules, clear=False),
        ):
            surface = sdk_compat.inspect_sdk_surface("source")

        self.assertEqual(surface["status"], "warn")
        self.assertTrue(surface["resolved_from_source"])
        self.assertEqual(surface["identity_seam"], "public")
        self.assertEqual(surface["warnings"], ["distribution metadata drift: installed dist version=0.10.6, source pyproject=0.12.0"])
        self.assertEqual(surface["errors"], [])
        self.assertTrue(surface["pulse"]["schema_has_dimensions"])

    def test_inspect_sdk_surface_errors_when_pulse_surface_missing(self):
        fake_source = Path("/tmp/oasyce-sdk")
        fake_sdk_module = SimpleNamespace(
            __file__=str(fake_source / "oasyce_sdk" / "__init__.py"),
            __version__="0.12.0",
        )
        fake_contract = {
            "sdk_module": fake_sdk_module,
            "OasyceClient": object,
            "NativeSigner": make_fake_signer(include_pulse=False),
            "seam": "public",
        }
        fake_scanner = SimpleNamespace(__file__="/tmp/oasyce-sdk/oasyce_sdk/agent/scanner.py", scan=lambda **_: [])
        fake_crypto = types.ModuleType("oasyce_sdk.crypto")
        fake_modules = {
            "oasyce_sdk.crypto": fake_crypto,
            "oasyce_sdk.crypto.msg_schemas": fake_msg_schema_module(include_dimensions=False),
        }

        with (
            mock.patch.object(sdk_compat, "_load_sdk_contract", return_value=fake_contract),
            mock.patch.object(sdk_compat, "_source_checkout_path", return_value=fake_source),
            mock.patch.object(sdk_compat, "_read_pyproject_version", return_value="0.12.0"),
            mock.patch.object(sdk_compat, "_safe_dist_version", return_value="0.12.0"),
            mock.patch.object(sdk_compat, "import_module", return_value=fake_scanner),
            mock.patch.dict(sys.modules, fake_modules, clear=False),
        ):
            surface = sdk_compat.inspect_sdk_surface("source")

        self.assertEqual(surface["status"], "error")
        self.assertIn("pulse_sigil helper", "\n".join(surface["errors"]))
        self.assertIn("dimensions map field", "\n".join(surface["errors"]))

    def test_inspect_sdk_surface_errors_when_source_mode_resolves_outside_checkout(self):
        fake_source = Path("/tmp/oasyce-sdk")
        fake_sdk_module = SimpleNamespace(
            __file__="/tmp/site-packages/oasyce_sdk/__init__.py",
            __version__="0.12.0",
        )
        fake_contract = {
            "sdk_module": fake_sdk_module,
            "OasyceClient": object,
            "NativeSigner": make_fake_signer(),
            "seam": "public",
        }
        fake_scanner = SimpleNamespace(__file__="/tmp/site-packages/oasyce_sdk/agent/scanner.py", scan=lambda **_: [])
        fake_crypto = types.ModuleType("oasyce_sdk.crypto")
        fake_modules = {
            "oasyce_sdk.crypto": fake_crypto,
            "oasyce_sdk.crypto.msg_schemas": fake_msg_schema_module(include_dimensions=True),
        }

        with (
            mock.patch.object(sdk_compat, "_load_sdk_contract", return_value=fake_contract),
            mock.patch.object(sdk_compat, "_source_checkout_path", return_value=fake_source),
            mock.patch.object(sdk_compat, "_read_pyproject_version", return_value="0.12.0"),
            mock.patch.object(sdk_compat, "_safe_dist_version", return_value="0.12.0"),
            mock.patch.object(sdk_compat, "import_module", return_value=fake_scanner),
            mock.patch.dict(sys.modules, fake_modules, clear=False),
        ):
            surface = sdk_compat.inspect_sdk_surface("source")

        self.assertEqual(surface["status"], "error")
        self.assertIn("did not resolve from the adjacent checkout", "\n".join(surface["errors"]))

    def test_inspect_sdk_surface_errors_when_public_seam_falls_back_to_compat_bridge(self):
        fake_source = Path("/tmp/oasyce-sdk")
        fake_sdk_module = SimpleNamespace(
            __file__=str(fake_source / "oasyce_sdk" / "__init__.py"),
            __version__="0.12.0",
        )
        fake_contract = {
            "sdk_module": fake_sdk_module,
            "OasyceClient": object,
            "NativeSigner": make_fake_signer(),
            "seam": "compat_bridge",
            "public_seam_error": "missing resolve_identity",
        }
        fake_scanner = SimpleNamespace(__file__="/tmp/oasyce-sdk/oasyce_sdk/agent/scanner.py", scan=lambda **_: [])
        fake_crypto = types.ModuleType("oasyce_sdk.crypto")
        fake_modules = {
            "oasyce_sdk.crypto": fake_crypto,
            "oasyce_sdk.crypto.msg_schemas": fake_msg_schema_module(include_dimensions=True),
        }

        with (
            mock.patch.object(sdk_compat, "_load_sdk_contract", return_value=fake_contract),
            mock.patch.object(sdk_compat, "_source_checkout_path", return_value=fake_source),
            mock.patch.object(sdk_compat, "_read_pyproject_version", return_value="0.12.0"),
            mock.patch.object(sdk_compat, "_safe_dist_version", return_value="0.12.0"),
            mock.patch.object(sdk_compat, "import_module", return_value=fake_scanner),
            mock.patch.dict(sys.modules, fake_modules, clear=False),
        ):
            surface = sdk_compat.inspect_sdk_surface("source")

        self.assertEqual(surface["status"], "error")
        self.assertEqual(surface["identity_seam"], "compat_bridge")
        self.assertEqual(surface["public_seam_error"], "missing resolve_identity")
        self.assertIn("public identity seam unavailable", "\n".join(surface["errors"]))


if __name__ == "__main__":
    unittest.main()
