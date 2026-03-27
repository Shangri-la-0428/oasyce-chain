#!/usr/bin/env python3
"""Cross-project consistency checker.

Parses proto files (source of truth) and verifies that downstream Python
projects (oasyce-sdk, oasyce-net) have matching field definitions.

Why this exists:
  When a new proto field is added to the chain (e.g. DataAsset.service_url),
  downstream SDK/CLI dataclasses must be updated too. Without automated
  checking, these fall out of sync silently — buyers see empty fields,
  CLI commands miss flags, and nobody notices until a user reports it.

Usage:
    python3 scripts/check_cross_project_sync.py

Exit code 0 = all in sync, 1 = drift detected.
"""

import ast
import os
import re
import sys
from pathlib import Path
from typing import Dict, List, Set, Tuple

# ---------------------------------------------------------------------------
# Config: proto → Python class mappings
# ---------------------------------------------------------------------------

# Base directories (relative to this script's parent's parent)
CHAIN_DIR = Path(__file__).resolve().parent.parent
NET_DIR = CHAIN_DIR.parent / "oasyce-net"
SDK_DIR = CHAIN_DIR.parent / "oasyce-sdk"

# Proto files → message → downstream Python files + class names
# Each entry: (proto_file, message_name, [(python_file, class_name), ...])
SYNC_MAP: List[Tuple[str, str, List[Tuple[Path, str]]]] = [
    (
        "proto/oasyce/datarights/v1/types.proto",
        "DataAsset",
        [
            (SDK_DIR / "oasyce_sdk" / "types.py", "DataAsset"),
            (NET_DIR / "oasyce" / "proto" / "oasyce" / "datarights" / "v1" / "__init__.py", "DataAsset"),
        ],
    ),
    (
        "proto/oasyce/datarights/v1/tx.proto",
        "MsgRegisterDataAsset",
        [
            (NET_DIR / "oasyce" / "proto" / "oasyce" / "datarights" / "v1" / "__init__.py", "MsgRegisterDataAsset"),
        ],
    ),
]

# Fields that exist in proto but are intentionally omitted from Python
# (e.g. complex types that need special handling)
ALLOWED_MISSING: Dict[str, Set[str]] = {
    "DataAsset": {
        "co_creators",      # complex nested type, parsed separately
        "created_at",       # timestamp, not surfaced in simple dataclass
        "shutdown_initiated_at",  # timestamp
        "migration_enabled",      # internal lifecycle flag
    },
    "MsgRegisterDataAsset": {
        "co_creators",  # complex nested type
        "parent_asset_id",  # versioning, handled separately
    },
}


def parse_proto_fields(proto_path: Path, message_name: str) -> Set[str]:
    """Extract field names from a proto message definition."""
    if not proto_path.exists():
        print(f"  SKIP: {proto_path} not found")
        return set()

    text = proto_path.read_text()
    # Find the message block
    pattern = rf"message\s+{message_name}\s*\{{(.*?)\}}"
    match = re.search(pattern, text, re.DOTALL)
    if not match:
        print(f"  SKIP: message {message_name} not found in {proto_path}")
        return set()

    body = match.group(1)
    fields = set()
    # Match lines like: string field_name = N;
    # or: repeated string field_name = N;
    # or: RightsType field_name = N;
    for line in body.split("\n"):
        line = line.strip()
        # Skip options, comments, empty lines
        if not line or line.startswith("//") or line.startswith("option"):
            continue
        # Match field definition: [repeated] type name = number [options];
        field_match = re.match(
            r"(?:repeated\s+)?\w+(?:\.\w+)*\s+(\w+)\s*=\s*\d+", line
        )
        if field_match:
            fields.add(field_match.group(1))

    return fields


def parse_python_class_fields(py_path: Path, class_name: str) -> Set[str]:
    """Extract field names from a Python dataclass."""
    if not py_path.exists():
        print(f"  SKIP: {py_path} not found")
        return set()

    text = py_path.read_text()
    try:
        tree = ast.parse(text)
    except SyntaxError:
        print(f"  SKIP: {py_path} has syntax errors")
        return set()

    fields = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef) and node.name == class_name:
            for item in node.body:
                if isinstance(item, ast.AnnAssign) and isinstance(item.target, ast.Name):
                    fields.add(item.target.id)
                elif isinstance(item, ast.Assign):
                    for target in item.targets:
                        if isinstance(target, ast.Name):
                            fields.add(target.id)
            break

    return fields


def check_sync() -> int:
    """Check all proto→Python mappings. Returns number of drift issues."""
    issues = 0

    for proto_rel, msg_name, py_targets in SYNC_MAP:
        proto_path = CHAIN_DIR / proto_rel
        print(f"\n{'='*60}")
        print(f"Proto: {msg_name} ({proto_rel})")

        proto_fields = parse_proto_fields(proto_path, msg_name)
        if not proto_fields:
            continue

        allowed = ALLOWED_MISSING.get(msg_name, set())

        for py_path, py_class in py_targets:
            py_fields = parse_python_class_fields(py_path, py_class)
            if not py_fields:
                continue

            # Check for fields in proto but missing in Python
            # Map proto snake_case to Python: id→asset_id (special case for DataAsset)
            proto_check = proto_fields - allowed

            # Build a mapping of proto field → expected Python field
            missing = []
            for pf in sorted(proto_check):
                # Check if the field exists in Python (exact or with common renames)
                candidates = {pf}
                if pf == "id":
                    candidates.add("asset_id")  # common rename
                if not candidates & py_fields:
                    missing.append(pf)

            rel_py = py_path.relative_to(py_path.parents[3]) if len(py_path.parents) > 3 else py_path.name
            if missing:
                print(f"  DRIFT in {rel_py}::{py_class}")
                for f in missing:
                    print(f"    - missing: {f}")
                issues += len(missing)
            else:
                print(f"  OK    {rel_py}::{py_class} ({len(py_fields)} fields)")

    return issues


def main():
    print("Cross-project proto↔Python sync check")
    print(f"Chain: {CHAIN_DIR}")
    print(f"SDK:   {SDK_DIR}")
    print(f"Net:   {NET_DIR}")

    issues = check_sync()

    print(f"\n{'='*60}")
    if issues:
        print(f"FAIL: {issues} field(s) out of sync")
        print("\nTo fix: update the Python dataclass to include the missing proto fields.")
        print("If a field is intentionally omitted, add it to ALLOWED_MISSING in this script.")
        return 1
    else:
        print("PASS: all downstream projects in sync with proto definitions")
        return 0


if __name__ == "__main__":
    sys.exit(main())
