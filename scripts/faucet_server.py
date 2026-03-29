#!/usr/bin/env python3
"""Minimal HTTP faucet for Oasyce testnet."""
import http.server
import socketserver
import subprocess
import json
import time
import os
import re
import urllib.parse

PORT = int(os.environ.get("FAUCET_PORT", "8080"))
AMOUNT = int(os.environ.get("FAUCET_AMOUNT", "100"))  # OAS
CHAIN_ID = os.environ.get("CHAIN_ID", "oasyce-testnet-1")
NODE = os.environ.get("NODE", "tcp://localhost:26657")
HOME = os.environ.get("OASYCE_HOME", "/home/oasyce/.oasyced")
FAUCET_KEY = os.environ.get("FAUCET_KEY", "faucet")
RATE_SECONDS = 3600  # 1 hour per address
RATE_FILE = os.environ.get("FAUCET_RATE_FILE", "/tmp/faucet_rate.json")

rate_limit = {}  # address -> last_request_timestamp


def load_rate_limit():
    """Load rate limit state from disk."""
    global rate_limit
    try:
        with open(RATE_FILE) as f:
            rate_limit = json.load(f)
    except (FileNotFoundError, json.JSONDecodeError):
        rate_limit = {}


def save_rate_limit():
    """Persist rate limit state to disk."""
    try:
        with open(RATE_FILE, "w") as f:
            json.dump(rate_limit, f)
    except OSError:
        pass


def run_faucet_send(address, retries=2):
    amount_uoas = AMOUNT * 1_000_000
    faucet_addr = subprocess.check_output([
        "oasyced", "keys", "show", FAUCET_KEY, "-a",
        "--keyring-backend", "test", "--home", HOME
    ], text=True).strip()

    cmd = [
        "oasyced", "tx", "send", faucet_addr, address,
        f"{amount_uoas}uoas",
        "--from", FAUCET_KEY,
        "--fees", "10000uoas", "--yes",
        "--keyring-backend", "test",
        "--chain-id", CHAIN_ID,
        "--node", NODE,
        "--home", HOME,
        "--output", "json",
    ]

    for attempt in range(1, retries + 1):
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        output = (result.stdout or result.stderr).strip()
        if result.returncode != 0:
            return False, output
        try:
            payload = json.loads(output)
        except json.JSONDecodeError:
            return False, output or "non-json faucet tx output"
        code = int(payload.get("code", 0) or 0)
        if code == 0:
            return True, payload
        if code == 19 and attempt < retries:
            time.sleep(6)
            continue
        return False, payload.get("raw_log", output)

    return False, "max retries exceeded"


class ReuseAddrServer(http.server.HTTPServer):
    allow_reuse_address = True


class FaucetHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        if parsed.path != "/faucet":
            self.send_json(404, {"error": "not found"})
            return

        params = urllib.parse.parse_qs(parsed.query)
        address = params.get("address", [None])[0]

        if not address:
            self.send_json(400, {"error": "missing ?address= parameter"})
            return

        if not re.match(r"^oasyce1[a-z0-9]{38}$", address):
            self.send_json(400, {"error": "invalid address format"})
            return

        now = time.time()
        last = rate_limit.get(address, 0)
        if now - last < RATE_SECONDS:
            wait = int(RATE_SECONDS - (now - last))
            self.send_json(429, {"error": f"rate limited, retry in {wait}s"})
            return

        try:
            ok, payload = run_faucet_send(address)
            if not ok:
                self.send_json(500, {"error": str(payload)})
                return

            rate_limit[address] = now
            save_rate_limit()
            self.send_json(200, {
                "status": "ok",
                "amount": f"{AMOUNT} OAS",
                "to": address,
                "txhash": payload.get("txhash", ""),
            })
        except Exception as e:
            self.send_json(500, {"error": str(e)})

    def send_json(self, code, data):
        body = json.dumps(data).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        print(f"[faucet] {args[0]}")


if __name__ == "__main__":
    load_rate_limit()
    print(f"Faucet listening on :{PORT} ({AMOUNT} OAS per request)")
    httpd = ReuseAddrServer(("0.0.0.0", PORT), FaucetHandler)
    httpd.serve_forever()
