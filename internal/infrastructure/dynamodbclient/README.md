# infrastructure/dynamodbclient

The substrate shared by every Realtime Delivery adapter that talks to a DynamoDB-compatible store:
`eventlog/dynamodb`, `streamticket/dynamodb`, `instancelease/dynamodb`, the `realtime-init`
initializer and the contract-test kit. It has no boundary interface of its own — like
[`awsclient`](../awsclient/README.md) it exists so that one connection shape and one retry policy
are defined once.

## What it fixes

| Concern | Definition |
| --- | --- |
| Credentials | `awsclient.Resolve` — empty key pair ⇒ SDK default chain (IAM role), both set ⇒ static; one of the two ⇒ startup error |
| Endpoint | `Config.Endpoint` non-empty ⇒ `BaseEndpoint` (DynamoDB Local); empty ⇒ SDK default resolution (AWS DynamoDB) |
| Retry | `MaxAttempts = 3`, standard mode, `MaxBackoff = 2s`. The SDK's own defaults happen to be similar, but the numbers are declared here so an unreachable store surfaces as `ErrUnavailable` within seconds instead of retrying without bound; the caller (503 + `Retry-After`, the outbox backoff) decides what to do next |
| Call timeout | `CallTimeout = 10s`, applied to the whole call (retries included) by an Initialize-step middleware. Retry counts *failed attempts*, so it never fires on an attempt that simply does not return; nothing else bounds one either — the injected HTTP client sets no `Timeout`, and the dialer swapped in for the SSRF guard carries none. The SSE stream path is excluded from the request deadline budget, so no caller above supplies a bound. Matched to the SSE write deadline: a store call has no reason to outlast a single write |
| Error normalization | `Normalize(err, op)` — context cancellation ⇒ `apperror.ErrCanceled`, everything else ⇒ `apperror.ErrUnavailable`. Coarse on purpose, as in [`objectstorage`](../objectstorage/README.md): callers branch on the sentinel, never on an SDK type. The one failure a caller *does* need to tell apart — a conditional write refused — is exposed as `IsConditionalCheckFailed` and handled before normalizing |
| Table creation | `EnsureTable(ctx, client, TableSpec)` — idempotent: `ResourceInUseException` is success, then `TableExists` waiter, then TTL enabled only when `DescribeTimeToLive` does not already report it on the same attribute. `TableSpec` is what each adapter package publishes; the numbers of tables and their names live in `internal/cli/realtimeinit` |

Why retry is not an `awsclient` knob: that package's README keeps service-specific behaviour in each
adapter, and the S3 / SQS adapters have no reason to share DynamoDB's bound.

## `testkit/`

`NewTestClient(t)` builds a client from `config.NewRealtimeTestConnection` — DynamoDB Local on
`localhost:8000` by default, redirectable to AWS DynamoDB through `REALTIME_TEST_*` so the same
contract test runs against both. `TableName(t, base)` yields a per-run unique lowercase name, and
`DeleteOnCleanup` drops it when the test ends, so several checkouts can share one DynamoDB Local.

## Test strategy

- `New`, `Normalize`, `IsConditionalCheckFailed`, `isResourceInUse` are unit-tested without a
  connection: the client is inspected through `Options()`, the classifiers through SDK error types.
- `EnsureTable` and the testkit are contract-tested against DynamoDB Local (the shared
  `dynamodb_local` service locally, the `go-test` service container in CI). They are the substrate the
  three store adapters build on, so their contract is established against the real thing, not a fake.
