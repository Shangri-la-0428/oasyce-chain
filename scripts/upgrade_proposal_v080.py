#!/usr/bin/env python3
"""Render, validate, and dry-run the v0.8.0 software upgrade proposal."""

from __future__ import annotations

import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import shlex
import sys
import tempfile
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
UPGRADE_DIR = ROOT / "docs" / "upgrades" / "v0.8.0"
PROPOSAL_TEMPLATE = UPGRADE_DIR / "proposal.template.json"
METADATA_TEMPLATE = UPGRADE_DIR / "metadata.template.json"
DEFAULT_PLAN_NAME = "v0.8.0"
DEFAULT_DEPOSIT = "100000000uoas"
DEFAULT_TITLE = "Upgrade x/sigil to v2 effective activity height migration (v0.8.0)"
DEFAULT_SUMMARY = (
    "Apply the x/sigil v1 -> v2 state migration to rebuild active liveness indexes "
    "from MaxPulseHeight() semantics. State migration only, no new stores."
)
DEFAULT_BINARY = ROOT / "build" / "oasyced"
DEFAULT_CHAIN_ID = "oasyce-live-gate-1"
DEFAULT_KEYRING = "test"
DEFAULT_FEES = "20000uoas"
DEFAULT_FORUM_URL = ""
DEFAULT_METADATA_REF = "ipfs://PENDING_V080_METADATA_CID"
HEIGHT_PLACEHOLDER = "__UPGRADE_HEIGHT__"
TITLE_PLACEHOLDER = "__TITLE__"
SUMMARY_PLACEHOLDER = "__SUMMARY__"
METADATA_REF_PLACEHOLDER = "__METADATA_REF__"
FORUM_URL_PLACEHOLDER = "__PROPOSAL_FORUM_URL__"
EXPECTED_MSG_TYPE = "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade"
EXPECTED_AUTHORITY = "oasyce10d07y265gmmuvt4z0w9aw880jnsr700jjuangx"
DEPOSIT_RE = re.compile(r"^[1-9][0-9]*uoas$")
BECH32_RE = re.compile(r"^[a-z0-9]+1[0-9a-z]{10,}$")
MAX_METADATA_LEN = 255


def _load_live_gate_module():
    try:
        import live_gate_local as module  # type: ignore

        return module
    except ModuleNotFoundError:
        script_path = Path(__file__).resolve().parent / "live_gate_local.py"
        spec = importlib.util.spec_from_file_location("upgrade_live_gate_local", script_path)
        module = importlib.util.module_from_spec(spec)
        assert spec.loader is not None
        sys.modules["upgrade_live_gate_local"] = module
        spec.loader.exec_module(module)
        return module


live_gate = _load_live_gate_module()


def info(message: str) -> None:
    print(f"[upgrade-v080] {message}")


def load_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def replace_placeholders(value: Any, replacements: dict[str, str]) -> Any:
    if isinstance(value, str):
        for old, new in replacements.items():
            value = value.replace(old, new)
        return value
    if isinstance(value, list):
        return [replace_placeholders(item, replacements) for item in value]
    if isinstance(value, dict):
        return {key: replace_placeholders(item, replacements) for key, item in value.items()}
    return value


def render_metadata(template: dict[str, Any], *, title: str, summary: str, forum_url: str) -> dict[str, Any]:
    replacements = {
        TITLE_PLACEHOLDER: title,
        SUMMARY_PLACEHOLDER: summary,
        FORUM_URL_PLACEHOLDER: forum_url,
    }
    return replace_placeholders(template, replacements)


def render_proposal(
    template: dict[str, Any],
    *,
    height: int,
    title: str,
    summary: str,
    deposit: str,
    metadata_ref: str,
) -> dict[str, Any]:
    rendered = replace_placeholders(
        template,
        {
            TITLE_PLACEHOLDER: title,
            SUMMARY_PLACEHOLDER: summary,
            METADATA_REF_PLACEHOLDER: metadata_ref,
        },
    )
    rendered["deposit"] = deposit
    rendered["title"] = title
    rendered["summary"] = summary
    rendered["messages"][0]["plan"]["height"] = height
    return rendered


