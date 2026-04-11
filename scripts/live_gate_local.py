#!/usr/bin/env python3
"""Run the chain-side tempnet live gate end to end."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import sys
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import ProxyHandler, Request, build_opener


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_SDK_PATH = ROOT.parent / "oasyce-sdk"
DEFAULT_THRONGLETS_PATH = ROOT.parent / "Thronglets"
DEFAULT_CHAIN_ID = "oasyce-live-gate-1"
DEFAULT_KEYRING = "test"
DEFAULT_BINARY = ROOT / "build" / "oasyced"
HTTP = build_opener(ProxyHandler({}))


def info(message: str) -> None:
    print(f"[live-gate] {message}")


def run_cmd(
    args: list[str],
    *,
    cwd: Path = ROOT,
    env: dict[str, str] | None = None,
    timeout: int = 180,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        args,
        cwd=str(cwd),
        env=env,
        capture_output=True,
        text=True,
        timeout=timeout,
    )


def find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def http_get_json(url: str) -> dict:
    with HTTP.open(Request(url), timeout=10) as resp:
        return json.loads(resp.read().decode())


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


def with_local_no_proxy(env: dict[str, str]) -> dict[str, str]:
    updated = dict(env)
    local_hosts = "127.0.0.1,localhost"
    for key in ("NO_PROXY", "no_proxy"):
        current = updated.get(key, "").strip()
        if current:
            if "127.0.0.1" not in current and "localhost" not in current:
                updated[key] = current + "," + local_hosts
        else:
            updated[key] = local_hosts
    return updated


def _replace_section_value(text: str, section: str, pattern: str, replacement: str) -> str:
    compiled = re.compile(rf"(\[{re.escape(section)}\][^\[]*?){pattern}", re.DOTALL)
    updated, count = compiled.subn(rf"\1{replacement}", text, count=1)
    if count == 0:
        raise RuntimeError(f"Could not patch [{section}] section using pattern: {pattern}")
    return updated


def patch_genesis(genesis_path: Path) -> None:
    genesis = json.loads(genesis_path.read_text(encoding="utf-8"))
    genesis["app_state"]["oasyce_capability"]["params"]["min_provider_stake"] = {
        "denom": "uoas",
        "amount": "0",
    }
    genesis_path.write_text(json.dumps(genesis, indent=2) + "\n", encoding="utf-8")


def patch_config_toml(config_path: Path, rpc_port: int, p2p_port: int) -> None:
    text = config_path.read_text(encoding="utf-8")
    text = _replace_section_value(text, "rpc", r'laddr = ".*?"', f'laddr = "tcp://127.0.0.1:{rpc_port}"')
    text = _replace_section_value(text, "p2p", r'laddr = ".*?"', f'laddr = "tcp://127.0.0.1:{p2p_port}"')
    config_path.write_text(text, encoding="utf-8")


def patch_app_toml(app_path: Path, rest_port: int, grpc_port: int) -> None:
    text = app_path.read_text(encoding="utf-8")
    text = _replace_section_value(text, "api", r'enable = (true|false)', "enable = true")
    text = _replace_section_value(text, "api", r'address = ".*?"', f'address = "tcp://127.0.0.1:{rest_port}"')
    text = _replace_section_value(text, "grpc", r'address = ".*?"', f'address = "127.0.0.1:{grpc_port}"')
    text = re.sub(r'^minimum-gas-prices = ".*?"$', 'minimum-gas-prices = "0uoas"', text, count=1, flags=re.MULTILINE)
    app_path.write_text(text, encoding="utf-8")


def build_binary(binary_path: Path) -> dict:
    binary_path.parent.mkdir(parents=True, exist_ok=True)
    result = run_cmd(["make", "build"], timeout=600)
    if result.returncode != 0:
        return {
            "status": "error",
            "stdout": result.stdout,
            "stderr": result.stderr,
        }
    help_result = run_cmd([str(binary_path), "tx", "sigil", "--help"])
    help_text = (help_result.stdout or "") + "\n" + (help_result.stderr or "")
    has_pulse_cmd = "pulse" in help_text
    return {
        "status": "ok" if help_result.returncode == 0 and has_pulse_cmd else "error",
        "binary": str(binary_path),
        "has_pulse_cmd": has_pulse_cmd,
        "stdout": result.stdout,
        "stderr": result.stderr,
        "help": help_text.strip(),
    }


def init_tempnet(binary_path: Path, home: Path, chain_id: str, keyring_backend: str, ports: dict[str, int]) -> dict:
    commands = [
        [str(binary_path), "init", "live-gate", "--chain-id", chain_id, "--home", str(home)],
        [str(binary_path), "keys", "add", "validator", "--keyring-backend", keyring_backend, "--home", str(home)],
        [str(binary_path), "genesis", "add-genesis-account", "validator", "1000000000uoas", "--keyring-backend", keyring_backend, "--home", str(home)],
        [
            str(binary_path),
            "genesis",
            "gentx",
            "validator",
            "500000000uoas",
            "--chain-id",
            chain_id,
            "--keyring-backend",
            keyring_backend,
            "--home",
            str(home),
        ],
        [str(binary_path), "genesis", "collect-gentxs", "--home", str(home)],
    ]
    for command in commands:
        result = run_cmd(command, timeout=180)
        if result.returncode != 0:
            return {
                "status": "error",
                "command": command,
                "stdout": result.stdout,
                "stderr": result.stderr,
            }

    patch_genesis(home / "config" / "genesis.json")
    patch_config_toml(home / "config" / "config.toml", ports["rpc"], ports["p2p"])
    patch_app_toml(home / "config" / "app.toml", ports["rest"], ports["grpc"])
    return {
        "status": "ok",
        "home": str(home),
        "chain_id": chain_id,
        "ports": ports,
    }


def start_tempnet(binary_path: Path, home: Path, log_path: Path) -> subprocess.Popen[str]:
    log_file = log_path.open("w", encoding="utf-8")
    process = subprocess.Popen(
        [str(binary_path), "start", "--home", str(home), "--minimum-gas-prices", "0uoas"],
        cwd=str(ROOT),
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
            process.wait(timeout=10)
    log_file = getattr(process, "_codex_log_file", None)
    if log_file is not None:
        log_file.close()


def wait_for_localnet(rpc_url: str, rest_url: str) -> dict:
    wait_for(lambda: http_get_json(f"{rpc_url.rstrip('/')}/status"), 60, interval=2)
    wait_for(lambda: http_get_json(f"{rest_url.rstrip('/')}/cosmos/base/tendermint/v1beta1/node_info"), 60, interval=2)
    status = http_get_json(f"{rpc_url.rstrip('/')}/status")
    return {
        "status": "ok",
        "latest_height": status["result"]["sync_info"]["latest_block_height"],
        "rpc": rpc_url,
        "rest": rest_url,
    }


def extract_last_json(text: str) -> dict | None:
    positions = [idx for idx, char in enumerate(text) if char == "{"] 
    for start in reversed(positions):
        chunk = text[start:].strip()
        try:
            return json.loads(chunk)
        except json.JSONDecodeError:
            continue
    return None


def run_json_script(args: list[str], *, env: dict[str, str] | None = None, timeout: int = 300) -> tuple[int, dict | None, str, str]:
    result = run_cmd(args, env=env, timeout=timeout)
    payload = extract_last_json((result.stdout or "").strip())
    return result.returncode, payload, result.stdout, result.stderr


def _allowed_warning(message: str) -> bool:
    return message.startswith("distribution metadata drift:")


def finalize_report(report: dict) -> dict:
    warnings: list[str] = []
    sdk_surface = report.get("sdk_surface", {})
    warnings.extend(sdk_surface.get("warnings", []))
    pulse_surface = report.get("pulse_compat", {}).get("sdk_surface", {})
    warnings.extend(pulse_surface.get("warnings", []))
    deduped: list[str] = []
    seen = set()
    for item in warnings:
        if item not in seen:
            seen.add(item)
            deduped.append(item)
    report["warnings"] = deduped

    allowed_warnings = all(_allowed_warning(item) for item in deduped)
    statuses = [
        report.get("build", {}).get("status") == "ok",
        report.get("localnet", {}).get("status") == "ok",
        report.get("sdk_surface", {}).get("status") in {"ok", "warn"},
        report.get("pulse_compat", {}).get("chain", {}).get("source", {}).get("status") == "chain-ready",
        report.get("pulse_compat", {}).get("thronglets", {}).get("status") == "thronglets-ready",
        report.get("pulse_compat", {}).get("sdk", {}).get("status") == "sdk-ready",
        report.get("pulse_compat", {}).get("chain", {}).get("cli_live_tx", {}).get("status") == "ok",
        report.get("pulse_compat", {}).get("chain", {}).get("sdk_live_tx", {}).get("status") == "ok",
        report.get("autonomy", {}).get("status") == "ok",
        allowed_warnings,
    ]
    report["status"] = "ok" if all(statuses) else "error"
    return report


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sdk-path", default=str(DEFAULT_SDK_PATH), help="Adjacent oasyce-sdk checkout")
    parser.add_argument("--thronglets-path", default=str(DEFAULT_THRONGLETS_PATH), help="Adjacent Thronglets checkout")
    parser.add_argument("--chain-id", default=DEFAULT_CHAIN_ID, help="Tempnet chain ID")
    parser.add_argument("--keyring-backend", default=DEFAULT_KEYRING, help="Keyring backend for tempnet")
    parser.add_argument("--sdk-mode", default="source", choices=("source",), help="SDK resolution mode for live gate")
    parser.add_argument("--keep-temp-dir", action="store_true", help="Keep the tempnet directory for debugging")
    parser.add_argument("--json", action="store_true", help="Emit JSON only")
    args = parser.parse_args()

    os.environ["OASYCE_SDK_MODE"] = args.sdk_mode
    os.environ["OASYCE_SDK_PATH"] = args.sdk_path
    os.environ.update(with_local_no_proxy(os.environ))

    report = {
        "build": {},
        "localnet": {},
        "sdk_surface": {},
        "pulse_compat": {},
        "autonomy": {},
        "warnings": [],
        "status": "error",
    }

    temp_ctx = tempfile.TemporaryDirectory(prefix="oasyce-live-gate-") if not args.keep_temp_dir else None
    temp_root = Path(temp_ctx.name) if temp_ctx is not None else Path(tempfile.mkdtemp(prefix="oasyce-live-gate-"))
    logs_dir = temp_root / "logs"
    logs_dir.mkdir(parents=True, exist_ok=True)
    node_log = logs_dir / "node.log"
    ports = {
        "rpc": find_free_port(),
        "rest": find_free_port(),
        "grpc": find_free_port(),
        "p2p": find_free_port(),
    }
    rpc_url = f"http://127.0.0.1:{ports['rpc']}"
    rest_url = f"http://127.0.0.1:{ports['rest']}"
    home = temp_root / "home"
    process: subprocess.Popen[str] | None = None

    try:
        info("Building fresh oasyced binary")
        report["build"] = build_binary(DEFAULT_BINARY)
        if report["build"]["status"] != "ok":
            raise RuntimeError("build step failed")

        info("Initializing tempnet")
        report["localnet"] = init_tempnet(DEFAULT_BINARY, home, args.chain_id, args.keyring_backend, ports)
        if report["localnet"]["status"] != "ok":
            raise RuntimeError("tempnet init failed")

        info("Starting tempnet")
        process = start_tempnet(DEFAULT_BINARY, home, node_log)
        localnet_ready = wait_for_localnet(rpc_url, rest_url)
        localnet_ready["home"] = str(home)
        localnet_ready["log"] = str(node_log)
        report["localnet"] = localnet_ready

        script_env = with_local_no_proxy(os.environ.copy())
        script_env["OASYCE_SDK_MODE"] = args.sdk_mode
        script_env["OASYCE_SDK_PATH"] = args.sdk_path

        info("Running SDK surface check")
        sdk_rc, sdk_payload, sdk_stdout, sdk_stderr = run_json_script(
            [sys.executable, str(ROOT / "scripts" / "check_sdk_surface.py"), "--mode", "source", "--json"],
            env=script_env,
        )
        report["sdk_surface"] = sdk_payload or {"status": "error", "stdout": sdk_stdout, "stderr": sdk_stderr}
        if sdk_rc != 0 or report["sdk_surface"].get("status") == "error":
            raise RuntimeError("sdk surface check failed")

        info("Running Pulse compatibility check")
        pulse_rc, pulse_payload, pulse_stdout, pulse_stderr = run_json_script(
            [
                sys.executable,
                str(ROOT / "scripts" / "check_pulse_compat.py"),
                "--sdk-mode",
                "source",
                "--sdk-path",
                args.sdk_path,
                "--thronglets-path",
                args.thronglets_path,
                "--oasyced",
                str(DEFAULT_BINARY),
                "--home",
                str(home),
                "--keyring-backend",
                args.keyring_backend,
                "--chain-id",
                args.chain_id,
                "--rpc",
                rpc_url,
                "--rest",
                rest_url,
                "--json",
            ],
            env=script_env,
            timeout=300,
        )
        report["pulse_compat"] = pulse_payload or {"status": "error", "stdout": pulse_stdout, "stderr": pulse_stderr}
        if pulse_rc != 0:
            raise RuntimeError("pulse compatibility check failed")

        info("Running autonomy acceptance")
        autonomy_result = run_cmd(
            [
                sys.executable,
                str(ROOT / "scripts" / "e2e_autonomy.py"),
                "--sdk-mode",
                "source",
                "--sdk-path",
                args.sdk_path,
                "--oasyced",
                str(DEFAULT_BINARY),
                "--home",
                str(home),
                "--keyring-backend",
                args.keyring_backend,
                "--chain-id",
                args.chain_id,
                "--rpc",
                rpc_url,
                "--rest",
                rest_url,
            ],
            env=script_env,
            timeout=600,
        )
        autonomy_payload = extract_last_json(autonomy_result.stdout or "")
        report["autonomy"] = autonomy_payload or {}
        report["autonomy"]["status"] = "ok" if autonomy_result.returncode == 0 else "error"
        if autonomy_result.returncode != 0:
            report["autonomy"]["stdout"] = autonomy_result.stdout
            report["autonomy"]["stderr"] = autonomy_result.stderr
            raise RuntimeError("autonomy acceptance failed")
    except Exception as exc:  # noqa: BLE001
        report["error"] = str(exc)
        report = finalize_report(report)
        return_code = 1
    else:
        report = finalize_report(report)
        return_code = 0 if report["status"] == "ok" else 1
    finally:
        if process is not None:
            stop_process(process)
        if args.keep_temp_dir:
            info(f"Tempnet kept at {temp_root}")
        elif temp_ctx is not None:
            temp_ctx.cleanup()

    output = json.dumps(report, indent=2)
    print(output)
    return return_code


if __name__ == "__main__":
    raise SystemExit(main())
