import importlib.util
import sys
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


sdk_compat = load_script_module("sdk_compat_under_test", "scripts/_sdk_compat.py")


class SDKCompatTests(unittest.TestCase):
    def test_parse_uoas_accepts_numeric_and_oas_strings(self):
        self.assertEqual(sdk_compat.parse_uoas(7), 7)
        self.assertEqual(sdk_compat.parse_uoas(7.9), 7)
        self.assertEqual(sdk_compat.parse_uoas("123"), 123)
        self.assertEqual(sdk_compat.parse_uoas("42uoas"), 42)
        self.assertEqual(sdk_compat.parse_uoas("0.5 OAS"), 500000)
        self.assertEqual(sdk_compat.parse_uoas("bad"), 0)

    def test_split_csv_trims_and_drops_empty_values(self):
        self.assertEqual(sdk_compat.split_csv("alpha, beta ,, gamma "), ["alpha", "beta", "gamma"])

    def test_resolve_runtime_missing_identity_uses_sdk_hint(self):
        fake_contract = {
            "OasyceClient": lambda chain_rest: SimpleNamespace(chain_rest=chain_rest),
            "NativeSigner": object,
            "resolve_identity": lambda **_: SimpleNamespace(is_present=False, address=""),
            "seam": "public",
        }
        with mock.patch.object(sdk_compat, "_load_sdk_contract", return_value=fake_contract):
            with self.assertRaises(RuntimeError) as ctx:
                sdk_compat.resolve_runtime("http://127.0.0.1:1317", "oasyce-local-1", mode="source")

        self.assertIn("oasyce start", str(ctx.exception))
        self.assertIn("oasyce join", str(ctx.exception))

    def test_resolve_runtime_uses_public_identity_context(self):
        fake_wallet = SimpleNamespace(address="oasyce1delegate")
        fake_identity = SimpleNamespace(
            address="oasyce1delegate",
            is_present=True,
            context=SimpleNamespace(wallet=fake_wallet, principal="oasyce1principal"),
        )

        class FakeClient:
            def __init__(self, chain_rest):
                self.chain_rest = chain_rest

        class FakeSigner:
            def __init__(self, wallet, client, chain_id=None, principal=None):
                self.wallet = wallet
                self.client = client
                self.chain_id = chain_id
                self.principal = principal

        fake_contract = {
            "OasyceClient": FakeClient,
            "NativeSigner": FakeSigner,
            "resolve_identity": lambda **_: fake_identity,
            "seam": "public",
        }

        with mock.patch.object(sdk_compat, "_load_sdk_contract", return_value=fake_contract):
            runtime = sdk_compat.resolve_runtime("http://127.0.0.1:1317", "oasyce-local-1", mode="source")

        self.assertEqual(runtime.identity_seam, "public")
        self.assertEqual(runtime.actor_address, "oasyce1principal")
        self.assertEqual(runtime.signer_address, "oasyce1delegate")
        self.assertTrue(runtime.is_delegated)
        signer = runtime.signer
        self.assertEqual(signer.principal, "oasyce1principal")
        self.assertEqual(signer.wallet.address, "oasyce1delegate")

    def test_submit_single_wraps_delegated_identity_in_msg_exec(self):
        signer = SimpleNamespace(sign_and_broadcast=mock.Mock(return_value="ok"))
        runtime = SimpleNamespace(
            signer=signer,
            signer_address="oasyce1delegate",
            is_delegated=True,
        )

        result = sdk_compat.submit_single(
            runtime,
            "/oasyce.capability.v1.MsgFailInvocation",
            {"creator": "oasyce1principal", "invocation_id": "INV_123"},
        )

        self.assertEqual(result, "ok")
        signer.sign_and_broadcast.assert_called_once_with(
            [
                (
                    "/oasyce.delegate.v1.MsgExec",
                    {
                        "delegate": "oasyce1delegate",
                        "msgs": [
                            {
                                "@type": "/oasyce.capability.v1.MsgFailInvocation",
                                "creator": "oasyce1principal",
                                "invocation_id": "INV_123",
                            }
                        ],
                    },
                )
            ]
        )

    def test_submit_single_plain_for_non_delegate(self):
        signer = SimpleNamespace(sign_and_broadcast=mock.Mock(return_value="ok"))
        runtime = SimpleNamespace(signer=signer, is_delegated=False)

        result = sdk_compat.submit_single(
            runtime,
            "/oasyce.reputation.v1.MsgSubmitFeedback",
            {"creator": "oasyce1plain"},
        )

        self.assertEqual(result, "ok")
        signer.sign_and_broadcast.assert_called_once_with(
            [("/oasyce.reputation.v1.MsgSubmitFeedback", {"creator": "oasyce1plain"})]
        )


if __name__ == "__main__":
    unittest.main()
