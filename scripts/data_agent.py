#!/usr/bin/env python3
"""
Oasyce Data Agent -- Autonomous data asset registration.

Scans directories for files, runs oasyce-sdk privacy detection, and
auto-registers safe assets (risk=safe/low) on-chain. Zero human intervention.

USAGE
=====

1. Single-cycle test (dry run):

    python3 data_agent.py --once --dry-run --watch-dirs /path/to/data

2. Single-cycle registration:

    python3 data_agent.py --once --watch-dirs /path/to/data

3. Daemon mode (default):

    export WATCH_DIRS="/home/oasyce/production_data"
    python3 data_agent.py

ENVIRONMENT VARIABLES
=====================
    DATA_AGENT_KEY        -- keyring key name (default: "data-agent")
    OASYCE_CHAIN_REST     -- chain REST endpoint (default: "http://localhost:1317")
    OASYCED_BIN           -- path to oasyced binary (default: "oasyced")
    OASYCED_CHAIN_ID      -- chain ID (default: "oasyce-testnet-1")
    OASYCED_KEYRING       -- keyring backend (default: "test")
    DATA_AGENT_PORT       -- health endpoint port (default: 8431)
    WATCH_DIRS            -- comma-separated directories to scan (REQUIRED)
    SCAN_INTERVAL         -- seconds between scans (default: 1800 = 30 min)
    ALLOWED_EXTENSIONS    -- comma-separated extension allowlist (empty = all)
    EXCLUDE_PATTERNS      -- comma-separated path substrings to exclude
    DEFAULT_TAGS          -- extra tags appended to every registration
    MAX_RISK_LEVEL        -- max acceptable risk: "safe" or "low" (default: "low")
    SERVICE_URL_TEMPLATE  -- service_url template with {hash} placeholder
    STATE_FILE            -- persistent state file path
"""

import json
import logging
import os
import subprocess
import sys
import threading
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

AGENT_KEY = os.environ.get("DATA_AGENT_KEY", "data-agent")
CHAIN_REST = os.environ.get("OASYCE_CHAIN_REST", "http://localhost:1317").rstrip("/")
OASYCED = os.environ.get("OASYCED_BIN", "oasyced")
CHAIN_ID = os.environ.get("OASYCED_CHAIN_ID", "oasyce-testnet-1")
KEYRING = os.environ.get("OASYCED_KEYRING", "test")
AGENT_PORT = int(os.environ.get("DATA_AGENT_PORT", "8431"))
WATCH_DIRS = [d.strip() for d in os.environ.get("WATCH_DIRS", "").split(",") if d.strip()]
SCAN_INTERVAL = int(os.environ.get("SCAN_INTERVAL", "1800"))
ALLOWED_EXTENSIONS = {e.strip() for e in os.environ.get("ALLOWED_EXTENSIONS", "").split(",") if e.strip()}
EXCLUDE_PATTERNS = [p.strip() for p in os.environ.get(
    "EXCLUDE_PATTERNS", ".git,node_modules,__pycache__,.venv,build,dist"
).split(",") if p.strip()]
DEFAULT_TAGS = [t.strip() for t in os.environ.get("DEFAULT_TAGS", "").split(",") if t.strip()]
MAX_RISK_LEVEL = os.environ.get("MAX_RISK_LEVEL", "low")
SERVICE_URL_TEMPLATE = os.environ.get("SERVICE_URL_TEMPLATE", "")
STATE_FILE = os.environ.get("STATE_FILE", "/tmp/data_agent_state.json")

ACCEPTABLE_RISKS = {"safe"} if MAX_RISK_LEVEL == "safe" else {"safe", "low"}

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("data-agent")

# ---------------------------------------------------------------------------
# Runtime state
# ---------------------------------------------------------------------------

_last_cycle_stats = {}
_last_cycle_time = ""
_total_registered = 0
_total_scanned = 0
_total_cycles = 0
_lock = threading.Lock()

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
        log.debug("REST GET %s failed: %s", url, e)
        return None


def get_agent_address():
    """Get the data agent's bech32 address from keyring."""
    try:
        result = subprocess.run(
            [OASYCED, "keys", "show", AGENT_KEY, "-a",
             "--keyring-backend", KEYRING],
            capture_output=True, text=True, timeout=10,
        )
        addr = result.stdout.strip()
        return addr if addr.startswith("oasyce") else None
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return None


