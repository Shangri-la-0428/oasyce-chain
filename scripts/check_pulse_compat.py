#!/usr/bin/env python3
"""Check chain-side Pulse readiness against adjacent SDK and Thronglets sources."""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.request import ProxyHandler, Request, build_opener

from _sdk_compat import _ensure_sdk_importable, inspect_sdk_surface


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_OASYCED = ROOT / "build" / "oasyced"
DEFAULT_SDK_PATH = ROOT.parent / "oasyce-sdk"
DEFAULT_THRONGLETS_PATH = ROOT.parent / "Thronglets"
DEFAULT_RPC = "http://127.0.0.1:26657"
DEFAULT_REST = "http://127.0.0.1:1317"
DEFAULT_CHAIN_ID = "oasyce-local-1"
DEFAULT_KEYRING = "test"
SDK_PULSE_PRIVATE_KEY = "44" * 32
HTTP = build_opener(ProxyHandler({}))


def run_cmd(args: list[str], *, timeout: int = 60) -> tuple[int, str, str]:
    result = subprocess.run(args, capture_output=True, text=True, timeout=timeout)
    return result.returncode, result.stdout, result.stderr


def http_get_json(url: str) -> dict:
    with HTTP.open(Request(url), timeout=10) as resp:
        return json.loads(resp.read().decode())


def rest_get_json(rest_url: str, path: str) -> dict:
    return http_get_json(f"{rest_url.rstrip('/')}{path}")


def rpc_to_node(rpc: str) -> str:
    if rpc.startswith("http://"):
        return "tcp://" + rpc[len("http://") :]
    if rpc.startswith("https://"):
        return "tcp://" + rpc[len("https://") :]
    return rpc


def file_contains(path: Path, needle: str) -> bool:
    return path.exists() and needle in path.read_text(encoding="utf-8")


def derive_sigil_id(pubkey_hex: str) -> str:
    digest = hashlib.sha256(bytes.fromhex(pubkey_hex)).hexdigest()[:32]
    return f"SIG_{digest}"


def wait_for(predicate, timeout: float, *, interval: float = 1.0):
    deadline = time.time() + timeout
    last_error = None
    while time.time() < deadline:
        try:
            value = predicate()
        except Exception as exc:  # noqa: BLE001
            last_error = exc
            time.sleep(interval)
            continue
        if value:
            return value
        time.sleep(interval)
    if last_error is not None:
        raise RuntimeError(str(last_error))
    raise RuntimeError("timed out waiting for condition")


def ensure_local_no_proxy() -> None:
    local_hosts = "127.0.0.1,localhost"
    for key in ("NO_PROXY", "no_proxy"):
        current = os.environ.get(key, "").strip()
        if current:
            if "127.0.0.1" not in current and "localhost" not in current:
                os.environ[key] = current + "," + local_hosts
        else:
            os.environ[key] = local_hosts


def chain_source_state() -> dict:
    cli_path = ROOT / "x" / "sigil" / "cli" / "tx.go"
    msg_server_path = ROOT / "x" / "sigil" / "keeper" / "msg_server.go"
    msgs_path = ROOT / "x" / "sigil" / "types" / "msgs.go"
    source_checks = {
        "cli_cmd_pulse": file_contains(cli_path, 'Use:   "pulse [sigil-id]"'),
        "msg_server_pulse": file_contains(msg_server_path, "func (m msgServer) Pulse("),
        "msg_type_pulse": file_contains(msgs_path, "func (msg *MsgPulse) ValidateBasic() error"),
    }
    ready = all(source_checks.values())
    return {
        "status": "chain-ready" if ready else "chain-gap",
        "checks": source_checks,
    }


def chain_binary_state(binary: Path) -> dict:
    if not binary.exists():
        return {"status": "missing", "has_pulse_cmd": False, "detail": f"binary not found: {binary}"}

    _, stdout, stderr = run_cmd([str(binary), "tx", "sigil", "--help"])
    text = (stdout or "") + "\n" + (stderr or "")
    has_pulse_cmd = "pulse" in text
    return {
        "status": "ready" if has_pulse_cmd else "stale-build",
        "has_pulse_cmd": has_pulse_cmd,
        "detail": text.strip(),
    }