def validate_proposal_payload(payload: dict[str, Any]) -> list[str]:
    errors: list[str] = []

    messages = payload.get("messages")
    if not isinstance(messages, list) or not messages:
        return ["messages[0] is required"]

    msg = messages[0]
    if msg.get("@type") != EXPECTED_MSG_TYPE:
        errors.append(f"messages[0].@type must be {EXPECTED_MSG_TYPE}")
    if msg.get("authority") != EXPECTED_AUTHORITY:
        errors.append(f"messages[0].authority must be {EXPECTED_AUTHORITY}")

    plan = msg.get("plan", {})
    if plan.get("name") != DEFAULT_PLAN_NAME:
        errors.append(f"messages[0].plan.name must be {DEFAULT_PLAN_NAME}")
    height = plan.get("height")
    if not isinstance(height, int) or height <= 0:
        errors.append("messages[0].plan.height must be a positive integer")
    info_value = str(plan.get("info", ""))
    for token in ("sigil v1 -> v2", "effective activity height migration", "state migration only", "no new stores"):
        if token not in info_value:
            errors.append(f"messages[0].plan.info must mention: {token}")

    title = str(payload.get("title", "")).strip()
    if not title or TITLE_PLACEHOLDER in title:
        errors.append("title must be non-empty and rendered")

    summary = str(payload.get("summary", "")).strip()
    if not summary or SUMMARY_PLACEHOLDER in summary:
        errors.append("summary must be non-empty and rendered")

    deposit = str(payload.get("deposit", "")).strip()
    if not DEPOSIT_RE.fullmatch(deposit):
        errors.append("deposit must match ^[1-9][0-9]*uoas$")

    metadata_raw = payload.get("metadata")
    if not isinstance(metadata_raw, str) or not metadata_raw.strip():
        errors.append("metadata must be a non-empty reference string")
    else:
        if METADATA_REF_PLACEHOLDER in metadata_raw:
            errors.append("metadata must be rendered")
        if len(metadata_raw.encode("utf-8")) > MAX_METADATA_LEN:
            errors.append(f"metadata must be at most {MAX_METADATA_LEN} bytes")

    return errors


def build_submit_command(
    proposal_path: Path,
    *,
    binary: Path,
    from_name: str,
    chain_id: str,
    fees: str,
    home: str | None = None,
    keyring_backend: str | None = None,
    node: str | None = None,
    dry_run: bool = False,
) -> list[str]:
    cmd = [str(binary), "tx", "gov", "submit-proposal", str(proposal_path), "--from", from_name, "--chain-id", chain_id, "--fees", fees, "--output", "json"]
    if home:
        cmd.extend(["--home", home])
    if keyring_backend:
        cmd.extend(["--keyring-backend", keyring_backend])
    if node:
        cmd.extend(["--node", node])
    if dry_run:
        cmd.append("--dry-run")
    else:
        cmd.append("--yes")
    return cmd


def is_bech32_address(value: str) -> bool:
    return bool(BECH32_RE.fullmatch(value))


def resolve_from_address(
    from_name: str,
    *,
    binary: Path,
    home: str,
    keyring_backend: str,
) -> str:
    if is_bech32_address(from_name):
        return from_name

    result = live_gate.run_cmd(
        [
            str(binary),
            "keys",
            "show",
            from_name,
            "-a",
            "--home",
            home,
            "--keyring-backend",
            keyring_backend,
        ],
        env=live_gate.with_local_no_proxy(os.environ.copy()),
        timeout=60,
    )
    if result.returncode != 0:
        stderr = (result.stderr or "").strip()
        stdout = (result.stdout or "").strip()
        detail = stderr or stdout or "unknown error"
        raise RuntimeError(f"failed to resolve --from address for {from_name}: {detail}")

    address = (result.stdout or "").strip()
    if not is_bech32_address(address):
        raise RuntimeError(f"failed to resolve --from address for {from_name}: unexpected key output {address!r}")
    return address


