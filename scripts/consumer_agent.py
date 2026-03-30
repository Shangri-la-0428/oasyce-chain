#!/usr/bin/env python3
"""
Oasyce Consumer Agent — autonomous capability consumer for testnet.

Runs on cron (every 30 min). Each cycle:
  1. Check balance, request faucet if low
  2. Discover active capabilities
  3. Invoke one capability on-chain
  4. POST input to provider endpoint
  5. Submit reputation feedback on result

USAGE
=====
    # One-shot cycle:
    python3 consumer_agent.py

    # Cron (every 30 min):
    */30 * * * * /usr/bin/python3 /opt/oasyce/src/scripts/consumer_agent.py >> /var/log/oasyce-consumer.log 2>&1

ENVIRONMENT
===========
    CONSUMER_KEY        — keyring key name (default: "consumer")
    OASYCE_CHAIN_REST   — chain REST (default: "http://127.0.0.1:11317")
    OASYCE_CHAIN_RPC    — chain RPC  (default: "http://127.0.0.1:26667")
    OASYCED_CHAIN_ID    — chain ID (default: "oasyce-testnet-1")
    OASYCED_KEYRING     — keyring backend (default: "test")
    OASYCE_HOME         — oasyced home (default: "/home/oasyce/.oasyced")
    PROVIDER_ENDPOINT   — provider agent URL (default: "http://127.0.0.1:8430")
    FAUCET_URL          — faucet URL (default: "http://127.0.0.1:18080")
    MIN_BALANCE_UOAS    — min balance before faucet (default: 5000000 = 5 OAS)
"""

import hashlib
import json
import logging
import os
import re
import subprocess
import sys
import time
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

CONSUMER_KEY = os.environ.get("CONSUMER_KEY", "consumer")
CHAIN_REST = os.environ.get("OASYCE_CHAIN_REST", "http://127.0.0.1:11317").rstrip("/")
CHAIN_RPC = os.environ.get("OASYCE_CHAIN_RPC", "http://127.0.0.1:26667").rstrip("/")
CHAIN_ID = os.environ.get("OASYCED_CHAIN_ID", "oasyce-testnet-1")
KEYRING = os.environ.get("OASYCED_KEYRING", "test")
HOME = os.environ.get("OASYCE_HOME", "/home/oasyce/.oasyced")
OASYCED = os.environ.get("OASYCED_BIN", "oasyced")
PROVIDER_ENDPOINT = os.environ.get("PROVIDER_ENDPOINT", "http://127.0.0.1:8430").rstrip("/")
FAUCET_URL = os.environ.get("FAUCET_URL", "http://127.0.0.1:18080").rstrip("/")
MIN_BALANCE_UOAS = int(os.environ.get("MIN_BALANCE_UOAS", "5000000"))
TX_FEE_UOAS = int(os.environ.get("TX_FEE_UOAS", "10000"))

STATE_FILE = os.environ.get("CONSUMER_STATE_FILE", "/tmp/consumer_agent_state.json")
STATE_DEFAULTS = {
    "total_invocations": 0,
    "total_settlements": 0,
    "total_data_purchases": 0,
    "last_run": "",
    "last_success": "",
    "last_status": "never_run",
    "last_error": "",
}

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [consumer] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger()

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def rest_get(path):
    url = f"{CHAIN_REST}{path}"
    try:
        with urlopen(Request(url), timeout=10) as resp:
            return json.loads(resp.read().decode())
    except (URLError, HTTPError, json.JSONDecodeError) as e:
        log.error("GET %s failed: %s", url, e)
        return None


def http_get(url):
    try:
        with urlopen(Request(url), timeout=10) as resp:
            return json.loads(resp.read().decode())
    except (URLError, HTTPError, json.JSONDecodeError) as e:
        log.error("GET %s failed: %s", url, e)
        return None


