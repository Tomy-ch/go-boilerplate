---
status: accepted
date: 2026-08-08
deciders: [maintainers]
supersedes: 0047
tags: [worker, outbox, async, dependencies]
---

# ADR-0051: Measure broker-SDK isolation as coupling, not as linkage

## Status

accepted (supersedes [ADR-0050](0050-sqs-adapter-opt-in.md))

## Context

[ADR-0050](0050-sqs-adapter-opt-in.md) kept the SQS reference adapter out of the default `cmd`
wiring so that `aws-sdk-go-v2` would not be linked into the shipped binary.
[`docs/design/worker.md`](../design/worker.md) records the resulting invariant as **E3** — "the
shipped binary contains no broker SDK". Two things have changed since that record was written.

**The premise drifted.** The neutral `ObjectStorage` boundary and its S3 adapter were added after
ADR-0050 was accepted. `github.com/aws/aws-sdk-go-v2/service/s3` — and with it the SDK core
(`aws`, `credentials`, `signer/v4`, `retry`, `transport/http`) and `smithy-go` — is now linked into
the default binary. ADR-0050's context ("every binary — including `serve`, which never consumes a
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

**The same question arises on the publish side.** `publisher.Publisher` had a single implementation,
an HTTP POST, so no core file referenced a broker adapter. Giving the outbox a queue target means a
core file — the implementation selector in `internal/infrastructure/publisher` — imports one. The
direction of travel differs, but the shape of the question does not, so what follows is stated over
both seams rather than over the worker alone.

The concern behind ADR-0050 is better stated as **coupling**, not linkage: the repository must not
become structurally dependent on one vendor. Linkage is a poor proxy for that. Replaceability is
preserved by the seam (`worker.Consumer` / `publisher.Publisher`), and declining to import the
adapter adds nothing to it — it only removes the worked example.

## Decision

A broker adapter **may be wired into the default build**, on either side of the queue: the consuming
seam (`worker.Consumer` / `worker.FailureHandler`) and the outbox publish seam
(`publisher.Publisher`) are governed alike. E3 is restated to measure coupling rather than linkage.

> **E3**: knowledge of a concrete broker is confined to its adapter package and to the wiring that
> selects it. No core `*.go` and no core document names a broker adapter; core code names only the
> seam.

E3 is mechanically verifiable, because it is a statement about which files name what rather than
about what the linker produced. `checkNoDanglingReferences` in
`scripts/setup/verify-sample-removal.ts` performs the `*.go` half of the check; core documents
describe structure (`internal/controller/worker/<name>/`) and never a concrete adapter.

What carries the vendor is the **wiring**, not the adapter. The adapter package, the local broker
service that stands in for the real one, and the configuration they read name only their own vendor
and are reached from nowhere else — as the object-storage adapter and its local Garage service
already are. What puts a core file in the position of referencing a broker adapter is exactly three
things: the import, the discriminator branch, and the value that selects it. Confining the vendor to
those is what makes replacing it a bounded change, and it is why the adapter is left built and
unit-tested rather than deleted when it is not the one in use.

**E1 and E2 are unchanged**: the engine does not import infrastructure, and the engine is green
against the in-memory fake alone.

## Consequences

### Positive Consequences

- The subsystem gains a demonstrated end-to-end path. The adapter and the wiring template stop
  being code that has never run.
- E3 can be enforced automatically, and is. Its linkage form could not be: neither `depguard` nor
  `internal/architest/**` carried a rule for it, and none would have expressed it.
- Replacing the broker is a bounded change: the import, the discriminator branch, and the value
  that selects it, with nothing to find elsewhere.

### Negative Consequences

- Wiring an adapter links its SDK, so a deployment that targets a different broker carries packages
  it does not use until the wiring is changed.
- The seam's pre-wiring form has to be kept somewhere retrievable if the wiring is meant to be
  reversible, which is harder to read than a single form.

### Neutral Consequences

- `go.mod` already declares `service/sqs` as a direct dependency because the adapter is built and
  tested. This decision changes what is *linked*, not what is *required*.

## Alternatives Considered

### Keep E3 in its linkage form

Forbids any wired example, which reduces the worker subsystem to engine plus seam plus an adapter
nobody may use. Rejected: a demonstrated path is worth more than a marginally smaller binary, and
the isolation a linkage rule buys is not the isolation the underlying principle wants.

### Split the binary

Separate `main` packages for `serve` and `worker` would make linkage per-role and let the linkage
form stand literally. Rejected for now: ADR-0050 already rejected build tags for this purpose, and a binary
split is a larger structural change than the goal justifies for a deployment model that is one image
with subcommands. It remains available where shipping constraints require it.

### Demonstrate with a broker that needs no SDK

A Postgres-backed pull-ack adapter would exercise the seam without linking any vendor SDK, and
would additionally prove the seam is not SQS-shaped. Rejected: it means building and maintaining a
second adapter for the example's sake, while E3 already bounds the coupling a wired example can
introduce. The reference adapter also stays reusable because the local broker is SQS-compatible.

## Notes

- Supersedes [ADR-0050](0050-sqs-adapter-opt-in.md). Parent decision:
  [ADR-0048](0048-broker-agnostic-worker-scaffold.md). Principle: [ADR-0001](0001-avoid-lock-in.md).
- [ADR-0049](0049-out-of-scope-push-streaming-brokers.md) is unaffected: push-type and
  streaming-log brokers remain outside the worker port. Choosing a pull-ack broker as the outbox
  publish target was never in its scope.
- E3 is stated in [`docs/design/worker.md`](../design/worker.md).
- Reference: [`internal/infrastructure/queue/sqs/README.md`](../../internal/infrastructure/queue/sqs/README.md)
  for the adapter, [`internal/infrastructure/publisher/README.md`](../../internal/infrastructure/publisher/README.md)
  for the discriminator that selects it.
