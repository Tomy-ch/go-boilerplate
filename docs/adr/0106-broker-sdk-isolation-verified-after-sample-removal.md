---
status: accepted
date: 2026-08-04
deciders: [maintainers]
supersedes: 0044
tags: [worker, async, dependencies]
---

# ADR-0106: Broker-SDK isolation is verified after sample removal, not by leaving the adapter unwired

## Status

accepted (supersedes [ADR-0044](0044-sqs-adapter-opt-in.md))

## Context

[ADR-0044](0044-sqs-adapter-opt-in.md) kept the SQS reference adapter out of the default `cmd`
wiring so that `aws-sdk-go-v2` would not be linked into the shipped binary.
[`docs/design/worker.md`](../design/worker.md) records the resulting invariant as **E3** — "the
shipped binary contains no broker SDK". Two things have changed since that record was written.

**The premise drifted.** The neutral `ObjectStorage` boundary and its S3 adapter were added after
ADR-0044 was accepted. `github.com/aws/aws-sdk-go-v2/service/s3` — and with it the SDK core
(`aws`, `credentials`, `signer/v4`, `retry`, `transport/http`) and `smithy-go` — is now linked into
the default binary. ADR-0044's context ("every binary — including `serve`, which never consumes a
queue — would link `aws-sdk-go-v2`") was accurate when written and no longer holds. Wiring
`internal/infrastructure/queue/sqs` today adds exactly five packages: `service/sqs`,
`service/sqs/internal/endpoints`, `service/sqs/types`, `aws/protocol/restjson`, and
`smithy-go/encoding/json`.

**The mechanism does not match the binary layout.** This repository ships a single binary with
Cobra subcommands (`serve` / `worker` / `outbox-relay` / …). Linkage is per-binary while roles are
per-subcommand, so "keep `serve` free of the queue SDK" is unachievable by import discipline for as
long as `worker` lives in the same binary. What E3 actually guarantees is therefore not a lean
`serve` — it is that the bundled reference adapter is never wired, and so never exercised end to
end. The engine, the seam, and the adapter are all built and unit-tested, but no message has ever
travelled through them here, and `provideWorkers()` in `internal/di/module/worker.go` is empty. The
adapter's SQS-specific behaviour (visibility-timeout extension, receipt-handle round-trip, attribute
mapping, `ApproximateReceiveCount` parsing) is covered only against a mocked client.

The concern behind ADR-0044 is better stated as **coupling**, not linkage: the repository must not
become structurally dependent on one vendor. Linkage is a poor proxy for that. Replaceability is
preserved by the seam (`worker.Consumer` / `publisher.Publisher`), and declining to import the
adapter adds nothing to it — it only removes the worked example.

## Decision

A broker adapter **may be wired into the default build as part of the removable sample set**. E3 is
replaced by an invariant that measures coupling rather than linkage.

> **E3'**: after `make setup-remove-sample-api`, the repository's coupling is identical to what it
> was before the sample was added.

E3' decomposes into four conditions, each mechanically verifiable:

1. **No core `*.go` references a broker adapter** — checked by `checkNoDanglingReferences` in
   `scripts/setup/verify-sample-removal.mjs`.
2. **No core document references a sample** — core documents describe structure
   (`internal/controller/worker/<name>/`), never a sample's name.
3. **The seam returns to its pre-sample shape** — any sample-driven change to
   `internal/usecase/boundary/worker` is wrapped in a `sample-api:replace` block whose escrow side
   holds the pre-sample form.
4. **Removed dependencies leave `go.mod` / `vendor/`** — `make tidy-lib` runs in the removal chain.

**E1 and E2 are unchanged**: the engine does not import infrastructure, and the engine is green
against the in-memory fake alone.

## Consequences

### Positive Consequences

- The subsystem gains a demonstrated end-to-end path. The adapter and the wiring template stop
  being code that has never run.
- E3' is enforced on every pull request. `.github/workflows/sample-removal-check.yaml` already
  performs a full removal followed by `go build ./...`, `make lint`, and `make test`, so the
  post-removal state is exercised continuously. E3 had no automated enforcement at all — neither
  `depguard` nor `internal/architest/**` carried a rule for it.
- Condition 3's escrow side cannot rot: the same workflow compiles and tests it every run.
- Condition 2 resolves a standing defect where a core design document points at another
  subsystem's sample.

### Negative Consequences

- A fork targeting a non-AWS broker inherits five SQS packages until it runs the sample removal.
- A sample-driven seam change exists in two forms (active and escrow) within one file, which is
  harder to read than a single form.

### Neutral Consequences

- `go.mod` already declares `service/sqs` as a direct dependency because the adapter is built and
  tested. This decision changes what is *linked*, not what is *required*.

## Alternatives Considered

### Keep E3 as written

Forbids any wired example, which reduces the worker subsystem to engine plus seam plus an adapter
nobody may use. Rejected: for a template, a demonstrated path is worth more than a marginally
smaller binary, and the isolation E3 buys is not the isolation the underlying principle wants.

### Split the binary

Separate `main` packages for `serve` and `worker` would make linkage per-role and let E3 stand
literally. Rejected for now: ADR-0044 already rejected build tags for this purpose, and a binary
split is a larger structural change than the goal justifies for a template whose deployment model
is one image with subcommands. It remains available to a fork whose shipping constraints require
it.

### Demonstrate with a broker that needs no SDK

A Postgres-backed pull-ack adapter would exercise the seam without linking any vendor SDK, and
would additionally prove the seam is not SQS-shaped. Rejected: it means building and maintaining a
second adapter for the sample's sake, while E3' already bounds the coupling the sample can
introduce. The reference adapter also stays reusable because the local broker is SQS-compatible.

## Notes

- Supersedes [ADR-0044](0044-sqs-adapter-opt-in.md). Parent decision:
  [ADR-0042](0042-broker-agnostic-worker-scaffold.md). Principle: [ADR-0001](0001-avoid-lock-in.md).
- [ADR-0043](0043-out-of-scope-push-streaming-brokers.md) is unaffected: push-type and
  streaming-log brokers remain outside the worker port. Choosing a pull-ack broker as the outbox
  publish target was never in its scope.
- E3' is stated in [`docs/design/worker.md`](../design/worker.md) and enforced by
  `.github/workflows/sample-removal-check.yaml`.
- Reference: [`internal/infrastructure/queue/sqs/README.md`](../../internal/infrastructure/queue/sqs/README.md).
