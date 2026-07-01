# logging

English | [日本語](README.ja.md)

HTTP request/response structured logging middleware with trace context.

Skips ops endpoints (`/health`, `/metrics`, etc.).

## Role

Per-request access logging is a cross-cutting concern that every endpoint shares, so embedding it in handlers would scatter and duplicate it. Isolating it as a middleware produces one consistent structured log line per request — correlated with the trace context — while letting handlers stay focused on business logic. High-volume ops endpoints are skipped to keep the logs signal-rich.
