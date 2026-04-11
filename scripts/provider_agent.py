#!/usr/bin/env python3
"""
Oasyce Provider Agent — SDK-backed compatibility wrapper for selling capabilities.

Bridges off-chain API services (OpenAI, custom models, etc.) to the Oasyce
on-chain capability system. Handles verification, forwarding, settlement.

USAGE
=====

1. Register a capability on-chain:

    python3 provider_agent.py --register --name "Codex API" --price 50000
    # Registers with the local SDK signer, prints the capability ID.
    # Set OASYCE_CAPABILITY_ID to the printed ID, or pass --capability-id.

2. Start the provider agent:

    export UPSTREAM_API_KEY="sk-..."
    export UPSTREAM_API_URL="https://api.openai.com/v1/chat/completions"
    export OASYCE_CAPABILITY_ID="CAP_0000000000000001"
    python3 provider_agent.py

3. Consumer flow:

    a) Consumer invokes on-chain through the SDK-native signer path.

    b) Consumer POSTs to this agent:
       curl -X POST http://provider-host:8430/api/v1/process \\
           -H "Content-Type: application/json" \\
           -d '{"invocation_id":"INV_0000000000000001","input":{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}}'

    c) Agent verifies invocation on-chain, forwards to upstream, completes
       on-chain, returns result. After 100 blocks (~8 min), auto-claims payment.

ENVIRONMENT VARIABLES
=====================
    OASYCE_CHAIN_REST     — chain REST (default: "http://localhost:1317")
    OASYCE_CHAIN_RPC      — chain RPC  (default: "http://localhost:26657")
    OASYCE_CHAIN_ID       — chain ID (default: "oasyce-testnet-1")
    OASYCE_MNEMONIC       — optional headless signer override
    OASYCE_DIR            — local SDK binding dir (default: "~/.oasyce")
    UPSTREAM_API_URL      — upstream API endpoint
    UPSTREAM_API_KEY      — upstream API key
    OASYCE_CAPABILITY_ID  — the capability ID this agent serves
    PROVIDER_PORT         — HTTP listen port (default: 8430)

This script remains in the chain repo only as a thin wrapper. The canonical AI
runtime and signer path live in `oasyce-sdk`.
"""

import hashlib
import json
import os
import subprocess
import sys
import threading
import time
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
import socketserver
from urllib.parse import parse_qs, urlparse
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
if SCRIPT_DIR not in sys.path:
    sys.path.insert(0, SCRIPT_DIR)

from _sdk_compat import resolve_runtime, split_csv, submit_single, tx_status

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

CHAIN_REST = os.environ.get("OASYCE_CHAIN_REST", "http://localhost:1317").rstrip("/")
CHAIN_RPC = os.environ.get("OASYCE_CHAIN_RPC", "http://localhost:26657").rstrip("/")
UPSTREAM_API_URL = os.environ.get("UPSTREAM_API_URL", "")
UPSTREAM_API_KEY = os.environ.get("UPSTREAM_API_KEY", "")
CAPABILITY_ID = os.environ.get("OASYCE_CAPABILITY_ID", "")
PROVIDER_PORT = int(os.environ.get("PROVIDER_PORT", "8430"))
CHAIN_ID = os.environ.get("OASYCE_CHAIN_ID") or os.environ.get("OASYCED_CHAIN_ID", "oasyce-testnet-1")
ALERT_EMAIL = os.environ.get("OASYCE_ALERT_EMAIL", "ptc0428@qq.com")
ALERT_LOG = os.environ.get("OASYCE_ALERT_LOG", "/tmp/oasyce-provider-alert.log")
ALERT_STATE_DIR = os.environ.get("OASYCE_ALERT_STATE_DIR", "/tmp/oasyce_provider_alerts")
AUTO_DEACTIVATE_ON_BUY_FAILURE = os.environ.get("OASYCE_AUTO_DEACTIVATE_ON_BUY_FAILURE", "1") == "1"
AUTO_DEACTIVATE_FAILURE_THRESHOLD = max(
    1, int(os.environ.get("OASYCE_AUTO_DEACTIVATE_FAILURE_THRESHOLD", "3"))
)

