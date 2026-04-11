#!/usr/bin/env python3
"""Inspect the adjacent oasyce-sdk surface expected by chain-side wrappers."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

from _sdk_compat import inspect_sdk_surface


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_SDK_PATH = ROOT.parent / "oasyce-sdk"


def render_text(surface: dict) -> str:
    lines = [
        f"status: {surface['status']}",
        f"requested_mode: {surface['requested_mode']}",
        f"tested_baseline: {surface['tested_baseline']}",
        f"module_path: {surface['module_path']}",
        f"source_checkout: {surface['source_checkout'] or '(none)'}",
        f"source_checkout_version: {surface['source_checkout_version'] or '(unknown)'}",
        f"package_version: {surface['package_version'] or '(unknown)'}",
        f"distribution_version: {surface['distribution_version'] or '(unknown)'}",
        f"resolved_from_source: {surface['resolved_from_source']}",
        f"identity_seam: {surface['identity_seam']}",
        f"scanner_module: {surface['scanner_module'] or '(missing)'}",
        f"scanner_scan: {surface['scanner_scan']}",
    ]
    lines.append("signer_methods:")
    for name, present in sorted(surface["signer_methods"].items()):
        lines.append(f"  - {name}: {present}")

    pulse = surface["pulse"]
    lines.extend(
        [
            "pulse_surface:",
            f"  - helper_names: {', '.join(pulse['helper_names']) if pulse['helper_names'] else '(none)'}",
            f"  - schema_present: {pulse['schema_present']}",
            f"  - schema_has_dimensions: {pulse['schema_has_dimensions']}",
            f"  - schema_field_count: {pulse['schema_field_count']}",
        ]
    )
    if pulse.get("schema_error"):
        lines.append(f"  - schema_error: {pulse['schema_error']}")

    if surface["warnings"]:
        lines.append("warnings:")
        for item in surface["warnings"]:
            lines.append(f"  - {item}")
    if surface["errors"]:
        lines.append("errors:")
        for item in surface["errors"]:
            lines.append(f"  - {item}")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--mode",
        default="source",
        choices=("source", "installed", "auto"),
        help="How chain-side checks should resolve oasyce-sdk",
    )
    parser.add_argument(
        "--sdk-path",
        default=str(DEFAULT_SDK_PATH),
        help="Preferred oasyce-sdk checkout path for source/auto modes",
    )
    parser.add_argument("--json", action="store_true", help="Emit JSON instead of human-readable text")
    args = parser.parse_args()

    os.environ["OASYCE_SDK_MODE"] = args.mode
    os.environ["OASYCE_SDK_PATH"] = args.sdk_path

    surface = inspect_sdk_surface(args.mode)
    if args.json:
        print(json.dumps(surface, indent=2))
    else:
        print(render_text(surface))
    return 1 if surface["status"] == "error" else 0


if __name__ == "__main__":
    raise SystemExit(main())
