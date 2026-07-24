# boundary

English | [日本語](README.ja.md)

`internal/usecase/boundary` defines the **interfaces that the Usecase layer requires from external layers (Infrastructure)**.

## Position in Onion Architecture

```mermaid
flowchart TB
    subgraph "Usecase Layer"
        Boundary["boundary (interface definitions)"]
        UC["Usecase impl"]
    end
    subgraph "Infrastructure Layer"
        Impl["Concrete impl"]
    end

    UC --> Boundary
    Impl -. implements .-> Boundary
```

boundary is the mechanism for achieving the **Dependency Inversion Principle (DIP)**.

- Usecase depends only on boundary interfaces
- Infrastructure provides concrete implementations
- Usecase has no knowledge of Infrastructure implementation details

### Difference from Domain Repository Interface

|Aspect|Domain Repository|Usecase Boundary|
|---|---|---|
|Definition location|Domain layer|Usecase layer|
|Purpose|Aggregate persistence abstraction|Abstraction of external capabilities needed by Usecase|
|Scope|Persistence (CRUD)|Auth / encryption / time / transaction / job, etc.|

Domain Repository abstracts "how to persist Aggregates", while Usecase Boundary abstracts "external capabilities Usecase needs to execute business flows".

## Package List

|Package|Interface|Description|Implementation|
|---|---|---|---|
|`auth`|`Authenticator`|Obtain auth info (`Authn`) from token|`internal/infrastructure/auth/`|
|`authz`|`Authorizer`|Decide whether a subject may perform an action on a resource|`internal/infrastructure/authz/`|
|`clock`|`Clock`|Retrieve current time|`internal/infrastructure/system/`|
|`exchangerate`|`Gateway`|Semantic gateway to an external exchange-rate service (sample of the `<service>.Gateway` pattern)|`internal/infrastructure/webapi/exchangerate/`|
|`idempotency`|`Store`|Idempotency-key persistence boundary (claim / replay / conflict)|`internal/infrastructure/rdb/system_cqrs/idempotency/`|
|`job`|`Job`, `Runner`, `State`|Job definition, execution, state management|`internal/controller/job/`|
|`objectstorage`|`Storage`|Substrate-agnostic object-storage boundary (`Put` an object by key, return its stored `Path`)|`internal/infrastructure/objectstorage/s3/`|
|`outbox`|`Store`|Transactional outbox table persistence boundary|`internal/infrastructure/rdb/system_cqrs/outbox/`|
|`publisher`|`Publisher`|Substrate-agnostic outbound message publish boundary|`internal/infrastructure/publisher/`|
|`tx`|`Manager`|Transaction boundary management|`internal/infrastructure/rdb/driver/`|
|`worker`|`Consumer`, `Handler`, `FailureHandler`, `Worker`, `State`|Broker-agnostic worker seam (pull-ack)|`internal/infrastructure/queue/sqs/`|

## Package Details

### auth

Provides interfaces and value objects for authentication.

|Type / Function|Description|
|---|---|
|`Authenticator`|Interface to generate `Authn` from `Credential`|
|`Authn`|Authentication result (subject / userID / issuer / scopes / claims)|
|`New(subject, issuer, scopes, claims)`|Create `Authn` with the UserID unresolved (empty subject returns `ErrUnauthenticatedSubjectMissing`)|
|`WithUserID(userID)`|Return a copy of `Authn` with the internal UserID resolved|
|`Credential`|Value object holding the auth scheme + token|
|`NewCredential(scheme, token)`|Create `Credential` (empty token returns `ErrTokenMissing`)|

Errors:

|Error|Description|
|---|---|
|`ErrUnauthenticatedSubjectMissing`|Subject is empty|
|`ErrUserIDUnresolved`|Internal UserID is unresolved|
|`ErrTokenMissing`|Token is empty|

### authz

Provides the interface and value objects for authorization — the counterpart to `auth`. The Usecase layer is the enforcement point (PEP): it calls `Authorize(...)` and maps a deny to `apperror.ErrPermissionDenied` (403).

|Type / Function|Description|
|---|---|
|`Authorizer`|Interface deciding whether `authn` may perform `action` on `resource` (`Authorize(ctx, *auth.Authn, Action, *Resource) error`)|
|`Action`|Operation being authorized (e.g. `ActionUserDelete` = `"user:delete"`)|
|`Resource`|Target resource carrying `Kind()` and optional `OwnerID()`, so ownership-based (object-level) decisions are expressible|
|`NewResource(kind, ownerID)`|Create a `Resource`|

Errors:

|Error|Description|
|---|---|
|`ErrForbidden`|Authorization denied (wraps `apperror.ErrPermissionDenied`, HTTP 403)|

Passing the full `auth.Authn` (subject / scopes / claims) plus the target `Resource` lets both RBAC (roles from claims) and ownership (subject == OwnerID) models be expressed. The default implementation is allow-all and restricted to non-production environments.

### clock

```go
type Clock interface {
    Now() time.Time
}
```

Abstraction to prevent Domain / Usecase from depending directly on `time.Now()`. Allows mock substitution in tests.

### exchangerate

