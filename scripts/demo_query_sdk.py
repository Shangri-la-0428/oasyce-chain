#!/usr/bin/env python3
"""
Oasyce Query SDK Demo

Demonstrates querying the Oasyce chain REST API for:
- AI capabilities marketplace
- Data asset bonding curves
- Escrow state transitions
- Reputation scores
- Access level gating

Usage:
    python3 scripts/demo_query_sdk.py [--base-url http://localhost:1317]
    python3 scripts/demo_query_sdk.py --base-url http://47.93.32.88:1317  # testnet
"""

import argparse
import json
import sys
from urllib.request import urlopen, Request
from urllib.error import URLError


def fetch(url: str) -> dict:
    """Fetch JSON from a URL."""
    try:
        req = Request(url, headers={"Accept": "application/json"})
        with urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())
    except URLError as e:
        print(f"  Error: Cannot reach {url} — {e}")
        return {}
    except json.JSONDecodeError:
        print(f"  Error: Invalid JSON from {url}")
        return {}


def section(title: str):
    print(f"\n{'═' * 60}")
    print(f"  {title}")
    print(f"{'═' * 60}\n")


def demo_capabilities(base: str):
    """Query AI capability marketplace."""
    section("AI Capability Marketplace")

    data = fetch(f"{base}/oasyce/capability/v1/capabilities")
    caps = data.get("capabilities", [])
    print(f"  Total capabilities registered: {len(caps)}\n")

    for cap in caps[:5]:
        print(f"  [{cap.get('id', 'N/A')}] {cap.get('name', 'Unnamed')}")
        print(f"    Provider:    {cap.get('provider', 'N/A')}")
        price = cap.get("price_per_call", {})
        print(f"    Price/Call:  {price.get('amount', '0')} {price.get('denom', 'uoas')}")
        print(f"    Active:      {cap.get('is_active', False)}")
        print(f"    Total Calls: {cap.get('total_calls', 0)}")
        print(f"    Success Rate: {int(cap.get('success_rate', 0)) / 100:.1f}%")
        tags = cap.get("tags", [])
        if tags:
            print(f"    Tags:        {', '.join(tags)}")
        print()


def demo_data_assets(base: str):
    """Query data assets and bonding curve state."""
    section("Data Assets (Bonding Curve)")

    data = fetch(f"{base}/oasyce/datarights/v1/data_assets")
    assets = data.get("data_assets", [])
    print(f"  Total data assets: {len(assets)}\n")

    for asset in assets[:5]:
        print(f"  [{asset.get('id', 'N/A')}] {asset.get('name', 'Unnamed')}")
        print(f"    Owner:         {asset.get('owner', 'N/A')}")
        print(f"    Total Shares:  {asset.get('total_shares', '0')}")
        reserve = asset.get("reserve_balance", {})
        print(f"    Reserve:       {reserve.get('amount', '0')} {reserve.get('denom', 'uoas')}")
        status = asset.get("status", "ACTIVE")
        print(f"    Status:        {status}")
        tags = asset.get("tags", [])
        if tags:
            print(f"    Tags:          {', '.join(tags)}")
        print()


def demo_access_level(base: str, asset_id: str, address: str):
    """Query access level for an address on an asset."""
    section(f"Access Level: {asset_id}")

    data = fetch(f"{base}/oasyce/datarights/v1/access_level/{asset_id}/{address}")
    if not data:
        print("  No data returned.")
        return

    print(f"  Address:      {address}")
    print(f"  Shares:       {data.get('shares', '0')}")
    print(f"  Total Shares: {data.get('total_shares', '0')}")
    print(f"  Equity:       {int(data.get('equity_bps', 0)) / 100:.2f}%")
    level = data.get("access_level", "")
    level_desc = {
        "L0": "Metadata only",
        "L1": "Preview/sample",
        "L2": "Full read access",
        "L3": "Full data delivery",
    }
    print(f"  Access Level: {level or 'None'} — {level_desc.get(level, 'Insufficient equity')}")


def demo_reputation(base: str):
    """Query reputation leaderboard."""
    section("Reputation Leaderboard")

    data = fetch(f"{base}/oasyce/reputation/v1/leaderboard")
    entries = data.get("entries", [])
    print(f"  Total scored addresses: {len(entries)}\n")

    for entry in entries[:10]:
        addr = entry.get("address", "N/A")
        score = entry.get("score", "0")
        print(f"  {addr[:20]}...  Score: {score}")


def demo_settlement_params(base: str):
    """Query settlement parameters."""
    section("Settlement Parameters")

    data = fetch(f"{base}/oasyce/settlement/v1/params")
    params = data.get("params", {})
    if params:
        print(f"  Protocol Fee Rate: {int(params.get('protocol_fee_rate', 0)) / 100:.1f}%")
        print(f"  Treasury Rate:     {int(params.get('treasury_rate', 0)) / 100:.1f}%")
        print(f"  Burn Rate:         2% (hardcoded)")
        print(f"  Provider Share:    90% (remainder)")
    else:
        print("  No params returned.")


def demo_chain_status(base: str):
    """Query basic chain status."""
    section("Chain Status")

    data = fetch(f"{base}/cosmos/base/tendermint/v1beta1/blocks/latest")
    block = data.get("block", {}).get("header", {})
    if block:
        print(f"  Chain ID: {block.get('chain_id', 'N/A')}")
        print(f"  Height:   {block.get('height', 'N/A')}")
        print(f"  Time:     {block.get('time', 'N/A')}")
    else:
        print("  Could not fetch latest block.")


def main():
    parser = argparse.ArgumentParser(description="Oasyce Query SDK Demo")
    parser.add_argument("--base-url", default="http://localhost:1317",
                        help="Base URL for the REST API (default: http://localhost:1317)")
    parser.add_argument("--asset-id", default="", help="Asset ID for access level query")
    parser.add_argument("--address", default="", help="Address for access level query")
    args = parser.parse_args()

    base = args.base_url.rstrip("/")

    print("╔════════════════════════════════════════════════════════════╗")
    print("║           Oasyce Chain — Query SDK Demo                   ║")
    print("║  Property, Contracts, and Arbitration for Agent Economy   ║")
    print("╚════════════════════════════════════════════════════════════╝")
    print(f"\n  API: {base}")

    demo_chain_status(base)
    demo_capabilities(base)
    demo_data_assets(base)
    demo_settlement_params(base)
    demo_reputation(base)

    if args.asset_id and args.address:
        demo_access_level(base, args.asset_id, args.address)

    section("DONE")
    print("  All queries use standard REST endpoints.")
    print("  For gRPC (high performance): localhost:9090")
    print("  For CLI:  oasyced query <module> <subcommand> --output json")


if __name__ == "__main__":
    main()
