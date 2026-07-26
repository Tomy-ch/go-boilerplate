# Idempotency (Idempotency-Key)

English | [日本語](README.ja.md)

Make non-idempotent writes (POST/PATCH/PUT) safe against client retries: a side effect runs **at most once**, and a retry gets the **same response**.

## 1. Concept — why a transaction is not enough

A database transaction only guarantees the **atomicity of a single request**. It does **not** dedupe a request that is **sent twice** (network timeout, double submit, client auto-retry). When a write has no natural unique key — a `uuid.New()`-allocated POST, a balance increment, a charge, an email send — a retry would execute the side effect again.

This package closes that gap with a client-supplied `Idempotency-Key`.

Not to be confused with:

- **Optimistic locking** — prevents lost updates when two *different* operations race on the same row (a `version` column). Orthogonal to idempotency; use both if needed.
- **Rate limiting** — an edge (gateway/LB) concern, intentionally out of scope here.

Use an idempotency key **only** for non-idempotent, retry-prone writes. Adding it to a `GET` is meaningless; omitting it from a charge endpoint is a bug.

## 2. State transitions

```text
(none)
  │  INSERT ON CONFLICT DO NOTHING  (inside the business tx)
  ▼
claimed ──── business fn ok + result saved (same tx) ───▶ completed
  │                                                          │
  │ business error / crash  →  tx rollback                   │ retry
  ▼  (key auto-released → re-executes)                       ▼
(none)                                              replay saved result

retry branches:
  - retry to completed  → replay (business fn NOT run)
  - retry to claimed    → 409 (being processed, retry later)
  - fingerprint mismatch → 422 (same key, different request)
  - TTL expired (GC'd)  → fresh execution
```

The claim and the business write share **one** transaction (strict consistency). A business failure rolls back the claim too — **failure auto-releases the key**, so error caching needs no special handling.

## 3. How to make an endpoint idempotent

Two steps (opt-in; an endpoint without the steps is unchanged):

1. **Entry middleware** — add the StrictMiddleware to the handler's `NewStrictHandler` second argument:

   ```go
   gen.RegisterHandlers(e, gen.NewStrictHandler(server,
       []gen.StrictMiddlewareFunc{idempotencymw.StrictMiddleware[gen.StrictHandlerFunc]()}))
   ```

2. **Wrap the usecase call** in `Run[T]` with the success status:

   ```go
   dto, _, err := idempotency.Run(ctx, s.idem, http.StatusCreated, func(ctx context.Context) (user.UserView, error) {
       return s.uc.CreateUser(ctx, params)
   })
   ```

`PostUsers` (`internal/controller/handler/v1/users/v1_users_handler.go`) is the reference adoption. The middleware only triggers when the `Idempotency-Key` header is present; without it `Run` just runs `businessFn` (non-breaking).

## 4. Client contract

- **`Idempotency-Key` header**: non-empty, ≤ 255 chars, printable ASCII (`0x21`–`0x7E`). UUID is recommended but not required. Malformed → **400**.
- **Replay**: a retry with the same key + same request returns the original response.
- **409 Conflict**: the key is currently being processed (concurrent retry) — retry later.
- **422 Unprocessable Entity**: the same key was reused with a *different* request body.
- Scoped to the authenticated principal: a key is unique **per user** (`UNIQUE (scope, idempotency_key)`), so one user cannot collide with or read another user's key.

## 5. (c) Per-endpoint scope extension (no config flag)

Default scope = principal. To isolate keys per endpoint as well (`scope = principal + operationId`), change the scope composition in **one place** — the entry middleware already has `operationId` for free (also used as the o11y label). Do not add a runtime config flag; keep it a documented code change. Onion note: the operationId comes from HTTP, so prefer threading the usecase-method identity if you want a purer source — both are acceptable.

## 6. Operations

- **GC job** `idempotency-gc` (`internal/controller/job/idempotencygc/`) batch-deletes expired entries. Run it from an external scheduler: `cmd job idempotency-gc` (`--batch-size=N`, default 10,000). Recommended interval: **hourly** (TTL is 24h, so realtime is unnecessary).
- **TTL = 24h** = the retry window. A retry after the TTL becomes a fresh execution.
- **Metrics**: the idempotency outcome / failure / GC-cleanup counters are observed at the usecase boundary (not by guessing from HTTP status), since hit/miss/conflict can only be decided from the `Claim`/`Get`/`Complete` results.
  - `Run[T]` reports `Deps.Metrics` (`idempotency.Metrics`): `IncMiss` (new claim), `IncHit` (completed replay), `IncConflict` (lock timeout / still-claimed / entry vanished after claim), `IncFingerprintMismatch`, `IncClaimFailure` (non-`ErrLockTimeout` claim error), `IncCompleteFailure`.
  - `GCUsecase` reports `GCMetrics`: `IncExpiredCleanup(count)` per successful batch and `IncExpiredCleanupFailure()` on a delete error.
  - The wired implementation is `observability.NewIdempotencyMetrics` (provided in `internal/di/module/usecase.go`, annotated as both interfaces). It emits OpenTelemetry counters `idempotency.requests{operation_id,result}`, `idempotency.failures{operation_id,phase}`, and `idempotency.expired_cleanup{job}`. High-cardinality / sensitive values (Idempotency-Key, scope, fingerprint, PII, raw error) are **never** labels; an empty `operation_id` is normalized to `unknown`.
  - Both `Deps.Metrics` and the `GCMetrics` argument remain optional: a `nil` value is **no-op** (so `Run`/`GC` work without an observability backend).
- **Single success status per operation**: `Run[T]` records one `successStatus` and `PostUsers` always returns 201. If you adopt `Run[T]` on an endpoint that can return multiple success statuses (e.g. 200 vs 201), extend the handler to dispatch on the stored status — replay currently re-renders via the handler's fixed response type.

## Security / storage notes

- **Scope isolation (IDOR)**: there is no DB FK/RLS on `scope`; isolation is enforced in code — every query carries `WHERE scope = <principal>` and there is no `id`-only lookup. The `Store` interface takes `scope` as a mandatory argument.
- **`response_payload` PII**: the result DTO is stored as JSON (BYTEA). For PII-bearing DTOs (e.g. `UserResponse`), be aware of DB dump/backup exposure (rows clear after the 24h TTL). If this matters, store a cache-only DTO, encrypt with pgcrypto, or avoid storing PII-bearing DTOs raw.
- **`request_fingerprint`** is always a 32-byte SHA-256 by construction (the middleware computes `sha256(method+path+typed-request)`); a DB `CHECK (octet_length = 32)` can be added as defense-in-depth.

## Layout

| Layer | Path |
| --- | --- |
| migration | `database/migrations/000001_create_idempotency_keys.*.sql` |
| sqlc DML | `database/dml/system_cqrs/idempotency/` |
| boundary | `internal/usecase/boundary/idempotency/` (`Store`) |
| infrastructure | `internal/infrastructure/rdb/system_cqrs/idempotency/` |
| usecase | `internal/usecase/idempotency/` (`Run[T]`, `GCUsecase`) |
| controller (entry) | `internal/controller/httpstack/idempotency/` |
| GC job | `internal/controller/job/idempotencygc/` |