Sample Gateway boundary: a semantic port to an external exchange-rate service (`<service>.Gateway` pattern). Keeps Usecase depending on a semantic port rather than `net/http` or a vendor SDK, and translates transport failures into `apperror` sentinels at the boundary.

|Type / Function|Description|
|---|---|
|`Gateway`|Fetch a conversion rate via `GetRate(ctx, base, quote)`|
|`Rate`|Output DTO (`Base` / `Quote` / `Value`)|

### idempotency

Persistence boundary for idempotency keys. Every method requires a `scope` (no id-only lookup, to prevent cross-boundary access).

|Type / Function|Description|
|---|---|
|`Store`|Idempotency-key persistence boundary interface|
|`Claim(ctx, p)`|Create a claimed row; returns `claimed=true` when new, `false` when the key already exists (`ErrLockTimeout` on lock-wait timeout)|
|`Get(ctx, scope, key)`|Return the stored state for `(scope, key)` (nil when absent)|
|`Complete(ctx, p)`|Transition `claimed` → `completed` and store the response|
|`DeleteExpired(ctx, cutoff, limit)`|Delete rows older than `cutoff` up to `limit` (GC), returning the count|

Input / output value objects: `ClaimParams` / `CompleteParams` (inputs) and `Record` (stored state). Sentinel: `ErrLockTimeout` (mapped to 409 by the usecase).

### job

|Interface|Description|
|---|---|
|`Job`|Job definition with `Name()` + `Execute(ctx, args)`|
|`Runner`|Execute and list jobs via `Run(ctx, jobName, args)` + `Names()`|
|`State`|Manage job execution state via `Set(name, args, done)` + `Snapshot()`|

### objectstorage

Substrate-agnostic object-storage boundary. Usecase depends only on this port; the S3-compatible adapter (infrastructure) implements it, and vendor vocabulary (bucket / region / etag) never leaks across the boundary.

|Type / Function|Description|
|---|---|
|`Storage`|Save an object under its key via `Put(ctx, PutObject) (Path, error)`; failures return an `apperror` sentinel (e.g. `ErrUnavailable`)|
|`PutObject`|Input DTO (`Key` / `Body` / `ContentType`); the caller assigns `Key` (e.g. `products/{uuid}.png`)|
|`Path`|The stored object path (key); the display URL is composed separately by the caller|

### outbox

Persistence boundary for the transactional outbox table. The emit usecase and the relay engine (controller layer) both depend on it.

|Type / Function|Description|
|---|---|
|`Store`|Outbox table persistence boundary interface|
|`Insert(ctx, p)`|INSERT one outbox row within the business tx, returning the assigned `message_id`|
|`ClaimPending(ctx, limit)`|Claim up to `limit` pending rows (`FOR UPDATE SKIP LOCKED`)|
|`MarkPublished(ctx, id)`|Transition a published row to `published` (no-op unless still pending)|
|`MarkFailed(ctx, id, lastErr)`|Increment `attempts`, record `last_error`, return the new attempt count|
|`MarkDead(ctx, id)`|Transition a row to `dead` (no-op unless still pending)|
|`ReplayDead(ctx, messageID)`|Return `dead` rows to `pending` (all dead rows when `messageID` is nil), returning the count|
|`DeletePublished(ctx, cutoff, limit)`|Delete published rows older than `cutoff` up to `limit` (GC), returning the count|
|`OldestPendingCreatedAt(ctx)`|Return the oldest pending row's `created_at` for the outbox-lag SLI (`ok=false` when none)|

Input / output value objects: `EmitParams` (INSERT input) and `PendingMessage` (claimed unpublished row).

### publisher

Outbound publish boundary for domain events plus a substrate-agnostic message envelope. Both the relay engine (controller layer) and the publish adapter (infrastructure layer) depend on it.

|Type / Function|Description|
|---|---|
|`Publisher`|Boundary that sends a message to its destination|
|`Publish(ctx, m)`|Send `m` to the destination; on failure returns an error and the relay re-sends on its next poll (at-least-once)|
|`Message`|Substrate-agnostic message envelope built from an outbox row (exposes no `net/http` types)|

### tx

|Type / Function|Description|
|---|---|
|`Manager`|Manage transaction boundaries via `Do(ctx, fn)`|
|`DoWithResult[T](ctx, m, fn)`|Generic helper to return a value from within a transaction|

### worker

|Type / Function|Description|
|---|---|
|`Consumer`|Broker-agnostic message intake — `Receive` / `Ack` / `Nack` / `NackWithBackoff` / `Extend` (implemented by a broker adapter)|
|`Handler`|Per-message business processing (must be idempotent)|
|`FailureHandler`|Dead-letter sink for permanent failures|
|`Worker`|Bundles Name / Consumer / Handler / FailureHandler|
|`State`|Selected-worker state shared with the engine|
|`QueueStatsProvider`|Optional queue depth / DLQ stats source for metrics|

## Design Policy

- boundary contains no business logic (only interfaces and value objects)
- Importing Infrastructure is prohibited (dependency direction violation)
- All interfaces have `mockgen` auto-generation configured