def http_post(url, data):
    body = json.dumps(data).encode()
    req = Request(url, data=body, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urlopen(req, timeout=120) as resp:
            return json.loads(resp.read().decode()), None
    except HTTPError as e:
        err = e.read().decode("utf-8", errors="replace")
        return None, f"HTTP {e.code}: {err[:300]}"
    except (URLError, json.JSONDecodeError) as e:
        return None, str(e)


def get_address():
    try:
        r = subprocess.run(
            [OASYCED, "keys", "show", CONSUMER_KEY, "-a",
             "--keyring-backend", KEYRING, "--home", HOME],
            capture_output=True, text=True, timeout=10,
        )
        if r.returncode == 0:
            return r.stdout.strip()
    except (subprocess.TimeoutExpired, FileNotFoundError):
        pass
    return None


def oasyced_tx(args, retries=2):
    cmd = [OASYCED, "tx"] + args + [
        "--from", CONSUMER_KEY,
        "--keyring-backend", KEYRING,
        "--chain-id", CHAIN_ID,
        "--home", HOME,
        "--fees", "10000uoas",
        "--gas", "200000",
        "--yes",
        "--output", "json",
    ]
    log.info("TX: %s", " ".join(cmd[2:6]))
    for attempt in range(1, retries + 1):
        try:
            r = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
            out = r.stdout.strip()
            err = r.stderr.strip()
            if r.returncode != 0:
                log.error("TX failed (rc=%d): stdout=%s stderr=%s", r.returncode, out[:200], err[:200])
                return False, out or err
            if not out:
                out = err
            try:
                d = json.loads(out)
                code = d.get("code", 0)
                if code == 0:
                    log.info("TX hash: %s", d.get("txhash", "?"))
                    return True, d.get("txhash", "")
                # code 19 = account sequence mismatch — retry after a block
                if code == 19 and attempt < retries:
                    log.warning("Sequence mismatch (attempt %d/%d), waiting for next block...", attempt, retries)
                    time.sleep(6)
                    continue
                log.error("TX CheckTx error (code=%s): %s", code, d.get("raw_log", ""))
                return False, d.get("raw_log", out)
            except json.JSONDecodeError:
                return True, out
        except subprocess.TimeoutExpired:
            return False, "timeout"
    return False, "max retries exceeded"


def load_state():
    try:
        with open(STATE_FILE) as f:
            data = json.load(f)
    except (FileNotFoundError, json.JSONDecodeError):
        data = {}
    merged = dict(STATE_DEFAULTS)
    if isinstance(data, dict):
        merged.update(data)
    return merged


def save_state(state):
    try:
        with open(STATE_FILE, "w") as f:
            json.dump(state, f)
    except PermissionError:
        # State file owned by different user — recreate
        try:
            os.remove(STATE_FILE)
            with open(STATE_FILE, "w") as f:
                json.dump(state, f)
        except OSError as e:
            log.warning("Cannot save state: %s", e)


def finish_cycle(state, status, exit_code=0, error="", success=False):
    now = time.strftime("%Y-%m-%d %H:%M:%S")
    state["last_run"] = now
    state["last_status"] = status
    state["last_error"] = error
    if success:
        state["last_success"] = now
    save_state(state)
    if error:
        log.info("Cycle finished: status=%s error=%s", status, error)
    else:
        log.info("Cycle finished: status=%s", status)
    return exit_code


def parse_uoas(value):
    if isinstance(value, int):
        return value
    if isinstance(value, str):
        text = value.strip().lower()
        if text.isdigit():
            return int(text)
        if text.endswith("uoas"):
            amount = text[:-4].strip()
            return int(amount) if amount.isdigit() else 0
        match = re.match(r"^([0-9]+(?:\.[0-9]+)?)\s*oas$", text)
        if match:
            return int(float(match.group(1)) * 1_000_000)
    return 0


def wait_for_balance_increase(addr, before_balance, timeout_seconds=20, poll_seconds=4):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        time.sleep(poll_seconds)
        current = check_balance(addr)
        if current > before_balance:
            return current
    return None


# ---------------------------------------------------------------------------
# Steps
# ---------------------------------------------------------------------------

def ensure_consumer_key(addr):
    """Create consumer key if it doesn't exist."""
    if addr:
        return addr
    log.info("Consumer key not found, creating...")
    r = subprocess.run(
        [OASYCED, "keys", "add", CONSUMER_KEY,
         "--keyring-backend", KEYRING, "--home", HOME, "--output", "json"],
        capture_output=True, text=True, timeout=10,
    )
    if r.returncode == 0:
        addr = get_address()
        log.info("Created consumer key: %s", addr)
        return addr
    log.error("Failed to create key: %s", r.stderr[:200])
    return None


def check_balance(addr):
    data = rest_get(f"/cosmos/bank/v1beta1/balances/{addr}")
    if data:
        for b in data.get("balances", []):
            if b["denom"] == "uoas":
                return int(b["amount"])
    return 0


def request_faucet(addr):
    log.info("Requesting faucet for %s...", addr)
    url = f"{FAUCET_URL}/faucet?address={addr}"
    try:
        with urlopen(Request(url), timeout=10) as resp:
            data = json.loads(resp.read().decode())
    except HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        message = f"HTTP {e.code}: {body[:300]}"
        log.warning("Faucet request failed: %s", message)
        return False, 0, message
    except (URLError, json.JSONDecodeError) as e:
        message = str(e)
        log.warning("Faucet request failed: %s", message)
        return False, 0, message

    if data.get("status") == "ok":
        amount_uoas = parse_uoas(str(data.get("amount", "0")))
        log.info("Faucet: %s", data.get("amount", "?"))
        return True, amount_uoas, ""

    message = str(data)
    log.warning("Faucet response: %s", message)
    return False, 0, message


def fetch_provider_health(probe=False):
    url = f"{PROVIDER_ENDPOINT}/health"
    if probe:
        url += "?probe=1"
    try:
        with urlopen(Request(url), timeout=10) as resp:
            return json.loads(resp.read().decode()), ""
    except HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")[:200]
        return None, f"HTTP {e.code}: {body}"
    except (URLError, json.JSONDecodeError) as e:
        return None, str(e)


def get_capability(cap_id):
    data = rest_get(f"/oasyce/capability/v1/capability/{cap_id}")
    if not data:
        return None
    cap = data.get("capability")
    return cap if isinstance(cap, dict) else None


def is_active_capability(cap):
    return bool(cap and cap.get("is_active") and cap.get("id", "").startswith("CAP_"))


def discover_capability(preferred_capability_id=""):
    """Find the best active capability — prefer Claude AI."""
    preferred_capability_id = preferred_capability_id.strip()
    if preferred_capability_id:
        cap = get_capability(preferred_capability_id)
        if is_active_capability(cap):
            return cap
        log.warning("Preferred capability %s is unavailable or inactive", preferred_capability_id)
        return None

    data = rest_get("/oasyce/capability/v1/capabilities")
    if not data:
        return None
    active = [c for c in data.get("capabilities", [])
              if c.get("is_active") and c.get("id", "").startswith("CAP_")]
    if not active:
        return None
    active.sort(key=lambda c: c.get("created_at", ""), reverse=True)
    # Prefer Claude AI capability
    for cap in active:
        if "claude" in cap.get("name", "").lower():
            return cap
    return active[0]


def invoke_on_chain(cap_id, input_data):
    input_json = json.dumps(input_data)
    ok, txhash = oasyced_tx([
        "oasyce_capability", "invoke", cap_id,
        "--input", input_json,
    ])
    if not ok:
        return None
    # Wait for TX inclusion
    time.sleep(7)
    # Find the invocation — query by listing and finding latest
    return txhash


def find_invocation_from_tx(txhash):
    """Extract invocation ID from TX events via RPC."""
    url = f"{CHAIN_RPC}/tx?hash=0x{txhash}"
    try:
        with urlopen(Request(url), timeout=10) as resp:
            data = json.loads(resp.read().decode())
    except (URLError, HTTPError, json.JSONDecodeError) as e:
        log.error("RPC tx query failed: %s", e)
        return None

    tx_result = data.get("result", {}).get("tx_result", {})
    if tx_result.get("code", -1) != 0:
        log.error("TX failed on-chain: code=%s log=%s",
                  tx_result.get("code"), tx_result.get("log", "")[:200])
        return None

    for event in tx_result.get("events", []):
        if event.get("type") == "capability_invoked":
            for attr in event.get("attributes", []):
                if attr.get("key") == "invocation_id":
                    return attr.get("value")
    return None


def post_to_provider(invocation_id, input_data):
    url = f"{PROVIDER_ENDPOINT}/api/v1/process"
    log.info("POST %s invocation=%s", url, invocation_id)
    resp, err = http_post(url, {
        "invocation_id": invocation_id,
        "input": input_data,
    })
    if err:
        log.error("Provider error: %s", err)
        return None
    return resp


def submit_feedback(invocation_id, score):
    """Submit reputation feedback (0-500 scale)."""
    ok, _ = oasyced_tx([
        "reputation", "submit-feedback",
        invocation_id, str(score),
    ])
    return ok


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def discover_data_asset():
    """Find an ACTIVE data asset to buy shares of."""
    data = rest_get("/oasyce/datarights/v1/data_assets")
    if not data:
        return None
    assets = [a for a in data.get("data_assets", [])
              if a.get("status", "").upper() in ("ACTIVE", "ASSET_STATUS_ACTIVE", "")
              and a.get("id", "").startswith("DATA_")]
    if not assets:
        return None
    # Pick one we haven't bought yet (smallest total_shares = freshest)
    assets.sort(key=lambda a: int(a.get("total_shares", "0") or "0"))
    return assets[0]


def buy_data_shares(asset_id, amount_uoas="100000"):
    """Buy shares of a data asset on the bonding curve."""
    ok, txhash = oasyced_tx([
        "datarights", "buy-shares", asset_id, f"{amount_uoas}uoas",
    ])
    if not ok:
        return None
    time.sleep(7)
    return txhash


def check_data_access(asset_id, addr):
    """Check access level after buying shares."""
    data = rest_get(f"/oasyce/datarights/v1/access_level/{asset_id}/{addr}")
    if not data:
        return None
    return data.get("access_level", "")


def data_asset_cycle(addr, state):
    """Optional: discover a data asset, buy shares, check access. Returns True if traded."""
    asset = discover_data_asset()
    if not asset:
        log.info("No data assets found, skipping data cycle")
        return False

    asset_id = asset["id"]
    asset_name = asset.get("name", "?")
    log.info("Data asset: %s (%s)", asset_name, asset_id)

    # Buy a small amount (0.1 OAS = 100000 uoas)
    log.info("Buying 0.1 OAS of shares in %s...", asset_id)
    txhash = buy_data_shares(asset_id, "100000")
    if not txhash:
        log.warning("Failed to buy data shares")
        return False

    # Check access level
    level = check_data_access(asset_id, addr)
    log.info("Access level for %s: %s", asset_id, level or "none")

    service_url = asset.get("service_url", "")
    if service_url:
        log.info("Data service_url: %s", service_url)

    state["total_data_purchases"] = state.get("total_data_purchases", 0) + 1
    return True


def main():
    state = load_state()
    log.info("=== Consumer Agent Cycle %d ===", state["total_invocations"] + 1)

    # 1. Ensure consumer key exists
    addr = get_address()
    addr = ensure_consumer_key(addr)
    if not addr:
        log.error("Cannot get consumer address, aborting")
        return finish_cycle(state, "consumer_key_error", exit_code=1, error="cannot get consumer address")

    log.info("Consumer: %s", addr)

    # 2. Check balance, faucet if low
    bal = check_balance(addr)
    log.info("Balance: %d uoas (%.1f OAS)", bal, bal / 1e6)
    if bal < MIN_BALANCE_UOAS:
        requested, _, faucet_error = request_faucet(addr)
        if requested:
            observed = wait_for_balance_increase(addr, bal)
            if observed is not None:
                bal = observed
            else:
                updated = check_balance(addr)
                log.warning(
                    "Faucet reported success but balance did not increase (before=%d after=%d)",
                    bal,
                    updated,
                )
                bal = updated
                faucet_error = "faucet reported success but balance did not increase"
        else:
            time.sleep(2)
            bal = check_balance(addr)
        log.info("Balance after faucet: %d uoas", bal)
        if bal < MIN_BALANCE_UOAS:
            log.warning("Balance below preferred minimum after faucet: %d", bal)

    # 3. Pre-check provider health (avoid burning OAS if upstream is down)
    prov_health, provider_error = fetch_provider_health(probe=True)
    if provider_error:
        log.error("Provider unreachable, skipping cycle: %s", provider_error)
        return finish_cycle(state, "provider_unreachable", exit_code=1, error=provider_error)
    if prov_health.get("status") != "ok":
        log.warning("Provider not ok (%s), skipping cycle", prov_health.get("status"))
        return finish_cycle(
            state,
            "provider_not_ok",
            exit_code=0,
            error=f"provider status={prov_health.get('status')}",
        )

    # 4. Discover active capability
    cap = discover_capability(prov_health.get("capability_id", ""))
    if not cap:
        log.error("No active capabilities found")
        return finish_cycle(
            state,
            "capability_unavailable",
            exit_code=1,
            error=f"provider capability unavailable: {prov_health.get('capability_id', '')}",
        )

    cap_id = cap["id"]
    cap_name = cap.get("name", "?")
    log.info("Target: %s (%s)", cap_name, cap_id)
    required_balance = parse_uoas(cap.get("price_per_call", "0")) + TX_FEE_UOAS
    if required_balance <= 0:
        required_balance = TX_FEE_UOAS
    if bal < required_balance:
        reason = f"balance {bal}uoas below required invoke budget {required_balance}uoas for {cap_id}"
        log.warning("Skipping cycle: %s", reason)
        return finish_cycle(state, "insufficient_funds", exit_code=0, error=reason)

    # 5. Invoke on-chain
    input_data = {
        "prompt": f"You are a helpful assistant. Say hello and tell me the current time estimate. This is test invocation #{state['total_invocations'] + 1}.",
    }
    log.info("Invoking %s on-chain...", cap_id)
    txhash = invoke_on_chain(cap_id, input_data)
    if not txhash:
        log.error("On-chain invoke failed")
        return finish_cycle(state, "invoke_failed", exit_code=1, error="on-chain invoke failed")

    # 6. Find invocation ID from TX events
    inv_id = find_invocation_from_tx(txhash)
    if not inv_id:
        log.error("Could not extract invocation ID from TX %s", txhash)
        return finish_cycle(
            state,
            "invocation_lookup_failed",
            exit_code=1,
            error=f"could not extract invocation ID from tx {txhash}",
        )

    state["total_invocations"] += 1
    log.info("Invocation: %s", inv_id)

    # 7. POST to provider
    final_status = "ok"
    final_error = ""
    resp = post_to_provider(inv_id, input_data)
    if resp:
        log.info("Provider responded: hash=%s", resp.get("output_hash", "?")[:16])
        result = resp.get("result", {})
        if isinstance(result, dict) and "text" in result:
            log.info("Result preview: %s", result["text"][:100])

        # 8. Submit positive feedback
        log.info("Submitting feedback (score=450)...")
        time.sleep(7)  # wait for complete-invocation to land
        if submit_feedback(inv_id, 450):
            state["total_settlements"] += 1
            log.info("Feedback submitted")
        else:
            final_status = "feedback_failed"
            final_error = f"feedback submission failed for {inv_id}"
    else:
        log.warning("Provider did not respond, skipping feedback")
        final_status = "provider_error"
        final_error = f"provider did not process invocation {inv_id}"

    # 9. Data asset cycle — discover, buy, check access
    data_asset_cycle(addr, state)

    log.info("Cycle complete: invocations=%d settlements=%d data_purchases=%d",
             state["total_invocations"], state["total_settlements"],
             state.get("total_data_purchases", 0))
    return finish_cycle(
        state,
        final_status,
        exit_code=0,
        error=final_error,
        success=(final_status == "ok"),
    )


if __name__ == "__main__":
    sys.exit(main())
