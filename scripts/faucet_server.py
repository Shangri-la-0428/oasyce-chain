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

        amount_uoas = AMOUNT * 1_000_000
        try:
            faucet_addr = subprocess.check_output([
                "oasyced", "keys", "show", FAUCET_KEY, "-a",
                "--keyring-backend", "test", "--home", HOME
            ], text=True).strip()

            result = subprocess.run([
                "oasyced", "tx", "send", faucet_addr, address,
                f"{amount_uoas}uoas",
                "--fees", "500uoas", "--yes",
                "--keyring-backend", "test",
                "--chain-id", CHAIN_ID,
                "--home", HOME,
            ], capture_output=True, text=True, timeout=30)

            if result.returncode != 0:
                self.send_json(500, {"error": result.stderr.strip()})
                return

            rate_limit[address] = now
            save_rate_limit()
            self.send_json(200, {
                "status": "ok",
                "amount": f"{AMOUNT} OAS",
                "to": address,
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
