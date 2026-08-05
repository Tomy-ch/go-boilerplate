# withdrawal-archive worker (sample)

English | [日本語](README.ja.md)

A worked example of the **consuming end** of the outbox path: a user withdraws, the outbox emits
`user.withdrawn.v1` in the same transaction, the relay publishes it to the broker, and this worker
consumes it and writes a withdrawal record to object storage.

It is part of the removable sample set — `make setup-remove-sample-api` deletes this package and the
two registration lines that reach it, leaving `provideWorkers()` empty again.

## What it archives, and why that

The archived object is the **event payload verbatim** (`{userId, deletedAt}`), stored at
`withdrawals/{userID}.json`.

Archiving the withdrawn user's full record was the obvious alternative and does not work: withdrawn
users are filtered out by `deleted_at IS NULL` in the user lookup query, so the consumer cannot read
them back. Archiving the payload avoids that entirely, and it is the more defensible design anyway —
the record is "who withdrew, and when", which stays meaningful as an audit trail after `user-purge`
physically deletes the user, and carries no personal data that would need purging in turn.

## Idempotency

At-least-once delivery means this handler will sometimes run twice for one withdrawal. Rather than
detect the repeat, **the operation is made idempotent**: the key is derived from the user ID alone and
the body is the payload unmodified, so a second run overwrites the object with identical bytes.

That is why nothing here consults an idempotency store. The HTTP idempotency subsystem is built for
replaying HTTP responses — its records carry method, path, and response status — and borrowing it
here would mean filling those with placeholder values.

The same property covers redelivery caused by a slow handler: if a run outlives
`CONSUMER_QUEUE_VISIBILITY_TIMEOUT` (30s by default, with no `WORKER_EXTEND_INTERVAL` heartbeat) the
message is redelivered and processed again, and the result is unchanged.

## Message selection

One queue carries every event the outbox emits, so the handler first checks the `event_type`
attribute and returns success for anything else — the engine then acks it and the message leaves the
queue. Treating a foreign event as a permanent failure would route every purchase event to the DLQ.
A message with no `event_type` attribute at all is treated the same way, so messages published
before the attribute existed do not fill the DLQ either.

This works because the sample is the only consumer of `gobp-events`. A deployment with several
consumers on one queue wants a subscription filter (or a queue per event type) instead, so that each
consumer only receives what it handles.

## Error classification

| Situation | Classification | Effect |
| --- | --- | --- |
| Other / missing `event_type` | success | acked, no archive written |
| Payload cannot be decoded | `ErrPermanent` | routed to the DLQ, then acked |
| Usecase rejects the input (`ErrValidation`) | `ErrPermanent` | routed to the DLQ, then acked |
| Storage unavailable | unclassified | engine default (retryable) → redelivered with backoff |

Note that the broker also has a redrive policy (`maxReceiveCount = 5` in
`docker/elasticmq/elasticmq.conf`), so a message that keeps failing for a *retryable* reason still
ends up in the DLQ once it runs out of receives. That is a second, broker-level net below the
`FailureHandler`, not a contradiction of the table above.

## Running it end to end

Everything below assumes a DB slot has been acquired in a worktree (`make slot-acquire`); on the main
checkout the default ports apply.

```bash
# 1. Shared infra + API (elasticmq and garage come up with it)
make serve

# 2. In a second terminal: the relay that publishes outbox rows to the queue
make outbox-relay

# 3. In a third terminal: this worker
make worker NAME=withdrawal-archive

# 4. Withdraw a user (see docs/get-started for obtaining a token from the mock auth server)
curl -X DELETE "http://localhost:${API_HOST_PORT:-8080}/v1/users/<userId>" \
  -H "Authorization: Bearer <token>"
```

What to watch, in order:

1. `api_server` — the withdrawal request completes; the outbox row is written in the same transaction
2. `outbox-relay` — a publish log for `user.withdrawn.v1`, and the row moves to `published`
3. this worker — the handler runs, with the trace continued from the publisher's `traceparent`
4. object storage — the archived object appears:

   ```bash
   docker run --rm --network host amazon/aws-cli s3 ls s3://gobp-local/withdrawals/ \
     --endpoint-url http://localhost:3900 --region us-east-1
   ```

   (credentials come from `OBJECT_STORAGE_ACCESS_KEY_ID` / `..._SECRET_ACCESS_KEY` in `env/.env`)

To see the idempotency property, publish the same message again — the object is rewritten with the
same bytes. To see the DLQ path, send a body that is not valid JSON with an `event_type` attribute of
`user.withdrawn.v1`; it lands in `gobp-events-dlq` with `failure_reason=permanent`.

> The queue is shared across every checkout and cannot be isolated per worktree
> (see [`docs/maintenance/db-worktree-pool.md`](../../../../docs/maintenance/db-worktree-pool.md)).
> Running this worker in two worktrees at once means either one may take the message.

## Structure

| File | Contents |
| --- | --- |
| `withdrawal_archive_worker.go` | `worker.Worker` — bundles name / consumer / handler / failure handler |
| `withdrawal_archive_handler.go` | `worker.Handler` — selection, decoding, error classification |

The broker adapters themselves are built in `internal/di/module/withdrawalarchive.go`, not here: the
controller layer must not import infrastructure, so this package never learns which broker it is
consuming from.