CHALLENGE_WINDOW = 100  # blocks
BLOCK_TIME_S = 5        # seconds per block
CLAIM_POLL_INTERVAL = 30  # seconds between claim-readiness checks

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("provider-agent")

_RUNTIME = None

# Exit codes: EX_CONFIG for preflight/config errors (systemd will not restart),
# 1 for runtime errors (systemd will restart).
EX_CONFIG = 2

# ---------------------------------------------------------------------------
# Chain helpers
# ---------------------------------------------------------------------------

def chain_rest_get(path):
    """GET a chain REST endpoint, return parsed JSON or None."""
    url = f"{CHAIN_REST}{path}"
    try:
        req = Request(url)
        with urlopen(req, timeout=10) as resp:
            return json.loads(resp.read().decode())
    except (URLError, HTTPError, json.JSONDecodeError) as e:
        log.error("REST GET %s failed: %s", url, e)
        return None


def chain_rpc_get(path):
    """GET a chain RPC endpoint, return parsed JSON or None."""
    url = f"{CHAIN_RPC}{path}"
    try:
        req = Request(url)
        with urlopen(req, timeout=10) as resp:
            return json.loads(resp.read().decode())
    except (URLError, HTTPError, json.JSONDecodeError) as e:
        log.error("RPC GET %s failed: %s", url, e)
        return None


def get_latest_block_height():
    """Query the current block height from the RPC endpoint."""
    data = chain_rpc_get("/status")
    if data:
        try:
            return int(data["result"]["sync_info"]["latest_block_height"])
        except (KeyError, ValueError, TypeError):
            pass
    return 0


def get_capability(cap_id):
    """Query a capability by ID via REST."""
    return chain_rest_get(f"/oasyce/capability/v1/capability/{cap_id}")



def get_runtime():
    global _RUNTIME
    if _RUNTIME is None:
        _RUNTIME = resolve_runtime(CHAIN_REST, CHAIN_ID)
    return _RUNTIME


def get_invocation(inv_id):
    """Query an invocation by ID via REST."""
    return chain_rest_get(f"/oasyce/capability/v1/invocation/{inv_id}")


def verify_invocation_on_chain(inv_id, capability_id):
    """
    Verify an invocation exists on-chain, is PENDING, and matches our capability.
    Uses the QueryInvocation REST endpoint.
    """
    if not inv_id or not inv_id.startswith("INV_"):
        return False, "invalid invocation ID"

    # Query invocation directly
    inv_data = get_invocation(inv_id)
    if inv_data and "invocation" in inv_data:
        inv = inv_data["invocation"]
        if inv.get("capability_id") != capability_id:
            return False, f"invocation belongs to capability {inv.get('capability_id')}, not {capability_id}"
        if "PENDING" not in inv.get("status", ""):
            return False, f"invocation status is {inv.get('status')}, expected PENDING"
        # Verify we are the provider
        provider_addr = get_provider_address()
        if provider_addr and inv.get("provider") != provider_addr:
            return False, f"invocation provider mismatch"
        return True, "ok"

    # Fallback: if invocation query fails, verify capability ownership at minimum
    cap_data = get_capability(capability_id)
    if not cap_data:
        return False, "capability not found on-chain"

    cap = cap_data.get("capability", {})
    if not cap.get("is_active", False):
        return False, "capability is not active"

    provider_addr = get_provider_address()
    if provider_addr and cap.get("provider", "") != provider_addr:
        return False, f"capability provider mismatch"

    return True, "ok (invocation query unavailable, capability verified)"


def get_provider_address():
    """Get the provider's chain actor address."""
    return get_runtime().actor_address


def log_alert_event(level, msg):
    ts = time.strftime("%Y-%m-%d %H:%M:%S")
    try:
        with open(ALERT_LOG, "a", encoding="utf-8") as f:
            f.write(f"{ts}: {level}: {msg}\n")
    except OSError as e:
        log.warning("Could not write alert log %s: %s", ALERT_LOG, e)


def ensure_alert_state_dir():
    os.makedirs(ALERT_STATE_DIR, exist_ok=True)


def alert_state_path(key):
    safe = "".join(ch if ch.isalnum() or ch in "._-" else "_" for ch in key)
    return os.path.join(ALERT_STATE_DIR, f"{safe}.active")


