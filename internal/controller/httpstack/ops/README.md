# ops

English | [日本語](README.ja.md)

Identifies operational/infrastructure endpoints.

Used by logging middleware to skip ops endpoints.

## Role

Several middlewares need to treat operational/infrastructure endpoints (health, metrics, etc.) differently from business endpoints — skipping logging, validation, and authentication for them. Centralizing the definition of "what counts as an ops endpoint" here gives every consumer one source of truth, so the classification cannot drift between the places that rely on it.
