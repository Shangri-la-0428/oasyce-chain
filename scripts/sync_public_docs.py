#!/usr/bin/env python3
"""Sync chain-side public beta entrypoints from a single contract file."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CONTRACT_PATH = ROOT / "docs" / "public_beta_contract.json"

README_ZH = ROOT / "README.md"
README_EN = ROOT / "README_EN.md"
INDEX_HTML = ROOT / "website" / "index.html"
DOCS_HTML = ROOT / "website" / "docs.html"


def read_contract() -> dict:
    return json.loads(CONTRACT_PATH.read_text())


def replace_block(text: str, block_name: str, content: str) -> str:
    begin = f"<!-- BEGIN GENERATED:{block_name} -->"
    end = f"<!-- END GENERATED:{block_name} -->"
    pattern = re.compile(rf"{re.escape(begin)}.*?{re.escape(end)}", re.DOTALL)
    replacement = f"{begin}\n{content}\n{end}"
    if not pattern.search(text):
        raise ValueError(f"Cannot find generated block {block_name}")
    return pattern.sub(replacement, text, count=1)


def render_readme(lang: str) -> str:
    data = read_contract()["readme"][lang]
    lines = [data["title"], ""]
    lines.extend(data["intro_lines"])
    lines.append("")
    if lang == "zh":
        lines.extend(["| 项目 | 值 |", "|------|-----|"])
        lines.extend(f"| {label} | {value} |" for label, value in data["table_rows"])
    else:
        lines.extend(data["bullet_lines"])
    return "\n".join(lines)


def render_index_testnet() -> str:
    data = read_contract()["website"]["testnet_section"]
    rows = "\n".join(f"        <tr><td>{label}</td><td>{value}</td></tr>" for label, value in data["rows"])
    buttons = "\n".join(f"      {button}" for button in data["buttons"])
    return (
        '<section class="reveal" id="testnet">\n'
        '  <div class="container">\n'
        '    <h2 data-i18n="testnet-title"><span class="live-dot" aria-label="Live"></span>Public Testnet</h2>\n'
        f'    <p style="margin-bottom: var(--space-lg);" data-i18n="testnet-desc">{data["description"]}</p>\n'
        '    <table class="comparison" style="max-width: 640px;">\n'
        '      <tbody>\n'
        f"{rows}\n"
        '      </tbody>\n'
        '    </table>\n'
        '    <div style="margin-top: var(--space-lg); display: flex; gap: var(--space-md); flex-wrap: wrap;">\n'
        f"{buttons}\n"
        '    </div>\n'
        '  </div>\n'
        '</section>'
    )


def render_docs_start_here() -> str:
    data = read_contract()["website"]["docs_start_here"]
    paragraphs = "\n".join(f"<p>{paragraph}</p>" for paragraph in data["paragraphs"])
    links = "\n".join(f"  {link}" for link in data["links"])
    return (
        '<section id="start-here">\n'
        '<h2>Start Here</h2>\n'
        '<div class="notice">\n'
        f"{paragraphs}\n"
        '<div class="notice-links">\n'
        f"{links}\n"
        '</div>\n'
        '</div>\n'
        '</section>'
    )


def sync_file(path: Path, block_name: str, content: str, *, write: bool) -> bool:
    current = path.read_text()
    updated = replace_block(current, block_name, content)
    if current == updated:
        return False
    if write:
        path.write_text(updated)
    else:
        raise RuntimeError(f"STALE: {path} block {block_name} is out of sync")
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description="Sync chain public beta entrypoints")
    parser.add_argument("--write", action="store_true", help="Write files instead of check-only mode")
    args = parser.parse_args()

    try:
        changed = []
        if sync_file(README_ZH, "PUBLIC_BETA_ZH", render_readme("zh"), write=args.write):
            changed.append(str(README_ZH))
        if sync_file(README_EN, "PUBLIC_BETA_EN", render_readme("en"), write=args.write):
            changed.append(str(README_EN))
        if sync_file(INDEX_HTML, "WEBSITE_TESTNET", render_index_testnet(), write=args.write):
            changed.append(str(INDEX_HTML))
        if sync_file(DOCS_HTML, "WEBSITE_DOCS_START", render_docs_start_here(), write=args.write):
            changed.append(str(DOCS_HTML))
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        print("Run 'python3 scripts/sync_public_docs.py --write' to update generated blocks.")
        return 1

    if changed:
        print("\n".join(f"UPDATED: {path}" for path in changed))
    else:
        print("Public docs sync OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
