import http.client
import importlib.util
import json
import socket
import threading
import unittest
from pathlib import Path
from unittest import mock


def load_script_module(name, relative_path):
    script_path = Path(__file__).resolve().parents[2] / relative_path
    spec = importlib.util.spec_from_file_location(name, script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class FakeResponse:
    def __init__(self, payload, status=200):
        self.payload = payload
        self.status = status

    def read(self):
        return json.dumps(self.payload).encode()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False


class ClaudeProxyTests(unittest.TestCase):
    def load_module(self):
        module = load_script_module(
            f"claude_proxy_under_test_{id(self)}",
            "scripts/claude_proxy.py",
        )
        module.RELAY_URL = "https://relay.example.com/v1/messages"
        module.RELAY_KEY = "sk-test"
        module.DEFAULT_MODEL = "claude-opus-4-6"
        module.health_cache.update({
            "checked_at": 0.0,
            "ok": False,
            "http_status": 503,
            "error": "not checked",
        })
        module.stats["total_calls"] = 0
        module.stats["total_errors"] = 0
        return module

    def start_server(self, module):
        server = module.HTTPServer(("127.0.0.1", 0), module.ProxyHandler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        return server, thread

    def request(self, server, method, path):
        conn = http.client.HTTPConnection("127.0.0.1", server.server_port, timeout=5)
        conn.request(method, path)
        resp = conn.getresponse()
        body = resp.read()
        conn.close()
        return resp, body

    def test_probe_upstream_uses_cache(self):
        module = self.load_module()
        with mock.patch.object(module, "urlopen", return_value=FakeResponse({"content": [{"type": "text", "text": "pong"}]})) as mock_urlopen:
            first = module.probe_upstream()
            second = module.probe_upstream()

        self.assertTrue(first["ok"])
        self.assertEqual(second["http_status"], 200)
        self.assertEqual(mock_urlopen.call_count, 1)

    def test_health_endpoint_returns_503_when_upstream_fails(self):
        module = self.load_module()
        error = module.HTTPError(
            module.RELAY_URL,
            503,
            "service unavailable",
            hdrs=None,
            fp=None,
        )
        error.read = lambda: json.dumps({"error": {"message": "No available accounts"}}).encode()
        with mock.patch.object(module, "urlopen", side_effect=error):
            server, thread = self.start_server(module)
            try:
                resp, body = self.request(server, "GET", "/health")
            finally:
                server.shutdown()
                server.server_close()
                thread.join(timeout=5)

        payload = json.loads(body.decode())
        self.assertEqual(resp.status, 503)
        self.assertEqual(payload["status"], "degraded")
        self.assertFalse(payload["upstream_ok"])
        self.assertEqual(payload["last_error"], "No available accounts")

    def test_head_v1_chat_returns_probe_status(self):
        module = self.load_module()
        with mock.patch.object(module, "urlopen", return_value=FakeResponse({"content": [{"type": "text", "text": "pong"}]})):
            server, thread = self.start_server(module)
            try:
                resp, body = self.request(server, "HEAD", "/v1/chat")
            finally:
                server.shutdown()
                server.server_close()
                thread.join(timeout=5)

        self.assertEqual(resp.status, 200)
        self.assertEqual(body, b"")
        self.assertEqual(resp.getheader("X-Upstream-Ok"), "true")


if __name__ == "__main__":
    unittest.main()
