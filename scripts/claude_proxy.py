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
BIND_HOST = os.environ.get("PROXY_HOST", "127.0.0.1")
DEFAULT_MODEL = os.environ.get("CLAUDE_MODEL", "claude-sonnet-4-20250514")
DEFAULT_MAX_TOKENS = int(os.environ.get("DEFAULT_MAX_TOKENS", "1024"))
HEALTH_CACHE_TTL_S = int(os.environ.get("CLAUDE_HEALTH_CACHE_TTL_S", "60"))
HEALTH_CHECK_TIMEOUT_S = int(os.environ.get("CLAUDE_HEALTH_CHECK_TIMEOUT_S", "20"))

# ── Stats (informational only, not for billing) ────────────────────────────
lock = threading.Lock()
stats = {
    "total_calls": 0,
    "total_errors": 0,
    "started_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
}
health_cache = {
    "checked_at": 0.0,
    "ok": False,
    "http_status": 503,
    "error": "not checked",
}


def _extract_error_message(payload):
    if isinstance(payload, dict):
        nested = payload.get("error")
        if isinstance(nested, dict):
            return nested.get("message") or nested.get("type") or json.dumps(nested, ensure_ascii=False)
        if isinstance(nested, str):
            return nested
        message = payload.get("message")
        if isinstance(message, str):
            return message
    return json.dumps(payload, ensure_ascii=False)


def relay_request(payload, timeout=120):
    relay_body = json.dumps(payload).encode()

    req = Request(RELAY_URL, data=relay_body, method="POST")
    req.add_header("Content-Type", "application/json")
    req.add_header("x-api-key", RELAY_KEY)
    req.add_header("anthropic-version", "2023-06-01")

    try:
        with urlopen(req, timeout=timeout) as resp:
            return True, resp.status, json.loads(resp.read())
    except HTTPError as e:
        error_body = e.read().decode(errors="replace")
        try:
            upstream_err = json.loads(error_body)
        except Exception:
            upstream_err = {"error": error_body}
        return False, e.code, upstream_err
    except (URLError, Exception) as e:
        return False, 502, {"error": f"upstream unreachable: {e}"}


def probe_upstream(force=False):
    now = time.monotonic()

    with lock:
        if (
            not force
            and health_cache["checked_at"]
            and now - health_cache["checked_at"] < HEALTH_CACHE_TTL_S
        ):
            return dict(health_cache)

    if not RELAY_URL or not RELAY_KEY:
        result = {
            "checked_at": now,
            "ok": False,
            "http_status": 500,
            "error": "proxy not configured",
        }
    else:
        ok, http_status, response = relay_request({
            "model": DEFAULT_MODEL,
            "max_tokens": 1,
            "messages": [{"role": "user", "content": "health check"}],
        }, timeout=HEALTH_CHECK_TIMEOUT_S)
        result = {
            "checked_at": now,
            "ok": ok,
            "http_status": 200 if ok else http_status,
            "error": "" if ok else _extract_error_message(response),
        }

    with lock:
        health_cache.update(result)
        return dict(health_cache)


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

    def _send_status(self, code, extra_headers=None):
        self.send_response(code)
        self.send_header("Access-Control-Allow-Origin", "*")
        if extra_headers:
            for key, value in extra_headers.items():
                self.send_header(key, value)
        self.end_headers()

    def _health_payload(self):
        upstream = probe_upstream()
        with lock:
            total_calls = stats["total_calls"]
            total_errors = stats["total_errors"]
            uptime_since = stats["started_at"]
        return upstream["http_status"], {
            "status": "active" if upstream["ok"] else ("misconfigured" if upstream["http_status"] == 500 else "degraded"),
            "model": DEFAULT_MODEL,
            "upstream_ok": upstream["ok"],
            "last_error": upstream["error"],
            "total_calls": total_calls,
            "total_errors": total_errors,
            "uptime_since": uptime_since,
        }

    def do_OPTIONS(self):
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type, Authorization")
        self.end_headers()

    def do_HEAD(self):
        if self.path == "/health":
            code, payload = self._health_payload()
            self._send_status(code, {
                "Content-Type": "application/json",
                "X-Proxy-Status": payload["status"],
                "X-Upstream-Ok": str(payload["upstream_ok"]).lower(),
            })
            return

        if self.path in ("/v1/chat", "/v1/chat/completions"):
            code, payload = self._health_payload()
            self._send_status(code, {
                "Allow": "HEAD, OPTIONS, POST",
                "X-Proxy-Status": payload["status"],
                "X-Upstream-Ok": str(payload["upstream_ok"]).lower(),
            })
            return

        self._send_status(200)

    def do_GET(self):
        if self.path == "/health":
            code, payload = self._health_payload()
            self._send_json(code, payload)
        else:
            self._send_json(200, {
                "service": "Oasyce Claude AI Proxy",
                "endpoint": "POST /v1/chat",
                "health": "GET/HEAD /health",
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

        ok, code, result = relay_request(relay_payload, timeout=120)
        if not ok:
            with lock:
                stats["total_errors"] += 1
            self._send_json(code, result)
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
    print(f"  Listen:   {BIND_HOST}:{PORT}")
    print(f"  Model:    {DEFAULT_MODEL}")
    print(f"  Relay:    {RELAY_URL[:40]}..." if len(RELAY_URL) > 40 else f"  Relay:    {RELAY_URL}")
    print(f"  Endpoint: POST /v1/chat")
    print(f"  Health:   GET/HEAD /health")
    print()

    server = HTTPServer((BIND_HOST, PORT), ProxyHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")


if __name__ == "__main__":
    main()