def render_artifacts(
    *,
    height: int,
    title: str,
    summary: str,
    deposit: str,
    forum_url: str,
    metadata_ref: str,
    proposal_template: Path = PROPOSAL_TEMPLATE,
    metadata_template: Path = METADATA_TEMPLATE,
    proposal_output: Path | None = None,
    metadata_output: Path | None = None,
) -> dict[str, Any]:
    proposal_template_payload = load_json(proposal_template)
    metadata_template_payload = load_json(metadata_template)
    metadata_payload = render_metadata(metadata_template_payload, title=title, summary=summary, forum_url=forum_url)
    metadata_ref = metadata_ref or forum_url or DEFAULT_METADATA_REF
    proposal_payload = render_proposal(
        proposal_template_payload,
        height=height,
        title=title,
        summary=summary,
        deposit=deposit,
        metadata_ref=metadata_ref,
    )
    errors = validate_proposal_payload(proposal_payload)
    if errors:
        raise ValueError("; ".join(errors))

    if proposal_output is None:
        proposal_output = Path(tempfile.mkdtemp(prefix="oasyce-upgrade-v080-")) / "proposal.json"
    if metadata_output is None:
        metadata_output = proposal_output.with_name("metadata.json")

    write_json(proposal_output, proposal_payload)
    write_json(metadata_output, metadata_payload)

    submit_cmd = build_submit_command(
        proposal_output,
        binary=DEFAULT_BINARY,
        from_name="validator",
        chain_id=DEFAULT_CHAIN_ID,
        fees=DEFAULT_FEES,
        dry_run=False,
    )

    return {
        "status": "ok",
        "proposal_path": str(proposal_output),
        "metadata_path": str(metadata_output),
        "metadata_ref": metadata_ref,
        "proposal": proposal_payload,
        "metadata": metadata_payload,
        "submit_command": shlex.join(submit_cmd),
    }


def validate_file(proposal_path: Path) -> dict[str, Any]:
    payload = load_json(proposal_path)
    errors = validate_proposal_payload(payload)
    plan = payload.get("messages", [{}])[0].get("plan", {})
    return {
        "status": "ok" if not errors else "error",
        "proposal_path": str(proposal_path),
        "plan_name": plan.get("name"),
        "plan_height": plan.get("height"),
        "deposit": payload.get("deposit"),
        "errors": errors,
    }


def run_gov_dry_run(
    proposal_path: Path,
    *,
    binary: Path,
    home: str,
    chain_id: str,
    keyring_backend: str,
    rpc_url: str,
    fees: str,
    from_name: str = "validator",
) -> dict[str, Any]:
    from_value = resolve_from_address(
        from_name,
        binary=binary,
        home=home,
        keyring_backend=keyring_backend,
    )
    cmd = build_submit_command(
        proposal_path,
        binary=binary,
        from_name=from_value,
        chain_id=chain_id,
        fees=fees,
        home=home,
        keyring_backend=keyring_backend,
        node=rpc_url.replace("http://", "tcp://").replace("https://", "tcp://"),
        dry_run=True,
    )
    env = live_gate.with_local_no_proxy(os.environ.copy())
    result = live_gate.run_cmd(cmd, env=env, timeout=180)
    return {
        "status": "ok" if result.returncode == 0 else "error",
        "command": shlex.join(cmd),
        "stdout": result.stdout,
        "stderr": result.stderr,
        "returncode": result.returncode,
    }


