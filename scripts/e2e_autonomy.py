#!/usr/bin/env python3
"""Local autonomy acceptance harness for chain-side SDK-backed wrappers.

This script verifies the chain repo's own wrapper surfaces:

1. provider_agent.py --register + live provider HTTP service
2. data_agent.py --once registering one safe asset
3. consumer_agent.py one-shot invoke -> provider process -> feedback -> buy-shares

It does not replace scripts/e2e_test.sh, which remains the CLI/module E2E surface.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess
import sys
import tempfile
import textwrap
import threading
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.error import HTTPError, URLError
from urllib.request import ProxyHandler, Request, build_opener

from _sdk_compat import inspect_sdk_surface


ROOT = Path(__file__).resolve().parent.parent
SCRIPT_DIR = ROOT / "scripts"
DEFAULT_OASYCED = ROOT / "build" / "oasyced"
DEFAULT_REST = "http://127.0.0.1:1317"
DEFAULT_RPC = "http://127.0.0.1:26657"
DEFAULT_CHAIN_ID = "oasyce-local-1"
DEFAULT_KEYRING = "test"
DEFAULT_PROVIDER_PRICE = 500000
DEFAULT_FUNDING_UOAS = 50_000_000
HTTP = build_opener(ProxyHandler({}))

PROVIDER_PRIVATE_KEY = "11" * 32
CONSUMER_PRIVATE_KEY = "22" * 32
DATA_PRIVATE_KEY = "33" * 32


def info(message: str) -> None:
    print(f"[autonomy] {message}")


def fail(message: str) -> None:
    raise SystemExit(f"[autonomy] ERROR: {message}")


def find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def http_get_json(url: str) -> dict:
    with HTTP.open(Request(url), timeout=10) as resp:
        return json.loads(resp.read().decode())


def rest_get(base_url: str, path: str) -> dict:
    return http_get_json(f"{base_url.rstrip('/')}{path}")


def wait_for(predicate, timeout: float, interval: float = 1.0, desc: str = "condition"):
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
        fail(f"Timed out waiting for {desc}: {last_error}")
    fail(f"Timed out waiting for {desc}")


def tail_file(path: Path, limit: int = 80) -> str:
    if not path.exists():
        return "(log file missing)"
    lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
    return "\n".join(lines[-limit:])


def load_sdk_wallet(private_key_hex: str):
    sdk_mode = os.environ.get("OASYCE_SDK_MODE", "source").strip().lower()
    sdk_path = os.environ.get("OASYCE_SDK_PATH", str(ROOT.parent / "oasyce-sdk"))
    if sdk_mode in {"source", "auto"} and sdk_path not in sys.path and Path(sdk_path, "oasyce_sdk").exists():
        sys.path.insert(0, sdk_path)
    from oasyce_sdk.crypto.wallet import Wallet

    return Wallet.from_private_key(private_key_hex)


def rpc_to_node(rpc_url: str) -> str:
    if rpc_url.startswith("http://"):
        return "tcp://" + rpc_url[len("http://") :]
    if rpc_url.startswith("https://"):
        return "tcp://" + rpc_url[len("https://") :]
    return rpc_url


def cli_cmd(binary: str, *args: str, home: str | None = None, node: str | None = None) -> list[str]:
    cmd = [binary, *args]
    if home:
        cmd.extend(["--home", home])
    if node:
        cmd.extend(["--node", node])
    return cmd


def run_cli(binary: str, args: list[str], *, home: str | None = None, node: str | None = None) -> str:
    cmd = cli_cmd(binary, *args, home=home, node=node)
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
    if result.returncode != 0:
        fail(
            "CLI command failed:\n"
            + " ".join(cmd)
            + "\nstdout:\n"
            + result.stdout
            + "\nstderr:\n"
            + result.stderr
        )
    return result.stdout.strip() or result.stderr.strip()


def ensure_local_chain(rest_url: str, rpc_url: str, binary: str, keyring: str, home: str | None) -> str:
    try:
        status = http_get_json(f"{rpc_url.rstrip('/')}/status")
    except (HTTPError, URLError, TimeoutError, ValueError) as exc:
        fail(
            "Local chain RPC is not reachable. Start a local node before running autonomy acceptance. "
            f"rpc={rpc_url} err={exc}"
        )

    latest_height = status["result"]["sync_info"]["latest_block_height"]
    info(f"Local chain reachable at {rpc_url} (height={latest_height})")

    try:
        validator = run_cli(binary, ["keys", "show", "validator", "-a", "--keyring-backend", keyring], home=home)
    except SystemExit:
        fail(
            "Validator key `validator` not found in the local test keyring. "
            "Use the local node bootstrap flow from README/AGENTS before running this script."
        )
    info(f"Validator address: {validator}")

    try:
        rest_get(rest_url, f"/cosmos/bank/v1beta1/balances/{validator}")
    except Exception as exc:  # noqa: BLE001
        fail(f"Local chain REST is not reachable at {rest_url}: {exc}")
    return validator


def fund_address(
    binary: str,
    chain_id: str,
    keyring: str,
    from_name: str,
    to_address: str,
    amount_uoas: int,
    home: str | None,
    node: str,
) -> None:
    run_cli(
        binary,
        [
            "tx",
            "send",
            from_name,
            to_address,
            f"{amount_uoas}uoas",
            "--from",
            from_name,
            "--keyring-backend",
            keyring,
            "--chain-id",
            chain_id,
            "--fees",
            "10000uoas",
            "--yes",
            "--output",
            "json",
        ],
        home=home,
        node=node,
    )


def wait_for_balance(rest_url: str, address: str, minimum_uoas: int) -> None:
    def _balance_ok():
        data = rest_get(rest_url, f"/cosmos/bank/v1beta1/balances/{address}")
        for coin in data.get("balances", []):
            if coin.get("denom") == "uoas" and int(coin.get("amount", "0")) >= minimum_uoas:
                return True
        return False

    wait_for(_balance_ok, timeout=20, interval=1, desc=f"balance for {address}")


class MockUpstreamHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        pass

    def do_HEAD(self):
        self.send_response(200)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def do_POST(self):
        content_length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(content_length) if content_length else b"{}"
        try:
            payload = json.loads(body.decode("utf-8"))
        except json.JSONDecodeError:
            payload = {}
        prompt = payload.get("prompt", "")
        data = json.dumps(
            {
                "text": f"mock-upstream-ok: {prompt[:40]}",
                "usage": {
                    "prompt_tokens": 12,
                    "completion_tokens": 7,
                    "total_tokens": 19,
                },
            }
        ).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def start_mock_upstream(port: int):
    server = HTTPServer(("127.0.0.1", port), MockUpstreamHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def actor_env(
    actor_root: Path,
    rest_url: str,
    rpc_url: str,
    chain_id: str,
    private_key_hex: str,
    *,
    sdk_mode: str,
    sdk_path: str,
) -> dict[str, str]:
    env = os.environ.copy()
    env["PYTHONUNBUFFERED"] = "1"
    env["HOME"] = str(actor_root)
    env["OASYCE_DIR"] = str(actor_root / ".oasyce")
    env["OASYCE_MNEMONIC"] = private_key_hex
    env["OASYCE_CHAIN_REST"] = rest_url
    env["OASYCE_CHAIN_RPC"] = rpc_url
    env["OASYCE_CHAIN_ID"] = chain_id
    env["OASYCE_SDK_MODE"] = sdk_mode
    env["OASYCE_SDK_PATH"] = sdk_path
    local_hosts = "127.0.0.1,localhost"
    for key in ("NO_PROXY", "no_proxy"):
        current = env.get(key, "").strip()
        if current:
            if "127.0.0.1" not in current and "localhost" not in current:
                env[key] = current + "," + local_hosts
        else:
            env[key] = local_hosts
    return env


def run_python(script: Path, args: list[str], *, env: dict[str, str], log_path: Path) -> subprocess.CompletedProcess[str]:
    with log_path.open("w", encoding="utf-8") as log_file:
        result = subprocess.run(
            [sys.executable, str(script), *args],
            cwd=str(ROOT),
            env=env,
            stdout=log_file,
            stderr=subprocess.STDOUT,
            text=True,
            timeout=180,
        )
    return result


def start_provider_process(env: dict[str, str], log_path: Path) -> subprocess.Popen[str]:
    log_file = log_path.open("w", encoding="utf-8")
    process = subprocess.Popen(
        [sys.executable, str(SCRIPT_DIR / "provider_agent.py")],
        cwd=str(ROOT),
        env=env,
        stdout=log_file,
        stderr=subprocess.STDOUT,
        text=True,
    )
    process._codex_log_file = log_file  # type: ignore[attr-defined]
    return process


def stop_process(process: subprocess.Popen[str]) -> None:
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)
    log_file = getattr(process, "_codex_log_file", None)
    if log_file is not None:
        log_file.close()


def parse_capability_id(log_text: str) -> str:
    match = re.search(r"\bCAP_[A-Z0-9_]+\b", log_text)
    if not match:
        fail("Could not parse capability ID from provider registration output.")
    return match.group(0)


def find_asset_by_hash(rest_url: str, content_hash: str) -> dict | None:
    assets = rest_get(rest_url, "/oasyce/datarights/v1/data_assets").get("data_assets", [])
    for asset in assets:
        if asset.get("content_hash") == content_hash:
            return asset
    return None


def fetch_provider_capability(rest_url: str, provider_addr: str) -> dict:
    data = rest_get(rest_url, f"/oasyce/capability/v1/capabilities/provider/{provider_addr}")
    caps = data.get("capabilities", [])
    if not caps:
        fail(f"No capability found for provider {provider_addr}")
    return caps[-1]


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Run local autonomy acceptance for chain-side SDK-backed wrappers.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=textwrap.dedent(
            """\
            Preconditions:
              - local node running at the configured REST/RPC URLs
              - validator key named `validator` exists in the configured keyring
              - oasyce-sdk is available locally or via OASYCE_SDK_PATH
            """
        ),
    )
    parser.add_argument("--rest", default=DEFAULT_REST, help="Chain REST endpoint")
    parser.add_argument("--rpc", default=DEFAULT_RPC, help="Chain RPC endpoint")
    parser.add_argument("--chain-id", default=DEFAULT_CHAIN_ID, help="Chain ID")
    parser.add_argument("--oasyced", default=str(DEFAULT_OASYCED), help="Path to oasyced binary")
    parser.add_argument("--keyring-backend", default=DEFAULT_KEYRING, help="CLI keyring backend for funding")
    parser.add_argument("--home", default="", help="Optional oasyced home for local keyring")
    parser.add_argument("--provider-price", type=int, default=DEFAULT_PROVIDER_PRICE, help="Capability price in uoas")
    parser.add_argument("--funding-uoas", type=int, default=DEFAULT_FUNDING_UOAS, help="Initial balance per actor")
    parser.add_argument("--skip-buy-shares", action="store_true", help="Skip data-agent registration before consumer run")
    parser.add_argument(
        "--sdk-mode",
        default="source",
        choices=("source", "installed", "auto"),
        help="SDK resolution mode for chain-side acceptance (default: source)",
    )
    parser.add_argument(
        "--sdk-path",
        default=str(ROOT.parent / "oasyce-sdk"),
        help="Preferred oasyce-sdk checkout path for source/auto modes",
    )
    args = parser.parse_args()

    os.environ["OASYCE_SDK_MODE"] = args.sdk_mode
    os.environ["OASYCE_SDK_PATH"] = args.sdk_path

    sdk_surface = inspect_sdk_surface(args.sdk_mode)
    info(
        "SDK surface "
        f"status={sdk_surface['status']} mode={sdk_surface['requested_mode']} "
        f"seam={sdk_surface['identity_seam']} module={sdk_surface['module_path']}"
    )
    if sdk_surface["warnings"]:
        for warning in sdk_surface["warnings"]:
            info(f"SDK warning: {warning}")
    if sdk_surface["errors"]:
        fail("SDK surface check failed:\n" + json.dumps(sdk_surface, indent=2))

    oasyced_path = Path(args.oasyced)
    if not oasyced_path.exists():
        fail(f"oasyced binary not found: {oasyced_path}")

    validator = ensure_local_chain(args.rest, args.rpc, str(oasyced_path), args.keyring_backend, args.home or None)
    node = rpc_to_node(args.rpc)

    provider_wallet = load_sdk_wallet(PROVIDER_PRIVATE_KEY)
    consumer_wallet = load_sdk_wallet(CONSUMER_PRIVATE_KEY)
    data_wallet = load_sdk_wallet(DATA_PRIVATE_KEY)
    info(f"Provider wallet: {provider_wallet.address}")
    info(f"Consumer wallet: {consumer_wallet.address}")
    info(f"Data wallet: {data_wallet.address}")

    for address in (provider_wallet.address, consumer_wallet.address, data_wallet.address):
        fund_address(
            str(oasyced_path),
            args.chain_id,
            args.keyring_backend,
            "validator",
            address,
            args.funding_uoas,
            args.home or None,
            node,
        )
        wait_for_balance(args.rest, address, args.funding_uoas)

    provider_port = find_free_port()
    upstream_port = find_free_port()
    upstream_server, _ = start_mock_upstream(upstream_port)
    info(f"Mock upstream listening on 127.0.0.1:{upstream_port}")

    with tempfile.TemporaryDirectory(prefix="oasyce-autonomy-") as tmpdir:
        tmp = Path(tmpdir)
        logs = tmp / "logs"
        logs.mkdir(parents=True, exist_ok=True)

        provider_root = tmp / "provider"
        consumer_root = tmp / "consumer"
        data_root = tmp / "data"
        watch_dir = tmp / "watch"
        watch_dir.mkdir(parents=True, exist_ok=True)
        sample_file = watch_dir / "autonomy_sample.txt"
        sample_file.write_text("autonomy acceptance fixture\n", encoding="utf-8")

        provider_env = actor_env(
            provider_root,
            args.rest,
            args.rpc,
            args.chain_id,
            PROVIDER_PRIVATE_KEY,
            sdk_mode=args.sdk_mode,
            sdk_path=args.sdk_path,
        )
        provider_env.update(
            {
                "UPSTREAM_API_URL": f"http://127.0.0.1:{upstream_port}/v1/chat",
                "UPSTREAM_API_KEY": "local-test-key",
                "PROVIDER_PORT": str(provider_port),
                "OASYCE_ALERT_LOG": str(tmp / "provider-alert.log"),
                "OASYCE_ALERT_STATE_DIR": str(tmp / "provider-alerts"),
            }
        )
        consumer_env = actor_env(
            consumer_root,
            args.rest,
            args.rpc,
            args.chain_id,
            CONSUMER_PRIVATE_KEY,
            sdk_mode=args.sdk_mode,
            sdk_path=args.sdk_path,
        )
        consumer_env.update(
            {
                "PROVIDER_ENDPOINT": f"http://127.0.0.1:{provider_port}",
                "CONSUMER_STATE_FILE": str(tmp / "consumer-state.json"),
                "MIN_BALANCE_UOAS": "1000000",
                "FAUCET_URL": "http://127.0.0.1:18080",
            }
        )
        data_env = actor_env(
            data_root,
            args.rest,
            args.rpc,
            args.chain_id,
            DATA_PRIVATE_KEY,
            sdk_mode=args.sdk_mode,
            sdk_path=args.sdk_path,
        )
        data_env.update(
            {
                "WATCH_DIRS": str(watch_dir),
                "STATE_FILE": str(tmp / "data-state.json"),
                "DATA_AGENT_PORT": str(find_free_port()),
                "DEFAULT_TAGS": "autonomy,acceptance",
                "MAX_RISK_LEVEL": "low",
            }
        )

        provider_register_log = logs / "provider-register.log"
        register_result = run_python(
            SCRIPT_DIR / "provider_agent.py",
            ["--register", "--name", "Autonomy Provider", "--price", str(args.provider_price)],
            env=provider_env,
            log_path=provider_register_log,
        )
        if register_result.returncode != 0:
            fail("Provider registration failed:\n" + tail_file(provider_register_log))

        capability_id = parse_capability_id(provider_register_log.read_text(encoding="utf-8", errors="replace"))
        provider_env["OASYCE_CAPABILITY_ID"] = capability_id
        info(f"Registered provider capability: {capability_id}")

        provider_process = start_provider_process(provider_env, logs / "provider.log")
        try:
            wait_for(
                lambda: http_get_json(f"http://127.0.0.1:{provider_port}/health").get("status") == "ok",
                timeout=30,
                interval=1,
                desc="provider /health",
            )

            if not args.skip_buy_shares:
                data_result = run_python(
                    SCRIPT_DIR / "data_agent.py",
                    ["--once", "--watch-dirs", str(watch_dir)],
                    env=data_env,
                    log_path=logs / "data-agent.log",
                )
                if data_result.returncode != 0:
                    fail("Data agent run failed:\n" + tail_file(logs / "data-agent.log"))

            consumer_result = run_python(
                SCRIPT_DIR / "consumer_agent.py",
                [],
                env=consumer_env,
                log_path=logs / "consumer.log",
            )
            if consumer_result.returncode != 0:
                fail("Consumer run failed:\n" + tail_file(logs / "consumer.log"))

            consumer_state = json.loads((tmp / "consumer-state.json").read_text(encoding="utf-8"))
            if consumer_state.get("last_status") != "ok":
                fail("Consumer did not finish successfully:\n" + json.dumps(consumer_state, indent=2))
            if consumer_state.get("total_invocations", 0) < 1:
                fail("Consumer did not record an invocation.")
            if consumer_state.get("total_settlements", 0) < 1:
                fail("Consumer did not record feedback/settlement success.")

            provider_cap = fetch_provider_capability(args.rest, provider_wallet.address)
            if provider_cap.get("id") != capability_id:
                fail(f"Provider capability mismatch: expected {capability_id}, got {provider_cap.get('id')}")

            if not args.skip_buy_shares:
                content_hash = hashlib.sha256(sample_file.read_bytes()).hexdigest()
                asset = wait_for(
                    lambda: find_asset_by_hash(args.rest, content_hash),
                    timeout=20,
                    interval=1,
                    desc="registered data asset",
                )
                if consumer_state.get("total_data_purchases", 0) < 1:
                    fail(
                        "Consumer did not record a data-asset purchase even though an asset was registered.\n"
                        + json.dumps(consumer_state, indent=2)
                    )
                info(f"Registered asset: {asset.get('id')} ({asset.get('name')})")

            info("Autonomy acceptance passed.")
            print(
                json.dumps(
                    {
                        "status": "ok",
                        "provider_capability_id": capability_id,
                        "consumer_state": consumer_state,
                        "provider_address": provider_wallet.address,
                        "consumer_address": consumer_wallet.address,
                        "data_address": data_wallet.address,
                    },
                    indent=2,
                )
            )
            return 0
        finally:
            stop_process(provider_process)
            upstream_server.shutdown()
            upstream_server.server_close()


if __name__ == "__main__":
    raise SystemExit(main())