def thronglets_state(thronglets_path: Path) -> dict:
    pulse_rs = thronglets_path / "src" / "pulse.rs"
    main_rs = thronglets_path / "src" / "main.rs"
    checks = {
        "pulse_module": pulse_rs.exists(),
        "pulse_loop": file_contains(pulse_rs, "pub fn pulse_loop("),
        "pulse_emitter": file_contains(main_rs, "PulseEmitter"),
    }
    ready = all(checks.values())
    return {
        "status": "thronglets-ready" if ready else "thronglets-gap",
        "checks": checks,
        "path": str(thronglets_path),
    }


def sdk_pulse_state(surface: dict) -> dict:
    pulse = surface["pulse"]
    helper_present = "pulse_sigil" in pulse["helper_names"]
    dimensions_ready = bool(pulse["schema_present"] and pulse["schema_has_dimensions"])
    create_sigil_ready = bool(surface["signer_methods"].get("create_sigil", False))
    status = "sdk-ready" if helper_present and dimensions_ready and create_sigil_ready else "sdk-gap"
    return {
        "status": status,
        "helper_present": helper_present,
        "dimensions_ready": dimensions_ready,
        "create_sigil_ready": create_sigil_ready,
        "detail": pulse,
    }


def local_chain_state(rpc_url: str, rest_url: str) -> dict:
    state = {
        "reachable": False,
        "rest_reachable": False,
        "detail": "",
        "rest_detail": "",
    }
    try:
        status = http_get_json(f"{rpc_url.rstrip('/')}/status")
    except (HTTPError, URLError, TimeoutError, ValueError) as exc:
        state["detail"] = str(exc)
    else:
        state["reachable"] = True
        state["latest_height"] = status["result"]["sync_info"]["latest_block_height"]

    try:
        rest_get_json(rest_url, "/cosmos/base/tendermint/v1beta1/node_info")
    except (HTTPError, URLError, TimeoutError, ValueError) as exc:
        state["rest_detail"] = str(exc)
    else:
        state["rest_reachable"] = True
    return state


def validator_identity(binary: Path, keyring_backend: str, home: str | None) -> dict:
    cmd = [str(binary), "keys", "show", "validator", "--keyring-backend", keyring_backend, "--output", "json"]
    if home:
        cmd.extend(["--home", home])
    code, stdout, stderr = run_cmd(cmd)
    if code != 0:
        raise RuntimeError(stderr.strip() or stdout.strip() or "validator key lookup failed")

    payload = json.loads(stdout)
    pubkey_json = json.loads(payload["pubkey"])
    pubkey_hex = base64.b64decode(pubkey_json["key"]).hex()
    return {
        "address": payload["address"],
        "pubkey_hex": pubkey_hex,
        "sigil_id": derive_sigil_id(pubkey_hex),
    }


def query_sigil(binary: Path, node: str, sigil_id: str, home: str | None) -> dict:
    cmd = [str(binary), "query", "sigil", "sigil", sigil_id, "--node", node, "--output", "json"]
    if home:
        cmd.extend(["--home", home])
    code, stdout, stderr = run_cmd(cmd)
    if code != 0:
        raise RuntimeError(stderr.strip() or stdout.strip() or "sigil query failed")
    payload = json.loads(stdout)
    return payload.get("sigil", payload)


def ensure_local_sigil_cli(
    binary: Path,
    node: str,
    chain_id: str,
    keyring_backend: str,
    home: str | None,
    identity: dict,
) -> None:
    try:
        query_sigil(binary, node, identity["sigil_id"], home)
        return
    except Exception:  # noqa: BLE001
        pass

    tx_cmd = [
        str(binary),
        "tx",
        "sigil",
        "genesis",
        identity["pubkey_hex"],
        "--from",
        "validator",
        "--keyring-backend",
        keyring_backend,
        "--chain-id",
        chain_id,
        "--fees",
        "10000uoas",
        "--yes",
        "--output",
        "json",
        "--node",
        node,
    ]
    if home:
        tx_cmd.extend(["--home", home])
    code, stdout, stderr = run_cmd(tx_cmd)
    if code != 0:
        raise RuntimeError(stderr.strip() or stdout.strip() or "sigil genesis failed")
    wait_for(lambda: query_sigil(binary, node, identity["sigil_id"], home), 20, interval=2)


def send_tokens(
    binary: Path,
    node: str,
    chain_id: str,
    keyring_backend: str,
    home: str | None,
    to_address: str,
    amount_uoas: int,
) -> None:
    cmd = [
        str(binary),
        "tx",
        "send",
        "validator",
        to_address,
        f"{amount_uoas}uoas",
        "--from",
        "validator",
        "--keyring-backend",
        keyring_backend,
        "--chain-id",
        chain_id,
        "--fees",
        "10000uoas",
        "--yes",
        "--output",
        "json",
        "--node",
        node,
    ]
    if home:
        cmd.extend(["--home", home])
    code, stdout, stderr = run_cmd(cmd)
    if code != 0:
        raise RuntimeError(stderr.strip() or stdout.strip() or "funding tx failed")


