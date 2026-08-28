# redaction

Removes credential values from a request URI and its query parameters before either is written to a log.

## Role

The stream ticket travels as a query parameter (ADR-0074 (query-ticket-stream-authentication)), so
the request URI itself carries a credential. Every place that logs a request — the access log, the
error handler, the panic recovery — must pass the URI and the query map through a `Redactor` first.
The rule is enforced in the HTTP stack, not left to handlers.

## Where the names come from

`FromSpec(spec)` reads the OpenAPI `securitySchemes` and collects the parameter name of every
`apiKey` scheme delivered `in: query`. The spec is the only place the name `ticket` is written down:
adding another query-credential scheme automatically extends the redaction set, and there is no
second list in Go to keep in step (ADR-0016 (spec-driven-request-validation)).

## API

- `New(names []string) Redactor` — redact the given parameter names. The zero value redacts nothing.
- `FromSpec(spec *openapi3.T) Redactor` — derive the names from the spec.
- `SecretQueryParamNames(spec) []string` — the derived names, sorted, for tests and diagnostics.
- `Redactor.URI(raw string) string` — replace the value of each secret pair in the raw request URI
  with `[REDACTED]`, preserving pair order and encoding. A pair whose name cannot be decoded is
  treated as secret.
- `Redactor.QueryParams(map[string][]string) map[string][]string` — return a copy with every
  value of a secret name replaced; the input map is not modified.

## Wiring

`internal/di/module/core.RedactionModule()` builds the `Redactor` from the same `*openapi3.T` the
validator uses, and the three log paths receive it through DI:

| Path | Where it is applied |
| --- | --- |
| access log (`httpstack/logging`) | request and response fields |
| error handler (`httpstack/errorhandler`) | `Policies.Redact` → `server.BuildHTTPRequestLogInput` |
| panic recovery (`httpstack/recovery`) | `server.BuildHTTPRequestLogInput` |

The three paths are deliberately separate: the access log runs inside the OpenAPI validator, so a
request refused at authentication never reaches it and is logged by the error handler instead.

## Test Strategy

- Unit tests pin the URI rewriting (order, encoding, repeated names, undecodable names) and the
  query-map copy semantics.
- `internal/integration` drives a real middleware chain with a `?ticket=` request and asserts that
  no log entry contains the raw value while `[REDACTED]` does appear — proving the value was
  redacted rather than the log suppressed.
