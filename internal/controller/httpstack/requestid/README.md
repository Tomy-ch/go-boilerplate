# requestid

English | [日本語](README.ja.md)

Generates unique X-Request-ID header for each request.

## Role

Correlating logs, traces, and client-reported failures requires a stable identifier that spans a single request end to end. Generating it once in a middleware guarantees every request carries an ID before any handler or log runs, giving the whole stack a shared correlation key without each layer inventing its own.