def send_alert_email(msg):
    ts = time.strftime("%Y-%m-%d %H:%M:%S")
    subject = f"[Oasyce Alert] {msg}"
    mail = (
        f"Subject: {subject}\n"
        f"From: Oasyce Monitor <ptc0428@qq.com>\n"
        f"To: {ALERT_EMAIL}\n"
        "Content-Type: text/plain; charset=utf-8\n\n"
        f"{msg}\n\nTime: {ts}\n"
    )
    try:
        subprocess.run(
            ["msmtp", ALERT_EMAIL],
            input=mail,
            text=True,
            capture_output=True,
            check=False,
        )
    except FileNotFoundError:
        log.warning("msmtp not found; alert email skipped")


def activate_alert_once(key, msg):
    ensure_alert_state_dir()
    path = alert_state_path(key)
    if os.path.exists(path):
        return False
    log_alert_event("ALERT", msg)
    send_alert_email(msg)
    with open(path, "w", encoding="utf-8") as f:
        f.write("1\n")
    return True

# ---------------------------------------------------------------------------
# Upstream API call
# ---------------------------------------------------------------------------

def call_upstream(input_data):
    """
    Forward a request to the upstream API. Returns (response_body: bytes, error: str|None).
    """
    if not UPSTREAM_API_URL:
        return None, "UPSTREAM_API_URL not configured"

    headers = {"Content-Type": "application/json"}
    if UPSTREAM_API_KEY:
        headers["Authorization"] = f"Bearer {UPSTREAM_API_KEY}"

    body = json.dumps(input_data).encode("utf-8")
    req = Request(UPSTREAM_API_URL, data=body, headers=headers, method="POST")

    try:
        with urlopen(req, timeout=120) as resp:
            return resp.read(), None
    except HTTPError as e:
        err_body = e.read().decode("utf-8", errors="replace")
        return None, f"upstream HTTP {e.code}: {err_body[:500]}"
    except URLError as e:
        return None, f"upstream connection error: {e.reason}"

# ---------------------------------------------------------------------------
# Background claim scheduler
# ---------------------------------------------------------------------------

# Upstream status is only updated on buyer-path probes or real invocation attempts.
_upstream_ok = True
_upstream_known = False
_upstream_error = ""
_upstream_check_ts = 0
_capability_ok = False
_capability_check_ts = 0
_capability_error = "capability not checked yet"
_deactivated = False
_buyer_failure_streak = 0

def record_upstream_status(ok, error=""):
    global _upstream_ok, _upstream_known, _upstream_error, _upstream_check_ts
    _upstream_ok = ok
    _upstream_known = True
    _upstream_error = error
    _upstream_check_ts = time.time()


def probe_upstream(force=False):
    """Probe upstream only when a buyer path requests it."""
    global _upstream_ok, _upstream_error, _upstream_check_ts, _upstream_known
    now = time.time()
    if not force and _upstream_known and now - _upstream_check_ts < 30:
        return _upstream_ok, _upstream_error
    _upstream_check_ts = now
    if not UPSTREAM_API_URL:
        record_upstream_status(False, "UPSTREAM_API_URL not configured")
        return _upstream_ok, _upstream_error
    try:
        req = Request(UPSTREAM_API_URL, method="HEAD")
        with urlopen(req, timeout=5) as resp:
            record_upstream_status(True, "")
    except Exception:
        try:
            body = json.dumps({"prompt": "health check", "max_tokens": 1}).encode()
            req = Request(UPSTREAM_API_URL, data=body,
                          headers={"Content-Type": "application/json"}, method="POST")
            with urlopen(req, timeout=10) as resp:
                record_upstream_status(True, "")
        except HTTPError as e:
            err = e.read().decode("utf-8", errors="replace")[:500]
            record_upstream_status(False, f"upstream HTTP {e.code}: {err}")
        except Exception as e:
            record_upstream_status(False, f"upstream connection error: {e}")
    return _upstream_ok, _upstream_error


