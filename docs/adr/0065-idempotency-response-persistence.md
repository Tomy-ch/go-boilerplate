---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [idempotency, privacy]
---

# ADR-0065: Persist the response body as JSON to enable deterministic replay (accepted PII tradeoff)

## Status

accepted

## Context

When a completed idempotency key is retried, the subsystem must return a response that is
identical to the original — the caller cannot distinguish a replay from a fresh success.
The only reliable way to achieve this is to store the original response and return it
verbatim; reconstructing it from the business state on each retry is fragile because
business state may have changed between the original call and the retry.

The alternative is to store only the success status code and replay a synthesized empty
body, but this breaks callers that depend on the response payload (e.g., to extract the
created resource's ID).

Persisting the full response body introduces a privacy tradeoff: if the response DTO
contains personally identifiable information (PII), that PII is stored in the
`idempotency_keys` table and will appear in database dumps, backups, and read replicas for
up to 24 hours (the fixed TTL).

## Decision

The `Complete` step serializes the business function result `T` to JSON and stores it in
the `response_payload` column of `idempotency_keys`. On replay, `Run[T]` deserializes this
payload and returns it directly without calling the business function. The PII tradeoff is
accepted: it is mitigated by the 24-hour TTL (after which GC removes the row) and by
ensuring that database access controls and backup encryption cover the `idempotency_keys`
table with the same rigor as any other table containing sensitive data.

## Consequences

### Positive Consequences

- Replay is deterministic and payload-complete; callers receive the same body on every
  retry of a completed operation.
- No business-state reconstruction logic is required in the replay path.
- The replay path is a simple JSON unmarshal from a stored column — straightforward to
  reason about and test.

### Negative Consequences

- PII in response DTOs is persisted for up to 24 hours beyond the request lifecycle.
  Database dumps and backups taken within the TTL window will contain this PII.
- The `response_payload` column grows with the size of the serialized DTO; large response
  bodies increase row size.
- Schema evolution of the DTO can cause deserialization failures on replay if the stored
  JSON no longer matches the current struct.

## Alternatives Considered

### Store status code only, reconstruct body at replay time

Replay a synthesized response (e.g., an empty body with the stored status code) rather
than the stored payload. Rejected because callers that depend on the response body — most
notably to retrieve the ID of a created resource — would receive a broken replay.

### Encrypt the response payload at rest

Store the payload encrypted so that database dumps do not expose PII in plaintext. Not
adopted as a default because key management adds operational complexity; integrators with
strict PII requirements can add column-level encryption as a project-specific extension
without changing the subsystem contract.

## Notes

- Source: [`docs/design/idempotency.md`](../design/idempotency.md) §4 (operational notes, "PII
  caveat").
- The `response_payload` column type and `Complete` SQL are defined in
  `database/dml/system_cqrs/idempotency/`.
- The 24-hour TTL that bounds PII retention is decided in ADR-0064.
- Integrators with PII-bearing DTOs should ensure their database backup encryption and
  access controls cover the `idempotency_keys` table.