def proposal_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    render = subparsers.add_parser("render", help="Render proposal and metadata JSON from templates")
    render.add_argument("--height", type=int, required=True, help="Upgrade height placeholder replacement")
    render.add_argument("--title", default=DEFAULT_TITLE, help="Proposal title")
    render.add_argument("--summary", default=DEFAULT_SUMMARY, help="Proposal summary")
    render.add_argument("--deposit", default=DEFAULT_DEPOSIT, help="Proposal deposit in uoas")
    render.add_argument("--proposal-forum-url", default=DEFAULT_FORUM_URL, help="Optional proposal forum URL")
    render.add_argument("--metadata-ref", default="", help="Short on-chain metadata reference (defaults to forum URL or ipfs placeholder)")
    render.add_argument("--proposal-output", default="", help="Where to write the rendered proposal JSON")
    render.add_argument("--metadata-output", default="", help="Where to write the rendered metadata JSON")
    render.add_argument("--proposal-template", default=str(PROPOSAL_TEMPLATE), help="Proposal template path")
    render.add_argument("--metadata-template", default=str(METADATA_TEMPLATE), help="Metadata template path")
    render.add_argument("--json", action="store_true", help="Emit JSON only")

    validate = subparsers.add_parser("validate", help="Validate a rendered proposal JSON file")
    validate.add_argument("proposal", help="Path to rendered proposal JSON")
    validate.add_argument("--json", action="store_true", help="Emit JSON only")

    dry_run = subparsers.add_parser("dry-run", help="Render and submit the proposal via local dry-run")
    dry_run.add_argument("--height", type=int, required=True, help="Upgrade height placeholder replacement")
    dry_run.add_argument("--title", default=DEFAULT_TITLE, help="Proposal title")
    dry_run.add_argument("--summary", default=DEFAULT_SUMMARY, help="Proposal summary")
    dry_run.add_argument("--deposit", default=DEFAULT_DEPOSIT, help="Proposal deposit in uoas")
    dry_run.add_argument("--proposal-forum-url", default=DEFAULT_FORUM_URL, help="Optional proposal forum URL")
    dry_run.add_argument("--metadata-ref", default="", help="Short on-chain metadata reference (defaults to forum URL or ipfs placeholder)")
    dry_run.add_argument("--proposal-template", default=str(PROPOSAL_TEMPLATE), help="Proposal template path")
    dry_run.add_argument("--metadata-template", default=str(METADATA_TEMPLATE), help="Metadata template path")
    dry_run.add_argument("--proposal-output", default="", help="Optional path for rendered proposal JSON")
    dry_run.add_argument("--metadata-output", default="", help="Optional path for rendered metadata JSON")
    dry_run.add_argument("--network", choices=("tempnet", "current"), default="tempnet", help="Dry-run target")
    dry_run.add_argument("--oasyced", default=str(DEFAULT_BINARY), help="Path to oasyced binary")
    dry_run.add_argument("--home", default="", help="Existing oasyced home for current-network dry-runs")
    dry_run.add_argument("--chain-id", default=DEFAULT_CHAIN_ID, help="Chain ID")
    dry_run.add_argument("--keyring-backend", default=DEFAULT_KEYRING, help="Keyring backend")
    dry_run.add_argument("--rpc", default="http://127.0.0.1:26657", help="RPC URL for current-network dry-runs")
    dry_run.add_argument("--fees", default=DEFAULT_FEES, help="Tx fees for dry-run command")
    dry_run.add_argument("--json", action="store_true", help="Emit JSON only")

    return parser


def render_command(args: argparse.Namespace) -> dict[str, Any]:
    return render_artifacts(
        height=args.height,
        title=args.title,
        summary=args.summary,
        deposit=args.deposit,
        forum_url=args.proposal_forum_url,
        metadata_ref=args.metadata_ref,
        proposal_template=Path(args.proposal_template),
        metadata_template=Path(args.metadata_template),
        proposal_output=Path(args.proposal_output) if args.proposal_output else None,
        metadata_output=Path(args.metadata_output) if args.metadata_output else None,
    )