def wait_for_balance(rest_url: str, address: str, minimum_uoas: int) -> None:
    def _balance_ok():
        data = rest_get_json(rest_url, f"/cosmos/bank/v1beta1/balances/{address}")
        for coin in data.get("balances", []):
            if coin.get("denom") == "uoas" and int(coin.get("amount", "0")) >= minimum_uoas:
                return True
        return False

    wait_for(_balance_ok, 20, interval=1)


def wait_for_dimensions(binary: Path, node: str, sigil_id: str, home: str | None, names: set[str]) -> dict:
    def _ready():
        sigil = query_sigil(binary, node, sigil_id, home)
        dimensions = sigil.get("dimension_pulses", {})
        if all(name in dimensions and int(dimensions[name]) > 0 for name in names):
            return {"sigil": sigil, "dimensions": dimensions}
        return None

    return wait_for(_ready, 20, interval=2)


def try_cli_live_pulse(
    binary: Path,
    rpc_url: str,
    rest_url: str,
    chain_id: str,
    keyring_backend: str,
    home: str | None,
) -> dict:
    chain = local_chain_state(rpc_url, rest_url)
    if not chain["reachable"] or not chain["rest_reachable"]:
        reason = chain["detail"] or chain["rest_detail"] or "local chain not reachable"
        return {"status": "error", "reason": reason}

    binary_state = chain_binary_state(binary)
    if not binary_state["has_pulse_cmd"]:
        return {"status": "error", "reason": "built oasyced does not expose `tx sigil pulse`"}

    identity = validator_identity(binary, keyring_backend, home)
    node = rpc_to_node(rpc_url)
    ensure_local_sigil_cli(binary, node, chain_id, keyring_backend, home, identity)

    tx_cmd = [
        str(binary),
        "tx",
        "sigil",
        "pulse",
        identity["sigil_id"],
        "--dimensions",
        "chain,thronglets",
        "--from",
        "validator",
        "--keyring-backend",
        keyring_backend,
        "--chain-id",
        chain_id,
        "--fees",
        "10000uoas",
        "--yes",
        "--output",
        "json",
        "--node",
        node,
    ]
    if home:
        tx_cmd.extend(["--home", home])
    code, stdout, stderr = run_cmd(tx_cmd)
    if code != 0:
        return {"status": "error", "reason": stderr.strip() or stdout.strip() or "pulse tx failed"}

    tx_payload = json.loads(stdout or "{}")
    try:
        observed = wait_for_dimensions(binary, node, identity["sigil_id"], home, {"chain", "thronglets"})
    except Exception as exc:  # noqa: BLE001
        return {"status": "error", "reason": str(exc)}
    return {
        "status": "ok",
        "txhash": tx_payload.get("txhash", ""),
        "sigil_id": identity["sigil_id"],
        "dimensions": observed["dimensions"],
    }


def try_sdk_live_pulse(
    binary: Path,
    rpc_url: str,
    rest_url: str,
    chain_id: str,
    keyring_backend: str,
    home: str | None,
    sdk_mode: str,
) -> dict:
    chain = local_chain_state(rpc_url, rest_url)
    if not chain["reachable"] or not chain["rest_reachable"]:
        reason = chain["detail"] or chain["rest_detail"] or "local chain not reachable"
        return {"status": "error", "reason": reason}

    ensure_local_no_proxy()
    _ensure_sdk_importable(sdk_mode)
    sdk_path = Path(os.environ.get("OASYCE_SDK_PATH", str(DEFAULT_SDK_PATH)))
    if sdk_path.exists() and str(sdk_path) not in sys.path:
        sys.path.insert(0, str(sdk_path))
    from oasyce_sdk import OasyceClient
    from oasyce_sdk.crypto import NativeSigner, Wallet

    wallet = Wallet.from_private_key(SDK_PULSE_PRIVATE_KEY)
    node = rpc_to_node(rpc_url)
    try:
        send_tokens(binary, node, chain_id, keyring_backend, home, wallet.address, 20_000_000)
        wait_for_balance(rest_url, wallet.address, 20_000_000)
        client = OasyceClient(rest_url)
        signer = NativeSigner(wallet, client, chain_id=chain_id)
        sigil_id = derive_sigil_id(wallet.public_key_bytes.hex())

        try:
            query_sigil(binary, node, sigil_id, home)
        except Exception:  # noqa: BLE001
            result = signer.create_sigil(wallet.public_key_bytes.hex())
            if not result.success:
                return {"status": "error", "reason": result.raw_log or f"code={result.code}"}
            time.sleep(2)

        pulse_result = signer.pulse_sigil(
            sigil_id,
            dimensions={"sdk": int(time.time()), "chain": int(time.time())},
        )
        if not pulse_result.success:
            return {"status": "error", "reason": pulse_result.raw_log or f"code={pulse_result.code}"}
        observed = wait_for_dimensions(binary, node, sigil_id, home, {"sdk", "chain"})
    except Exception as exc:  # noqa: BLE001
        return {"status": "error", "reason": str(exc)}
    return {
        "status": "ok",
        "txhash": pulse_result.tx_hash,
        "sigil_id": sigil_id,
        "dimensions": observed["dimensions"],
        "address": wallet.address,
    }


