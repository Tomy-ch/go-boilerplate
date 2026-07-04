# Architecture Decisions

> **Moved.** This monolithic decisions document has been split into per-file
> **Architecture Decision Records (ADR)**. See **[`docs/adr/`](adr/README.md)**
> for the full, indexed decision log (one immutable record per decision).

The technology-rationale content that used to live here (onion architecture, OpenAPI-first,
SQL-first, sqlc, Echo, Fx, the worker scaffold, library-selection policy, and the
observability gating) now lives as individual ADRs:

- Start at the [ADR log](adr/README.md) for the full list and ordering.
- The formerly-inline **direct dependency table** is a living inventory, not a decision, and
  moved to [`docs/reference/dependencies.md`](reference/dependencies.md) (where the
  previously-missing `net/http/otelhttp` and `otel/sdk/log` entries are now recorded).

Why the split: a single mutable file lost decision history on in-place edits and mixed
immutable decisions with a `go.mod`-tracking dependency table. Per-file ADRs let a fork
supersede one decision by adding one file, and keep the dependency inventory separate from
the immutable records. See [ADR-0000](adr/0000-record-architecture-decisions.md).
