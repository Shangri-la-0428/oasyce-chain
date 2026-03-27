#!/usr/bin/env python3
"""
Oasyce Provider Agent — Sell API access through the capability marketplace.

Bridges off-chain API services (OpenAI, custom models, etc.) to the Oasyce
on-chain capability system. Handles verification, forwarding, settlement.

USAGE
=====

1. Register a capability on-chain:

    python3 provider_agent.py --register --name "Codex API" --price 50000
    # Registers with the provider key, prints the capability ID.
    # Set OASYCE_CAPABILITY_ID to the printed ID, or pass --capability-id.

2. Start the provider agent:

    export UPSTREAM_API_KEY="sk-..."
    export UPSTREAM_API_URL="https://api.openai.com/v1/chat/completions"
    export OASYCE_CAPABILITY_ID="CAP_0000000000000001"
    python3 provider_agent.py

3. Consumer flow:

    a) Consumer invokes on-chain:
       oasyced tx oasyce_capability invoke $CAP_ID \\
           --input '{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}' \\
           --from consumer --chain-id oasyce-local-1 --keyring-backend test \\
           --fees 10000uoas -y

    b) Consumer POSTs to this agent:
       curl -X POST http://provider-host:8430/api/v1/process \\
           -H "Content-Type: application/json" \\
           -d '{"invocation_id":"INV_0000000000000001","input":{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}}'

    c) Agent verifies invocation on-chain, forwards to upstream, completes
       on-chain, returns result. After 100 blocks (~8 min), auto-claims payment.

ENVIRONMENT VARIABLES
=====================
    OASYCE_PROVIDER_KEY   — keyring key name (default: "provider")
    OASYCE_CHAIN_REST     — chain REST (default: "http://localhost:1317")
    OASYCE_CHAIN_RPC      — chain RPC  (default: "http://localhost:26657")
    UPSTREAM_API_URL      — upstream API endpoint
    UPSTREAM_API_KEY      — upstream API key
    OASYCE_CAPABILITY_ID  — the capability ID this agent serves
    PROVIDER_PORT         — HTTP listen port (default: 8430)
    OASYCED_BIN           — path to oasyced binary (default: "oasyced")
    OASYCED_CHAIN_ID      — chain ID (default: "oasyce-local-1")
    OASYCED_KEYRING       — keyring backend (default: "test")
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
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

PROVIDER_KEY = os.environ.get("OASYCE_PROVIDER_KEY", "provider")
CHAIN_REST = os.environ.get("OASYCE_CHAIN_REST", "http://localhost:1317").rstrip("/")
CHAIN_RPC = os.environ.get("OASYCE_CHAIN_RPC", "http://localhost:26657").rstrip("/")
UPSTREAM_API_URL = os.environ.get("UPSTREAM_API_URL", "")
UPSTREAM_API_KEY = os.environ.get("UPSTREAM_API_KEY", "")
CAPABILITY_ID = os.environ.get("OASYCE_CAPABILITY_ID", "")
PROVIDER_PORT = int(os.environ.get("PROVIDER_PORT", "8430"))
OASYCED = os.environ.get("OASYCED_BIN", "oasyced")
CHAIN_ID = os.environ.get("OASYCED_CHAIN_ID", "oasyce-testnet-1")
KEYRING = os.environ.get("OASYCED_KEYRING", "test")

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



def oasyced_tx(args):
    """
    Run an oasyced tx command. Returns (success: bool, output: str).
    Uses --output json for machine-readable output.
    """
    cmd = [
        OASYCED, "tx",
    ] + args + [
        "--from", PROVIDER_KEY,
        "--keyring-backend", KEYRING,
        "--chain-id", CHAIN_ID,
        "--gas", "auto",
        "--gas-adjustment", "1.5",
        "--fees", "10000uoas",
        "--yes",
        "--output", "json",
    ]
    log.info("TX: %s", " ".join(cmd))
    try:
        result = subprocess.run(
            cmd, capture_output=True, text=True, timeout=30,
        )
        output = result.stdout.strip() or result.stderr.strip()
        if result.returncode != 0:
            log.error("TX failed (rc=%d): %s", result.returncode, output)
            return False, output
        # Parse txhash from JSON output
        try:
            tx_data = json.loads(output)
            txhash = tx_data.get("txhash", "")
            code = tx_data.get("code", 0)
            if code != 0:
                log.error("TX CheckTx failed (code=%d): %s", code, tx_data.get("raw_log", ""))
                return False, tx_data.get("raw_log", output)
            log.info("TX submitted: %s", txhash)
            return True, txhash
        except json.JSONDecodeError:
            # Non-JSON output, still might be OK
            log.info("TX output: %s", output[:200])
            return True, output
    except subprocess.TimeoutExpired:
        log.error("TX timed out")
        return False, "timeout"
    except FileNotFoundError:
        log.error("oasyced binary not found at: %s", OASYCED)
        return False, "binary not found"


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
    """Get the provider's bech32 address from the keyring."""
    try:
        result = subprocess.run(
            [OASYCED, "keys", "show", PROVIDER_KEY, "-a",
             "--keyring-backend", KEYRING],
            capture_output=True, text=True, timeout=10,
        )
        if result.returncode == 0:
            return result.stdout.strip()
    except (subprocess.TimeoutExpired, FileNotFoundError):
        pass
    return None

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

