#!/usr/bin/env python3
"""Fail when chain-owned source files drift across frozen stack boundaries."""

from __future__ import annotations

import argparse
import sys
from dataclasses import dataclass
from pathlib import Path

BLACKLIST = (
    "device_join",
    "share_session",
    "presence_ping",
    "emotion_state",
    "dashboard_route",
    "handoff_artifact",
)
ALLOW_TOKEN = "stack-boundary: allow"
TARGET_DIRS = ("app", "cmd", "x")
TARGET_SUFFIXES = (".go", ".proto")


@dataclass(frozen=True)
class Finding:
    path: str
    line: int
    token: str
    text: str


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def iter_target_files(root: Path):
    for directory in TARGET_DIRS:
        base = root / directory
        if not base.exists():
            continue
        for path in base.rglob("*"):
            if path.is_file() and path.suffix in TARGET_SUFFIXES:
                yield path


def scan_text(path: Path, text: str) -> list[Finding]:
    findings: list[Finding] = []
    for lineno, raw_line in enumerate(text.splitlines(), start=1):
        if ALLOW_TOKEN in raw_line:
            continue
        for token in BLACKLIST:
            if token in raw_line:
                findings.append(
                    Finding(
                        path=str(path),
                        line=lineno,
                        token=token,
                        text=raw_line.strip(),
                    )
                )
    return findings


def scan_repo(root: Path) -> list[Finding]:
    findings: list[Finding] = []
    for path in iter_target_files(root):
        findings.extend(scan_text(path, path.read_text(encoding="utf-8")))
    return findings


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=repo_root(), help="repository root to scan")
    args = parser.parse_args(argv)

    root = args.root.resolve()
    findings = scan_repo(root)
    if not findings:
        print(f"stack boundary OK ({root})")
        return 0

    print(f"stack boundary violations ({root}):", file=sys.stderr)
    for finding in findings:
        print(
            f"  {Path(finding.path).relative_to(root)}:{finding.line}: "
            f"token={finding.token} text={finding.text}",
            file=sys.stderr,
        )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