def _check_capability_cached(force=False):
    """Return whether the configured capability exists, is active, and belongs to us."""
    global _capability_ok, _capability_check_ts, _capability_error
    if _deactivated:
        return False, _capability_error or "capability locally deactivated"
    now = time.time()
    if not force and now - _capability_check_ts < 60:
        return _capability_ok, _capability_error

    _capability_check_ts = now
    if not CAPABILITY_ID:
        _capability_ok = False
        _capability_error = "capability ID is not configured"
        return _capability_ok, _capability_error

    cap_data = get_capability(CAPABILITY_ID)
    if not cap_data:
        _capability_ok = False
        _capability_error = "capability not found on-chain"
        return _capability_ok, _capability_error

    cap = cap_data.get("capability", {})
    if not cap.get("is_active", False):
        _capability_ok = False
        _capability_error = "capability is inactive on-chain"
        return _capability_ok, _capability_error

    provider_addr = get_provider_address()
    if provider_addr and cap.get("provider", "") != provider_addr:
        _capability_ok = False
        _capability_error = f"capability belongs to {cap.get('provider', '')}, not {provider_addr}"
        return _capability_ok, _capability_error

    _capability_ok = True
    _capability_error = ""
    return _capability_ok, _capability_error


def disable_capability(reason, invocation_id=""):
    global _deactivated, _capability_ok, _capability_check_ts, _capability_error

    reason = (reason or "unknown upstream failure").strip()
    msg = f"Capability {CAPABILITY_ID} auto-disabled after buyer-path failure"
    if invocation_id:
        msg += f" ({invocation_id})"
    msg += f": {reason}"
    activate_alert_once(f"provider_capability_disabled_{CAPABILITY_ID}", msg)

    if _deactivated:
        return False

    _deactivated = True
    _capability_ok = False
    _capability_check_ts = time.time()
    _capability_error = f"capability locally disabled after buyer-path failure: {reason}"

    if AUTO_DEACTIVATE_ON_BUY_FAILURE and CAPABILITY_ID:
        result = submit_single(
            get_runtime(),
            "/oasyce.capability.v1.MsgDeactivateCapability",
            {"creator": get_provider_address(), "capability_id": CAPABILITY_ID},
        )
        ok, out = tx_status(result)
        if ok:
            log.error("Capability %s deactivated on-chain after buyer-path failure", CAPABILITY_ID)
            return True
        log.error("Capability %s local disable set but on-chain deactivate failed: %s", CAPABILITY_ID, out)
    return False


def reset_buyer_failure_streak():
    global _buyer_failure_streak
    _buyer_failure_streak = 0


def handle_buyer_path_failure(invocation_id, reason):
    global _buyer_failure_streak
    record_upstream_status(False, reason)
    if invocation_id:
        result = submit_single(
            get_runtime(),
            "/oasyce.capability.v1.MsgFailInvocation",
            {"creator": get_provider_address(), "invocation_id": invocation_id},
        )
        ok, out = tx_status(result)
        if not ok:
            log.warning("Failed to mark invocation %s as failed on-chain: %s", invocation_id, out)
    _buyer_failure_streak += 1
    log.warning(
        "Buyer-path failure streak for %s is now %d/%d: %s",
        CAPABILITY_ID,
        _buyer_failure_streak,
        AUTO_DEACTIVATE_FAILURE_THRESHOLD,
        reason,
    )
    if AUTO_DEACTIVATE_ON_BUY_FAILURE and _buyer_failure_streak >= AUTO_DEACTIVATE_FAILURE_THRESHOLD:
        disable_capability(
            f"failure threshold reached after {_buyer_failure_streak} consecutive buyer-path failures: {reason}",
            invocation_id=invocation_id,
        )


