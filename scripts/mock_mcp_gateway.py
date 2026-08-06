#!/usr/bin/env python3
"""Minimal MCP-gateway-compatible mock for agent demos (stdlib only).

Endpoints used by grounded-agent:
  GET  /v1/tools/schema
  POST /v1/servers/{server}/tools/{tool}

Default tool: hr.create_ticket
"""

from __future__ import annotations

import json
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


PORT = 8080


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args) -> None:  # quieter
        print("[%s] %s" % (self.log_date_time_string(), fmt % args))

    def _json(self, code: int, payload: dict | list) -> None:
        raw = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self) -> None:  # noqa: N802
        if self.path.rstrip("/") in ("/health", "/v1/health"):
            self._json(200, {"ok": True, "mock": True})
            return
        if self.path.startswith("/v1/tools/schema"):
            self._json(
                200,
                {
                    "tools": [
                        {
                            "server": "hr",
                            "mcp_tool": "create_ticket",
                            "type": "function",
                            "function": {
                                "name": "create_ticket",
                                "description": "Create an HR support ticket (demo mock)",
                            },
                        }
                    ]
                },
            )
            return
        self._json(404, {"error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else b"{}"
        try:
            payload = json.loads(body.decode("utf-8") or "{}")
        except json.JSONDecodeError:
            payload = {}
        path = self.path.rstrip("/")
        if path == "/v1/servers/hr/tools/create_ticket":
            args = payload.get("args") or {}
            tid = "HR-" + uuid.uuid4().hex[:8].upper()
            self._json(
                200,
                {
                    "ok": True,
                    "ticket_id": tid,
                    "title": args.get("title") or "untitled",
                    "priority": args.get("priority") or "normal",
                    "mock": True,
                },
            )
            return
        self._json(404, {"error": f"unknown tool path {path}"})


def main() -> None:
    server = ThreadingHTTPServer(("0.0.0.0", PORT), Handler)
    print(f"mock MCP gateway on http://127.0.0.1:{PORT} (hr.create_ticket)")
    server.serve_forever()


if __name__ == "__main__":
    main()