# Upstream health cache (avoid hammering upstream on every /health)
_upstream_ok = True
_upstream_check_ts = 0

def _check_upstream_cached():
    """Return whether upstream is reachable. Cached for 60 seconds."""
    global _upstream_ok, _upstream_check_ts
    now = time.time()
    if now - _upstream_check_ts < 60:
        return _upstream_ok
    _upstream_check_ts = now
    if not UPSTREAM_API_URL:
        _upstream_ok = False
        return False
    try:
        req = Request(UPSTREAM_API_URL, method="HEAD")
        with urlopen(req, timeout=5) as resp:
            _upstream_ok = True
    except Exception:
        # HEAD might not be supported, try a minimal POST
        try:
            body = json.dumps({"prompt": ""}).encode()
            req = Request(UPSTREAM_API_URL, data=body,
                          headers={"Content-Type": "application/json"}, method="POST")
            with urlopen(req, timeout=10) as resp:
                _upstream_ok = True
        except HTTPError as e:
            # 4xx means upstream is reachable, just rejected our test
            _upstream_ok = e.code < 500
        except Exception:
            _upstream_ok = False
    return _upstream_ok

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
            ok, out = oasyced_tx(["oasyce_capability", "claim-invocation", inv_id])
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
        if self.path == "/health":
            # Check upstream reachability (cached 60s)
            upstream_ok = _check_upstream_cached()
            status = "ok" if upstream_ok else "degraded"
            code = 200 if upstream_ok else 503
            self._respond_json(code, {
                "status": status,
                "capability_id": CAPABILITY_ID,
                "upstream": UPSTREAM_API_URL or "(not configured)",
                "upstream_ok": upstream_ok,
            })
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
            # Mark upstream as unhealthy so /health reflects reality
            global _upstream_ok, _upstream_check_ts
            _upstream_ok = False
            _upstream_check_ts = time.time()
            # Report failure on-chain
            oasyced_tx(["oasyce_capability", "fail-invocation", invocation_id])
            self._respond_json(502, {"error": f"upstream error: {err}"})
            return

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
        log.info("[%s] Submitting complete-invocation on-chain...", invocation_id)
        complete_args = [
            "oasyce_capability", "complete-invocation",
            invocation_id, output_hash,
        ]
        if usage_report:
            complete_args += ["--usage-report", usage_report]
        ok, tx_out = oasyced_tx(complete_args)
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

    args = [
        "oasyce_capability", "register",
        name,
        f"http://localhost:{PROVIDER_PORT}/api/v1/process",
        f"{price}uoas",
    ]
    if description:
        args += ["--description", description]
    if tags:
        args += ["--tags", tags]

    ok, out = oasyced_tx(args)
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
    log.warning("Query manually: oasyced query oasyce_capability list --output json")
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
        sys.exit(1)

    # Verify provider key exists
    addr = get_provider_address()
    if not addr:
        log.error("Provider key '%s' not found in keyring (backend=%s)", PROVIDER_KEY, KEYRING)
        sys.exit(1)
    log.info("Provider address: %s", addr)

    # Verify capability exists on-chain
    cap_data = get_capability(CAPABILITY_ID)
    if cap_data:
        cap = cap_data.get("capability", {})
        log.info("Serving capability: %s (%s)", cap.get("name", "?"), CAPABILITY_ID)
        log.info("Price per call: %s", cap.get("price_per_call", "?"))
        if cap.get("provider") != addr:
            log.error("Capability %s belongs to %s, not %s",
                      CAPABILITY_ID, cap.get("provider"), addr)
            sys.exit(1)
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