def build_health_status(probe=False):
    capability_ok, capability_error = _check_capability_cached(force=probe)

    if _deactivated:
        return 503, {
            "status": "deactivated",
            "capability_id": CAPABILITY_ID,
            "upstream": UPSTREAM_API_URL or "(not configured)",
            "upstream_ok": _upstream_ok if _upstream_known else None,
            "upstream_error": _upstream_error,
            "upstream_known": _upstream_known,
            "capability_ok": False,
            "capability_error": _capability_error,
            "deactivated": True,
        }

    if not capability_ok:
        return 503, {
            "status": "inactive",
            "capability_id": CAPABILITY_ID,
            "upstream": UPSTREAM_API_URL or "(not configured)",
            "upstream_ok": _upstream_ok if _upstream_known else None,
            "upstream_error": _upstream_error,
            "upstream_known": _upstream_known,
            "capability_ok": capability_ok,
            "capability_error": capability_error,
            "deactivated": False,
        }

    if probe:
        upstream_ok, upstream_error = probe_upstream(force=True)
        if not upstream_ok:
            return 503, {
                "status": "degraded",
                "capability_id": CAPABILITY_ID,
                "upstream": UPSTREAM_API_URL or "(not configured)",
                "upstream_ok": False,
                "upstream_error": upstream_error,
                "upstream_known": True,
                "capability_ok": capability_ok,
                "capability_error": capability_error,
                "deactivated": False,
            }

    return 200, {
        "status": "ok",
        "capability_id": CAPABILITY_ID,
        "upstream": UPSTREAM_API_URL or "(not configured)",
        "upstream_ok": _upstream_ok if _upstream_known else None,
        "upstream_error": _upstream_error,
        "upstream_known": _upstream_known,
        "capability_ok": capability_ok,
        "capability_error": capability_error,
        "deactivated": False,
        "buyer_failure_streak": _buyer_failure_streak,
    }

# Track pending claims: {invocation_id: completed_height}
_pending_claims = {}
_pending_lock = threading.Lock()


def schedule_claim(invocation_id, completed_height):
    """Schedule a claim for after the challenge window."""
    with _pending_lock:
        _pending_claims[invocation_id] = completed_height
    log.info("Scheduled claim for %s after block %d",
             invocation_id, completed_height + CHALLENGE_WINDOW)


def claim_worker():
    """Background thread that claims invocations after challenge window."""
    log.info("Claim worker started (poll interval=%ds)", CLAIM_POLL_INTERVAL)
    while True:
        time.sleep(CLAIM_POLL_INTERVAL)
        current_height = get_latest_block_height()
        if current_height == 0:
            continue

        claimable = []
        with _pending_lock:
            for inv_id, completed_h in list(_pending_claims.items()):
                if current_height >= completed_h + CHALLENGE_WINDOW:
                    claimable.append(inv_id)

        for inv_id in claimable:
            log.info("Challenge window passed for %s at height %d, claiming...",
                     inv_id, current_height)
            result = get_runtime().signer.claim_invocation(inv_id)
            ok, out = tx_status(result)
            if ok:
                log.info("Claimed %s successfully", inv_id)
                with _pending_lock:
                    _pending_claims.pop(inv_id, None)
            else:
                log.error("Claim failed for %s: %s (will retry)", inv_id, out)

# ---------------------------------------------------------------------------
# HTTP handler
# ---------------------------------------------------------------------------

