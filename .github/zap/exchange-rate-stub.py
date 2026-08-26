#!/usr/bin/env python3
"""Stand in for the external exchange-rate service during the DAST scan.

The sample gateway points at a reserved `.example.com` domain that never resolves, so
`GET /v1/exchange-rates` answers 5xx in every environment. ZAP reports that (rule 100000-2),
and `.github/zap/rules.tsv` deliberately does not ignore server errors — an endpoint that
always fails would otherwise train the scan to treat 5xx as normal. Serving a fixed rate here
keeps the rule meaningful: a 5xx from this path becomes a real finding again.

The shape is the one `internal/infrastructure/webapi/exchangerate` expects:
`GET /rates?base=<b>&quote=<q>` returning `{"rate": <positive number>, "date": "<YYYY-MM-DD>"}`.
"""

import json
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse

PORT = 9401
BODY = json.dumps({"rate": "150.00", "date": "2026-01-01"}).encode()


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802 - name fixed by BaseHTTPRequestHandler
        if urlparse(self.path).path != "/rates":
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(BODY)))
        self.end_headers()
        self.wfile.write(BODY)

    def log_message(self, *args: object) -> None:
        """Silence per-request logging; the scan makes many calls and the job log is shared."""


if __name__ == "__main__":
    HTTPServer(("127.0.0.1", PORT), Handler).serve_forever()