def oasyced_tx(args):
    """Run an oasyced tx command. Returns (success, txhash_or_error)."""
    cmd = [OASYCED, "tx"] + args + [
        "--from", AGENT_KEY,
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
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        output = result.stdout.strip() or result.stderr.strip()
        if result.returncode != 0:
            log.error("TX failed (rc=%d): %s", result.returncode, output)
            return False, output
        try:
            tx_data = json.loads(output)
            code = tx_data.get("code", 0)
            if code != 0:
                log.error("TX CheckTx failed (code=%d): %s", code, tx_data.get("raw_log", ""))
                return False, tx_data.get("raw_log", output)
            txhash = tx_data.get("txhash", "")
            log.info("TX submitted: %s", txhash)
            return True, txhash
        except json.JSONDecodeError:
            log.info("TX output: %s", output[:200])
            return True, output
    except subprocess.TimeoutExpired:
        log.error("TX timed out")
        return False, "timeout"
    except FileNotFoundError:
        log.error("oasyced binary not found at: %s", OASYCED)
        return False, "binary not found"


# ---------------------------------------------------------------------------
# oasyce-sdk agent integration (formerly DataVault)
# ---------------------------------------------------------------------------


def scan_and_filter(watch_dir):
    """Run oasyce-sdk agent scan with privacy detection. Returns filtered file list."""
    from oasyce_sdk.scanner import scan_directory
    from oasyce_sdk.inventory import Inventory

    inventory = Inventory()
    result = scan_directory(
        watch_dir, recursive=True, skip_privacy=False,
        save=True, inventory=inventory,
    )

    filtered = []
    for info in result.get("files", []):
        path = info["path"]
        ext = info.get("ext", "")
        risk = info.get("privacy_risk", "safe")

        # Extension filter
        if ALLOWED_EXTENSIONS and ext not in ALLOWED_EXTENSIONS:
            continue

        # Path exclusion
        if any(p in path for p in EXCLUDE_PATTERNS):
            continue

        # Privacy gate
        if risk not in ACCEPTABLE_RISKS:
            if risk in ("high", "critical"):
                log.warning("BLOCKED %s (risk=%s, PII detected)", path, risk)
            else:
                log.info("Skipped %s (risk=%s, above max=%s)", path, risk, MAX_RISK_LEVEL)
            continue

        filtered.append(info)

    return filtered, inventory


def is_already_registered(inventory, path, content_hash):
    """Check if file is already registered with the same content."""
    rows = inventory.search(query=path)
    for row in rows:
        if row.get("path") == path and row.get("oasyce_registered") and row.get("hash") == content_hash:
            return True
    return False


def compute_asset_name(file_path):
    """Generate asset name from filename (without extension)."""
    return Path(file_path).stem


def generate_tags(category, ext):
    """Produce comma-separated tag string."""
    tags = [category]
    if ext:
        tags.append(ext.lstrip("."))
    tags.extend(DEFAULT_TAGS)
    return ",".join(t for t in tags if t)


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------


def register_on_chain(name, content_hash, tags, service_url, description=""):
    """Register a data asset on-chain. Returns (success, asset_id_or_error)."""
    args = ["datarights", "register", name, content_hash]
    if tags:
        args.extend(["--tags", tags])
    if description:
        args.extend(["--description", description])
    if service_url:
        args.extend(["--service-url", service_url])

    ok, result = oasyced_tx(args)
    if not ok:
        return False, result

    # Wait for block inclusion
    time.sleep(7)

    # Try to find the asset by querying the list
    asset_id = find_asset_by_hash(content_hash)
    if asset_id:
        return True, asset_id

    # Fallback: generate expected ID pattern
    return True, f"DATA_{content_hash[:8].upper()}"


def find_asset_by_hash(content_hash):
    """Query chain for an asset matching this content_hash."""
    data = chain_rest_get("/oasyce/datarights/v1/data_assets")
    if not data:
        return None
    for asset in data.get("data_assets", []):
        if asset.get("content_hash") == content_hash:
            return asset.get("id", "")
    return None


# ---------------------------------------------------------------------------
# Main cycle
# ---------------------------------------------------------------------------


def scan_and_register_cycle(dry_run=False):
    """One full scan-register cycle. Returns stats dict."""
    global _last_cycle_stats, _last_cycle_time, _total_registered, _total_scanned, _total_cycles

    stats = {"scanned": 0, "registered": 0, "skipped": 0, "errors": 0, "dirs": []}

    for watch_dir in WATCH_DIRS:
        if not os.path.isdir(watch_dir):
            log.warning("Watch dir does not exist: %s", watch_dir)
            continue

        log.info("Scanning %s ...", watch_dir)
        try:
            filtered, inventory = scan_and_filter(watch_dir)
        except Exception as e:
            log.error("Scan failed for %s: %s", watch_dir, e)
            stats["errors"] += 1
            continue

        dir_stats = {"dir": watch_dir, "found": len(filtered), "registered": 0}

        for info in filtered:
            stats["scanned"] += 1
            path = info["path"]
            content_hash = info["hash"]
            category = info.get("category", "other")
            ext = info.get("ext", "")

            # Deduplication
            if is_already_registered(inventory, path, content_hash):
                stats["skipped"] += 1
                continue

            name = compute_asset_name(path)
            tags = generate_tags(category, ext)
            service_url = ""
            if SERVICE_URL_TEMPLATE:
                service_url = SERVICE_URL_TEMPLATE.format(hash=content_hash[:16])

            if dry_run:
                log.info("[DRY RUN] Would register: %s (hash=%s, tags=%s)",
                         name, content_hash[:12], tags)
                stats["registered"] += 1
                dir_stats["registered"] += 1
                continue

            ok, asset_id = register_on_chain(name, content_hash, tags, service_url)
            if ok:
                inventory.mark_registered(path, asset_id)
                log.info("REGISTERED %s -> %s", path, asset_id)
                stats["registered"] += 1
                dir_stats["registered"] += 1
            else:
                log.error("Failed to register %s: %s", path, asset_id)
                stats["errors"] += 1

        inventory.close()
        stats["dirs"].append(dir_stats)

    with _lock:
        _last_cycle_stats = stats
        _last_cycle_time = time.strftime("%Y-%m-%d %H:%M:%S")
        _total_registered += stats["registered"]
        _total_scanned += stats["scanned"]
        _total_cycles += 1

    log.info("Cycle complete: scanned=%d registered=%d skipped=%d errors=%d",
             stats["scanned"], stats["registered"], stats["skipped"], stats["errors"])

    save_state()
    return stats


# ---------------------------------------------------------------------------
# State persistence
# ---------------------------------------------------------------------------


def load_state():
    """Load cumulative state from disk."""
    global _total_registered, _total_scanned, _total_cycles
    try:
        with open(STATE_FILE) as f:
            state = json.load(f)
            _total_registered = state.get("total_registered", 0)
            _total_scanned = state.get("total_scanned", 0)
            _total_cycles = state.get("total_cycles", 0)
    except (FileNotFoundError, json.JSONDecodeError):
        pass


def save_state():
    """Persist cumulative state to disk."""
    with _lock:
        state = {
            "total_registered": _total_registered,
            "total_scanned": _total_scanned,
            "total_cycles": _total_cycles,
            "last_cycle_time": _last_cycle_time,
            "last_cycle_stats": _last_cycle_stats,
        }
    try:
        with open(STATE_FILE, "w") as f:
            json.dump(state, f, indent=2)
    except OSError as e:
        log.warning("Could not save state: %s", e)


# ---------------------------------------------------------------------------
# HTTP health handler
# ---------------------------------------------------------------------------


class DataAgentHandler(BaseHTTPRequestHandler):
    """Health and status endpoints."""

    def log_message(self, fmt, *args):
        pass  # Suppress default access logs

    def do_GET(self):
        if self.path == "/health":
            self._health()
        elif self.path == "/status":
            self._status()
        else:
            self.send_error(404)

    def _health(self):
        with _lock:
            body = {
                "status": "ok",
                "total_registered": _total_registered,
                "total_scanned": _total_scanned,
                "total_cycles": _total_cycles,
                "last_cycle_time": _last_cycle_time,
                "watch_dirs": WATCH_DIRS,
                "scan_interval_s": SCAN_INTERVAL,
            }
        self._json_response(200, body)

    def _status(self):
        with _lock:
            body = {
                "total_registered": _total_registered,
                "total_scanned": _total_scanned,
                "total_cycles": _total_cycles,
                "last_cycle_time": _last_cycle_time,
                "last_cycle_stats": _last_cycle_stats,
                "config": {
                    "watch_dirs": WATCH_DIRS,
                    "scan_interval_s": SCAN_INTERVAL,
                    "allowed_extensions": sorted(ALLOWED_EXTENSIONS) if ALLOWED_EXTENSIONS else "all",
                    "exclude_patterns": EXCLUDE_PATTERNS,
                    "max_risk_level": MAX_RISK_LEVEL,
                    "default_tags": DEFAULT_TAGS,
                    "service_url_template": SERVICE_URL_TEMPLATE,
                },
            }
        self._json_response(200, body)

    def _json_response(self, code, body):
        data = json.dumps(body, indent=2).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


# ---------------------------------------------------------------------------
# Background worker
# ---------------------------------------------------------------------------


def scan_worker():
    """Background thread: sleep → scan → register → repeat."""
    while True:
        try:
            scan_and_register_cycle(dry_run=False)
        except Exception as e:
            log.error("Scan cycle failed: %s", e)
        time.sleep(SCAN_INTERVAL)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Oasyce Data Agent -- autonomous data asset registration",
    )
    parser.add_argument("--once", action="store_true",
                        help="Run a single scan cycle and exit")
    parser.add_argument("--dry-run", action="store_true",
                        help="Scan and classify but do not broadcast TX")
    parser.add_argument("--watch-dirs", default="",
                        help="Override WATCH_DIRS (comma-separated)")
    parser.add_argument("--port", type=int, default=0,
                        help="Override DATA_AGENT_PORT")

    args = parser.parse_args()

    # CLI overrides
    global WATCH_DIRS, AGENT_PORT
    if args.watch_dirs:
        WATCH_DIRS = [d.strip() for d in args.watch_dirs.split(",") if d.strip()]
    if args.port:
        AGENT_PORT = args.port

    # Validate
    if not WATCH_DIRS:
        log.error("No watch directories configured. Set WATCH_DIRS or use --watch-dirs.")
        sys.exit(1)

    # Verify oasyce-sdk is importable
    try:
        from oasyce_sdk.scanner import scan_directory  # noqa: F401
        from oasyce_sdk.inventory import Inventory  # noqa: F401
    except ImportError:
        log.error("oasyce-sdk package not found. Install: pip install oasyce-sdk")
        sys.exit(1)

    # Load persistent state
    load_state()

    if not args.dry_run:
        # Verify agent key exists
        addr = get_agent_address()
        if not addr:
            log.error("Agent key '%s' not found in keyring (backend=%s)", AGENT_KEY, KEYRING)
            sys.exit(1)
        log.info("Agent address: %s", addr)

    log.info("Watch dirs: %s", WATCH_DIRS)
    log.info("Scan interval: %ds", SCAN_INTERVAL)
    log.info("Max risk level: %s", MAX_RISK_LEVEL)
    if ALLOWED_EXTENSIONS:
        log.info("Allowed extensions: %s", sorted(ALLOWED_EXTENSIONS))
    if DEFAULT_TAGS:
        log.info("Default tags: %s", DEFAULT_TAGS)

    # Single-cycle mode
    if args.once:
        stats = scan_and_register_cycle(dry_run=args.dry_run)
        sys.exit(0 if stats["errors"] == 0 else 1)

    # Daemon mode: worker thread + HTTP server
    t = threading.Thread(target=scan_worker, daemon=True)
    t.start()

    class ReuseServer(HTTPServer):
        allow_reuse_address = True

    server = ReuseServer(("0.0.0.0", AGENT_PORT), DataAgentHandler)
    log.info("Data agent listening on :%d", AGENT_PORT)
    log.info("  GET /health  -- health check")
    log.info("  GET /status  -- detailed stats")

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        log.info("Shutting down...")
        server.shutdown()


if __name__ == "__main__":
    main()