def report_ok(report: dict) -> bool:
    return all(
        [
            report["chain"]["source"]["status"] == "chain-ready",
            report["thronglets"]["status"] == "thronglets-ready",
            report["sdk"]["status"] == "sdk-ready",
            report["sdk_surface"]["status"] != "error",
            report["chain"]["cli_live_tx"]["status"] == "ok",
            report["chain"]["sdk_live_tx"]["status"] == "ok",
        ]
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--rpc", default=DEFAULT_RPC, help="Local chain RPC endpoint")
    parser.add_argument("--rest", default=DEFAULT_REST, help="Local chain REST endpoint")
    parser.add_argument("--chain-id", default=DEFAULT_CHAIN_ID, help="Chain ID for live pulse checks")
    parser.add_argument("--oasyced", default=str(DEFAULT_OASYCED), help="Path to oasyced binary")
    parser.add_argument("--home", default="", help="Optional oasyced home for local keyring")
    parser.add_argument("--keyring-backend", default=DEFAULT_KEYRING, help="CLI keyring backend")
    parser.add_argument(
        "--sdk-mode",
        default="source",
        choices=("source", "installed", "auto"),
        help="How chain-side pulse checks should resolve oasyce-sdk",
    )
    parser.add_argument("--sdk-path", default=str(DEFAULT_SDK_PATH), help="Preferred oasyce-sdk checkout path")
    parser.add_argument(
        "--thronglets-path",
        default=str(DEFAULT_THRONGLETS_PATH),
        help="Adjacent Thronglets checkout used for pulse compatibility checks",
    )
    parser.add_argument("--json", action="store_true", help="Emit JSON instead of human-readable text")
    args = parser.parse_args()

    os.environ["OASYCE_SDK_MODE"] = args.sdk_mode
    os.environ["OASYCE_SDK_PATH"] = args.sdk_path

    sdk_surface = inspect_sdk_surface(args.sdk_mode)
    report = {
        "classes": [],
        "chain": {
            "source": chain_source_state(),
            "binary": chain_binary_state(Path(args.oasyced)),
            "local_chain": local_chain_state(args.rpc, args.rest),
        },
        "thronglets": thronglets_state(Path(args.thronglets_path)),
        "sdk": sdk_pulse_state(sdk_surface),
        "sdk_surface": {
            "status": sdk_surface["status"],
            "module_path": sdk_surface["module_path"],
            "identity_seam": sdk_surface["identity_seam"],
            "warnings": sdk_surface["warnings"],
            "errors": sdk_surface["errors"],
        },
    }

    report["classes"].append(report["chain"]["source"]["status"])
    report["classes"].append(report["thronglets"]["status"])
    report["classes"].append(report["sdk"]["status"])

    report["chain"]["cli_live_tx"] = try_cli_live_pulse(
        Path(args.oasyced),
        args.rpc,
        args.rest,
        args.chain_id,
        args.keyring_backend,
        args.home or None,
    )
    report["chain"]["sdk_live_tx"] = try_sdk_live_pulse(
        Path(args.oasyced),
        args.rpc,
        args.rest,
        args.chain_id,
        args.keyring_backend,
        args.home or None,
        args.sdk_mode,
    )

    output = json.dumps(report, indent=2)
    print(output)

    return 0 if report_ok(report) else 1


if __name__ == "__main__":
    raise SystemExit(main())
