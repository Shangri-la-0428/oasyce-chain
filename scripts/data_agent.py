#!/usr/bin/env python3
"""
Oasyce Data Agent -- SDK-backed compatibility wrapper for autonomous data registration.

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
    OASYCE_CHAIN_REST     -- chain REST endpoint (default: "http://localhost:1317")
    OASYCE_CHAIN_ID       -- chain ID (default: "oasyce-testnet-1")
    OASYCE_MNEMONIC       -- optional headless signer override
    OASYCE_DIR            -- local SDK binding dir (default: "~/.oasyce")
    DATA_AGENT_PORT       -- health endpoint port (default: 8431)
    WATCH_DIRS            -- comma-separated directories to scan (REQUIRED)
    SCAN_INTERVAL         -- seconds between scans (default: 1800 = 30 min)
    ALLOWED_EXTENSIONS    -- comma-separated extension allowlist (empty = all)
    EXCLUDE_PATTERNS      -- comma-separated path substrings to exclude
    DEFAULT_TAGS          -- extra tags appended to every registration
    MAX_RISK_LEVEL        -- max acceptable risk: "safe" or "low" (default: "low")
    SERVICE_URL_TEMPLATE  -- service_url template with {hash} placeholder
    STATE_FILE            -- persistent state file path

This script remains in the chain repo only as a thin wrapper. The canonical AI
runtime and signer path live in `oasyce-sdk`.
"""

import json
import logging
import os
import sys
import threading
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
if SCRIPT_DIR not in sys.path:
    sys.path.insert(0, SCRIPT_DIR)

from _sdk_compat import _ensure_sdk_importable, resolve_runtime, split_csv, tx_status

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

CHAIN_REST = os.environ.get("OASYCE_CHAIN_REST", "http://localhost:1317").rstrip("/")
CHAIN_ID = os.environ.get("OASYCE_CHAIN_ID") or os.environ.get("OASYCED_CHAIN_ID", "oasyce-testnet-1")
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

_RUNTIME = None

# ---------------------------------------------------------------------------
# Runtime state
# ---------------------------------------------------------------------------

_last_cycle_stats = {}
_last_cycle_time = ""
_total_registered = 0
_total_scanned = 0
_total_cycles = 0
_registered_assets = {}
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


def get_runtime():
    global _RUNTIME
    if _RUNTIME is None:
        _RUNTIME = resolve_runtime(CHAIN_REST, CHAIN_ID)
    return _RUNTIME


def get_agent_address():
    """Get the data agent's chain actor address."""
    return get_runtime().actor_address


def get_sdk_agent_scanner():
    """Load the current scanner implementation from oasyce-sdk."""
    _ensure_sdk_importable()
    from oasyce_sdk.agent import scanner as agent_scanner

    return agent_scanner


# ---------------------------------------------------------------------------
# oasyce-sdk agent integration (formerly DataVault)
# ---------------------------------------------------------------------------


def scan_and_filter(watch_dir):
    """Run oasyce-sdk agent scan with privacy detection. Returns filtered file list."""
    scanner = get_sdk_agent_scanner()
    results = scanner.scan(
        paths=[watch_dir],
        known_hashes=set(_registered_assets.keys()),
        check_privacy=True,
    )

    filtered = []
    for info in results:
        path = info.path
        ext = Path(path).suffix.lower()
        risk = info.privacy_risk

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

        filtered.append({
            "path": path,
            "hash": info.sha256,
            "category": info.category,
            "ext": ext,
            "privacy_risk": risk,
        })

    return filtered


def mark_registered_asset(path, content_hash, asset_id):
    """Record one locally registered asset for future dedupe."""
    _registered_assets[content_hash] = {
        "path": path,
        "asset_id": asset_id,
    }


def is_already_registered(path, content_hash):
    """Check if the content hash is already known locally or on-chain."""
    known = _registered_assets.get(content_hash)
    if known and known.get("path") == path:
        return True
    asset_id = find_asset_by_hash(content_hash)
    if asset_id:
        mark_registered_asset(path, content_hash, asset_id)
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
    result = get_runtime().signer.register_asset(
        name=name,
        content_hash=content_hash,
        tags=split_csv(tags),
        description=description,
        service_url=service_url,
    )
    ok, detail = tx_status(result)
    if not ok:
        return False, detail

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
            filtered = scan_and_filter(watch_dir)
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
            if is_already_registered(path, content_hash):
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
                mark_registered_asset(path, content_hash, asset_id)
                log.info("REGISTERED %s -> %s", path, asset_id)
                stats["registered"] += 1
                dir_stats["registered"] += 1
            else:
                log.error("Failed to register %s: %s", path, asset_id)
                stats["errors"] += 1

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
    global _total_registered, _total_scanned, _total_cycles, _registered_assets
    try:
        with open(STATE_FILE) as f:
            state = json.load(f)
            _total_registered = state.get("total_registered", 0)
            _total_scanned = state.get("total_scanned", 0)
            _total_cycles = state.get("total_cycles", 0)
            _registered_assets = state.get("registered_assets", {})
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
            "registered_assets": _registered_assets,
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
        get_sdk_agent_scanner()
    except ImportError:
        log.error("oasyce-sdk package not found. Install: pip install -U 'oasyce-sdk>=0.12.0'")
        sys.exit(1)
    except RuntimeError as exc:
        log.error("%s", exc)
        sys.exit(1)

    # Load persistent state
    load_state()

    if not args.dry_run:
        try:
            addr = get_agent_address()
        except RuntimeError as exc:
            log.error("Cannot resolve data-agent identity: %s", exc)
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
