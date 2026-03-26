#!/usr/bin/env python3
"""
Oasyce Capability Proxy — Claude AI

Pure forwarding proxy: accepts agent requests, forwards to upstream relay
(Anthropic Messages API format), returns the response. No billing logic —
the relay's own balance limit handles budget enforcement.

Usage:
  export CLAUDE_RELAY_URL="https://relay.example.com/v1/messages"
  export CLAUDE_RELAY_KEY="sk-xxx"
  python3 claude_proxy.py       # listens on :8090
"""

import os
import json
import time
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError

# ── Config ──────────────────────────────────────────────────────────────────
RELAY_URL = os.environ.get("CLAUDE_RELAY_URL", "")
RELAY_KEY = os.environ.get("CLAUDE_RELAY_KEY", "")
PORT = int(os.environ.get("PROXY_PORT", "8090"))
DEFAULT_MODEL = os.environ.get("CLAUDE_MODEL", "claude-sonnet-4-20250514")
DEFAULT_MAX_TOKENS = int(os.environ.get("DEFAULT_MAX_TOKENS", "1024"))

# ── Stats (informational only, not for billing) ────────────────────────────
lock = threading.Lock()
stats = {
    "total_calls": 0,
    "total_errors": 0,
    "started_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
}


class ProxyHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        print(f"[{time.strftime('%H:%M:%S')}] {args[0]}")

    def _send_json(self, code, data):
        body = json.dumps(data, ensure_ascii=False).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(body)

    def do_OPTIONS(self):
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type, Authorization")
        self.end_headers()

    def do_GET(self):
        if self.path == "/health":
            self._send_json(200, {
                "status": "active",
                "model": DEFAULT_MODEL,
                "total_calls": stats["total_calls"],
                "total_errors": stats["total_errors"],
                "uptime_since": stats["started_at"],
            })
        else:
            self._send_json(200, {
                "service": "Oasyce Claude AI Proxy",
                "endpoint": "POST /v1/chat",
                "health": "GET /health",
                "body_format": {"prompt": "your message here"},
            })

    def do_POST(self):
        if self.path not in ("/v1/chat", "/v1/chat/completions"):
            self._send_json(404, {"error": "use POST /v1/chat"})
            return

        if not RELAY_URL or not RELAY_KEY:
            self._send_json(500, {"error": "proxy not configured"})
            return

        # Parse request body
        try:
            length = int(self.headers.get("Content-Length", 0))
            raw = self.rfile.read(length) if length > 0 else b"{}"
            body = json.loads(raw)
        except Exception:
            self._send_json(400, {"error": "invalid JSON body"})
            return

        # Accept both simple {"prompt": "..."} and OpenAI-compatible {"messages": [...]}
        messages = body.get("messages")
        if not messages:
            prompt = body.get("prompt") or body.get("message") or body.get("content", "")
            if not prompt:
                self._send_json(400, {
                    "error": "missing input",
                    "formats": [
                        {"prompt": "your message"},
                        {"messages": [{"role": "user", "content": "your message"}]},
                    ],
                })
                return
            messages = [{"role": "user", "content": prompt}]

        # Build relay request (Anthropic Messages API format)
        relay_payload = {
            "model": body.get("model", DEFAULT_MODEL),
            "max_tokens": body.get("max_tokens", DEFAULT_MAX_TOKENS),
            "messages": messages,
        }
        # Pass through optional system prompt
        if body.get("system"):
            relay_payload["system"] = body["system"]

        relay_body = json.dumps(relay_payload).encode()

        req = Request(RELAY_URL, data=relay_body, method="POST")
        req.add_header("Content-Type", "application/json")
        req.add_header("x-api-key", RELAY_KEY)
        req.add_header("anthropic-version", "2023-06-01")

        # Forward to relay
        try:
            with urlopen(req, timeout=120) as resp:
                result = json.loads(resp.read())
        except HTTPError as e:
            error_body = e.read().decode(errors="replace")
            with lock:
                stats["total_errors"] += 1
            try:
                upstream_err = json.loads(error_body)
            except Exception:
                upstream_err = {"error": error_body}
            self._send_json(e.code, upstream_err)
            return
        except (URLError, Exception) as e:
            with lock:
                stats["total_errors"] += 1
            self._send_json(502, {"error": f"upstream unreachable: {e}"})
            return

        with lock:
            stats["total_calls"] += 1

        # Normalize: extract text from Anthropic response for easy consumption
        text = ""
        if result.get("content"):
            for block in result["content"]:
                if block.get("type") == "text":
                    text += block["text"]

        self._send_json(200, {
            "text": text,
            "model": result.get("model", DEFAULT_MODEL),
            "usage": result.get("usage", {}),
            "raw": result,  # full Anthropic response for advanced consumers
        })


def main():
    if not RELAY_URL:
        print("WARNING: CLAUDE_RELAY_URL not set")
    if not RELAY_KEY:
        print("WARNING: CLAUDE_RELAY_KEY not set")

    print(f"Oasyce Claude AI Proxy")
    print(f"  Port:     :{PORT}")
    print(f"  Model:    {DEFAULT_MODEL}")
    print(f"  Relay:    {RELAY_URL[:40]}..." if len(RELAY_URL) > 40 else f"  Relay:    {RELAY_URL}")
    print(f"  Endpoint: POST /v1/chat")
    print(f"  Health:   GET /health")
    print()

    server = HTTPServer(("0.0.0.0", PORT), ProxyHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")


if __name__ == "__main__":
    main()