class ProviderHandler(BaseHTTPRequestHandler):
    """Handles incoming consumer requests."""

    def log_message(self, fmt, *args):
        log.info("HTTP %s", fmt % args)

    def _respond_json(self, code, data):
        body = json.dumps(data).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        parsed = urlparse(self.path)
        if parsed.path == "/health":
            probe = parse_qs(parsed.query).get("probe", ["0"])[0] == "1"
            code, payload = build_health_status(probe=probe)
            payload["probe"] = probe
            self._respond_json(code, payload)
            return

        if self.path == "/status":
            height = get_latest_block_height()
            with _pending_lock:
                pending = dict(_pending_claims)
            self._respond_json(200, {
                "capability_id": CAPABILITY_ID,
                "chain_height": height,
                "pending_claims": {k: v + CHALLENGE_WINDOW for k, v in pending.items()},
            })
            return

        self._respond_json(404, {"error": "not found"})

    def do_POST(self):
        if self.path != "/api/v1/process":
            self._respond_json(404, {"error": "not found"})
            return

        # Read body
        content_length = int(self.headers.get("Content-Length", 0))
        if content_length == 0 or content_length > 10 * 1024 * 1024:  # 10MB limit
            self._respond_json(400, {"error": "invalid content length"})
            return

        raw = self.rfile.read(content_length)
        try:
            req = json.loads(raw)
        except json.JSONDecodeError:
            self._respond_json(400, {"error": "invalid JSON"})
            return

        invocation_id = req.get("invocation_id", "").strip()
        input_data = req.get("input")

        if not invocation_id:
            self._respond_json(400, {"error": "missing invocation_id"})
            return
        if input_data is None:
            self._respond_json(400, {"error": "missing input"})
            return

        if _deactivated:
            handle_buyer_path_failure(invocation_id, "capability already deactivated")
            self._respond_json(503, {"error": "capability is deactivated"})
            return

        log.info("Processing invocation %s", invocation_id)

        # Step 1: Verify invocation on-chain
        log.info("[%s] Verifying invocation on-chain...", invocation_id)
        ok, reason = verify_invocation_on_chain(invocation_id, CAPABILITY_ID)
        if not ok:
            log.warning("[%s] Verification failed: %s", invocation_id, reason)
            self._respond_json(403, {"error": f"invocation verification failed: {reason}"})
            return
        log.info("[%s] Invocation verified", invocation_id)

        # Step 2: Forward to upstream API
        log.info("[%s] Calling upstream API...", invocation_id)
        upstream_resp, err = call_upstream(input_data)
        if err:
            log.error("[%s] Upstream call failed: %s", invocation_id, err)
            handle_buyer_path_failure(invocation_id, err)
            self._respond_json(502, {"error": f"upstream error: {err}"})
            return
        reset_buyer_failure_streak()

        # Step 3: Hash the output and extract usage metrics
        output_hash = hashlib.sha256(upstream_resp).hexdigest()
        log.info("[%s] Output hash: %s", invocation_id, output_hash)

        # Extract token usage from upstream response (OpenAI-compatible format)
        usage_report = ""
        try:
            resp_json = json.loads(upstream_resp.decode("utf-8"))
            if isinstance(resp_json, dict) and "usage" in resp_json:
                usage_report = json.dumps(resp_json["usage"], separators=(",", ":"))
                log.info("[%s] Usage: %s", invocation_id, usage_report)
        except (json.JSONDecodeError, UnicodeDecodeError):
            pass

        # Step 4: Complete invocation on-chain (starts challenge window)
        record_upstream_status(True, "")

        log.info("[%s] Submitting complete-invocation on-chain...", invocation_id)
        result = get_runtime().signer.complete_invocation(
            invocation_id,
            output_hash,
            usage_report=usage_report,
        )
        ok, tx_out = tx_status(result)
        if not ok:
            log.error("[%s] complete-invocation TX failed: %s", invocation_id, tx_out)
            # Still return the result to the consumer -- they paid for it.
            # The on-chain settlement can be retried manually.

        # Step 5: Schedule claim after challenge window
        current_height = get_latest_block_height()
        if current_height > 0:
            schedule_claim(invocation_id, current_height)
        else:
            log.warning("[%s] Could not get block height for claim scheduling", invocation_id)

        # Step 6: Return result to consumer
        try:
            result = json.loads(upstream_resp.decode("utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError):
            # Non-JSON response, return as base64-ish string
            result = {"raw": upstream_resp.decode("utf-8", errors="replace")}

        resp_body = {
            "invocation_id": invocation_id,
            "output_hash": output_hash,
            "result": result,
        }
        if usage_report:
            resp_body["usage"] = json.loads(usage_report)
        self._respond_json(200, resp_body)
        log.info("[%s] Response sent to consumer", invocation_id)

# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------

def register_capability(name, price, description="", tags=""):
    """Register a new capability on-chain. Returns capability ID or exits."""
    log.info("Registering capability: name=%s, price=%duoas", name, price)

    upstream_ok, upstream_error = probe_upstream(force=True)
    if not upstream_ok:
        log.error("Upstream validation failed; refusing to register capability: %s", upstream_error)
        sys.exit(1)

    result = get_runtime().signer.register_capability(
        name=name,
        endpoint=f"http://localhost:{PROVIDER_PORT}/api/v1/process",
        price_uoas=price,
        description=description,
        tags=split_csv(tags),
    )
    ok, out = tx_status(result)
    if not ok:
        log.error("Registration failed: %s", out)
        sys.exit(1)

    # Wait for TX to land
    log.info("Waiting for TX to be included in a block...")
    time.sleep(7)

    # Find our capability by listing all
    caps_data = chain_rest_get("/oasyce/capability/v1/capabilities")
    if caps_data:
        caps = caps_data.get("capabilities", [])
        provider_addr = get_provider_address()
        for cap in reversed(caps):
            if cap.get("name") == name and cap.get("provider") == provider_addr:
                cap_id = cap["id"]
                log.info("Registered capability: %s", cap_id)
                print(f"\nCapability registered successfully.")
                print(f"  ID:    {cap_id}")
                print(f"  Name:  {name}")
                print(f"  Price: {price}uoas")
                print(f"\nSet this environment variable before starting the agent:")
                print(f"  export OASYCE_CAPABILITY_ID={cap_id}")
                return cap_id

    log.warning("TX submitted (hash=%s) but could not confirm capability ID.", out)
    log.warning("Query the capability list through REST or the SDK to confirm the new capability.")
    print(f"\nTX submitted: {out}")
    print("Could not confirm capability ID -- query the chain manually.")
    return None

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Oasyce Provider Agent -- sell API access via the capability marketplace",
    )
    parser.add_argument("--register", action="store_true",
                        help="Register a new capability on-chain, then exit")
    parser.add_argument("--name", default="API Service",
                        help="Capability name (used with --register)")
    parser.add_argument("--price", type=int, default=100000,
                        help="Price per call in uoas (used with --register)")
    parser.add_argument("--description", default="",
                        help="Capability description (used with --register)")
    parser.add_argument("--tags", default="",
                        help="Comma-separated tags (used with --register)")
    parser.add_argument("--capability-id", default="",
                        help="Override OASYCE_CAPABILITY_ID env var")
    parser.add_argument("--port", type=int, default=0,
                        help="Override PROVIDER_PORT env var")

    args = parser.parse_args()

    # Handle --register mode
    if args.register:
        register_capability(args.name, args.price, args.description, args.tags)
        return

    # Resolve capability ID
    global CAPABILITY_ID, PROVIDER_PORT
    global _deactivated, _capability_ok, _capability_check_ts, _capability_error
    if args.capability_id:
        CAPABILITY_ID = args.capability_id
    if args.port:
        PROVIDER_PORT = args.port

    # Validate required config
    errors = []
    if not CAPABILITY_ID:
        errors.append("OASYCE_CAPABILITY_ID is not set (or use --capability-id)")
    if not UPSTREAM_API_URL:
        errors.append("UPSTREAM_API_URL is not set")
    if errors:
        for e in errors:
            log.error("Config error: %s", e)
        sys.exit(EX_CONFIG)

    # Resolve SDK-native identity
    try:
        addr = get_provider_address()
    except RuntimeError as exc:
        log.error("Cannot resolve provider identity: %s", exc)
        sys.exit(EX_CONFIG)
    log.info("Provider address: %s", addr)

    # Verify capability exists on-chain
    cap_data = get_capability(CAPABILITY_ID)
    if cap_data:
        cap = cap_data.get("capability", {})
        log.info("Serving capability: %s (%s)", cap.get("name", "?"), CAPABILITY_ID)
        log.info("Price per call: %s", cap.get("price_per_call", "?"))
        if not cap.get("is_active", False):
            _deactivated = True
            _capability_ok = False
            _capability_check_ts = time.time()
            _capability_error = "capability is inactive on-chain"
            log.warning("Capability %s is inactive on-chain; starting in deactivated mode", CAPABILITY_ID)
        if cap.get("provider") != addr:
            log.error("Capability %s belongs to %s, not %s",
                      CAPABILITY_ID, cap.get("provider"), addr)
            sys.exit(EX_CONFIG)
        if not _deactivated:
            _capability_ok = True
            _capability_check_ts = time.time()
            _capability_error = ""
    else:
        log.warning("Could not verify capability %s on-chain (REST may be down)", CAPABILITY_ID)

    # Start claim worker thread
    t = threading.Thread(target=claim_worker, daemon=True)
    t.start()

    # Start HTTP server
    class ReuseServer(HTTPServer):
        allow_reuse_address = True
    server = ReuseServer(("0.0.0.0", PROVIDER_PORT), ProviderHandler)
    log.info("Provider agent listening on :%d", PROVIDER_PORT)
    log.info("  POST /api/v1/process   -- process invocations")
    log.info("  GET  /health           -- health check")
    log.info("  GET  /status           -- pending claims & chain height")
    log.info("Upstream: %s", UPSTREAM_API_URL)

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        log.info("Shutting down...")
        server.shutdown()


if __name__ == "__main__":
    main()