def dry_run_command(args: argparse.Namespace) -> dict[str, Any]:
    rendered = render_artifacts(
        height=args.height,
        title=args.title,
        summary=args.summary,
        deposit=args.deposit,
        forum_url=args.proposal_forum_url,
        metadata_ref=args.metadata_ref,
        proposal_template=Path(args.proposal_template),
        metadata_template=Path(args.metadata_template),
        proposal_output=Path(args.proposal_output) if args.proposal_output else None,
        metadata_output=Path(args.metadata_output) if args.metadata_output else None,
    )

    proposal_path = Path(rendered["proposal_path"])
    validation = validate_file(proposal_path)
    if validation["status"] != "ok":
        return {
            "status": "error",
            "rendered": rendered,
            "validation": validation,
            "error": "rendered proposal failed validation",
        }

    binary = Path(args.oasyced)
    if args.network == "current":
        if not args.home:
            return {
                "status": "error",
                "rendered": rendered,
                "validation": validation,
                "error": "--home is required for --network current",
            }
        dry_run = run_gov_dry_run(
            proposal_path,
            binary=binary,
            home=args.home,
            chain_id=args.chain_id,
            keyring_backend=args.keyring_backend,
            rpc_url=args.rpc,
            fees=args.fees,
        )
        return {
            "status": dry_run["status"],
            "network": "current",
            "rendered": rendered,
            "validation": validation,
            "dry_run": dry_run,
        }

    build = live_gate.build_binary(binary)
    if build.get("status") != "ok":
        return {"status": "error", "rendered": rendered, "validation": validation, "build": build, "error": "build step failed"}

    temp_ctx = tempfile.TemporaryDirectory(prefix="oasyce-upgrade-v080-")
    temp_root = Path(temp_ctx.name)
    home = temp_root / "home"
    log_dir = temp_root / "logs"
    log_dir.mkdir(parents=True, exist_ok=True)
    node_log = log_dir / "node.log"
    ports = {
        "rpc": live_gate.find_free_port(),
        "rest": live_gate.find_free_port(),
        "grpc": live_gate.find_free_port(),
        "p2p": live_gate.find_free_port(),
    }
    rpc_url = f"http://127.0.0.1:{ports['rpc']}"
    rest_url = f"http://127.0.0.1:{ports['rest']}"
    process = None
    try:
        localnet = live_gate.init_tempnet(binary, home, args.chain_id, args.keyring_backend, ports)
        if localnet.get("status") != "ok":
            return {
                "status": "error",
                "rendered": rendered,
                "validation": validation,
                "build": build,
                "localnet": localnet,
                "error": "tempnet init failed",
            }
        process = live_gate.start_tempnet(binary, home, node_log)
        localnet_ready = live_gate.wait_for_localnet(rpc_url, rest_url)
        localnet_ready["home"] = str(home)
        localnet_ready["log"] = str(node_log)
        dry_run = run_gov_dry_run(
            proposal_path,
            binary=binary,
            home=str(home),
            chain_id=args.chain_id,
            keyring_backend=args.keyring_backend,
            rpc_url=rpc_url,
            fees=args.fees,
        )
        return {
            "status": dry_run["status"],
            "network": "tempnet",
            "rendered": rendered,
            "validation": validation,
            "build": build,
            "localnet": localnet_ready,
            "dry_run": dry_run,
        }
    finally:
        if process is not None:
            live_gate.stop_process(process)
        temp_ctx.cleanup()


def emit(result: dict[str, Any], *, json_only: bool) -> int:
    if json_only:
        print(json.dumps(result, indent=2))
    else:
        print(json.dumps(result, indent=2))
        if result.get("status") == "ok":
            if "rendered" in result:
                info(f"Proposal: {result['rendered']['proposal_path']}")
                info(f"Metadata: {result['rendered']['metadata_path']}")
                info(f"Submit: {result['rendered']['submit_command']}")
            elif "proposal_path" in result:
                info(f"Proposal: {result['proposal_path']}")
    return 0 if result.get("status") == "ok" else 1


def main(argv: list[str] | None = None) -> int:
    parser = proposal_parser()
    args = parser.parse_args(argv)

    if args.command == "render":
        result = render_command(args)
        return emit(result, json_only=args.json)

    if args.command == "validate":
        result = validate_file(Path(args.proposal))
        return emit(result, json_only=args.json)

    if args.command == "dry-run":
        result = dry_run_command(args)
        return emit(result, json_only=args.json)

    raise AssertionError(f"unhandled command: {args.command}")


if __name__ == "__main__":
    raise SystemExit(main())
