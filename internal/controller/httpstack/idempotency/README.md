# idempotency

`Idempotency-Key`-based request deduplication entry point, provided as an oapi-codegen StrictMiddleware slot (not `e.Use`).

## Role

Safe retries require the server to recognize when a client resends the same mutating request, so that a duplicate does not apply the operation twice. Placing that recognition at the middleware boundary — where the request is already parsed into its typed form — lets every endpoint opt into idempotency uniformly without each handler reimplementing key handling and request fingerprinting. This package only establishes the idempotency context for a request; the actual persistence and replay of stored responses live in the usecase layer.

## Notes

- `Middleware()` returns the entry point in the StrictMiddleware structural signature `func(next NextFunc, operationID string) NextFunc`, where `NextFunc` is `func(ctx *echo.Context, request any) (any, error)`. `StrictMiddleware[H]()` adapts it to a package-specific oapi-codegen `StrictMiddlewareFunc` type (e.g. `gen.StrictHandlerFunc`), so it is registered in the generated strict-handler middleware slot — not via `e.Use`.
- When the `Idempotency-Key` header is absent, the request passes through unchanged (treated as non-idempotent).
- Idempotency activates only when an authentication principal is present: the principal's subject is used as the scope key — not the internal `UserID` (ADR-0063) — so an unauthenticated request passes through unchanged.
- An absent or blank `Idempotency-Key` passes through unchanged. A present key is validated and a violating one is rejected with `400` (`apperror.ErrInvalidArgument`): it must be at most 255 bytes and contain only printable ASCII characters.
- The request fingerprint is the SHA-256 of `method + path + JSON(typed request)`. Marshalling the request is fail-closed: if it fails, an internal error (`apperror.ErrInternal`) is returned rather than proceeding with a weak fingerprint.
- On success, the middleware stores an `idempotency.Request` (scope, key, fingerprint, method, path, operationID) into the request context via the usecase's `WithRequest`, then delegates to the next handler; the usecase layer consumes it downstream.
- On the way back, an error the handler returns that is `idempotency.ErrKeyReused` or `idempotency.ErrKeyInProgress` gets an `apperror.Meta` with the code `IDEMPOTENCY_KEY_REUSED` / `IDEMPOTENCY_KEY_IN_PROGRESS` and its message; the status stays 422 / 409. Any other error passes through unchanged.
