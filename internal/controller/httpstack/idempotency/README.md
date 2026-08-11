# idempotency

English | [日本語](README.ja.md)

`Idempotency-Key`-based request deduplication entry point, provided as an oapi-codegen StrictMiddleware slot (not `e.Use`).

## Role

Safe retries require the server to recognize when a client resends the same mutating request, so that a duplicate does not apply the operation twice. Placing that recognition at the middleware boundary — where the request is already parsed into its typed form — lets every endpoint opt into idempotency uniformly without each handler reimplementing key handling and request fingerprinting. This package only establishes the idempotency context for a request; the actual persistence and replay of stored responses live in the usecase layer.

## Notes

- `Middleware()` returns the entry point in the StrictMiddleware structural signature `func(next NextFunc, operationID string) NextFunc`, where `NextFunc` is `func(ctx *echo.Context, request any) (any, error)`. `StrictMiddleware[H]()` adapts it to a package-specific oapi-codegen `StrictMiddlewareFunc` type (e.g. `gen.StrictHandlerFunc`), so it is registered in the generated strict-handler middleware slot — not via `e.Use`.
- When the `Idempotency-Key` header is absent, the request passes through unchanged (treated as non-idempotent).
- Idempotency activates only when the authenticated principal has a resolved internal `UserID`: that `UserID` is used as the scope key, so the request passes through unchanged if no authentication principal is present, or if the principal's `UserID` is unresolved.
- An absent or blank `Idempotency-Key` passes through unchanged. A present key is validated and a violating one is rejected with `400` (`apperror.ErrInvalidArgument`): it must be at most 255 bytes and contain only printable ASCII characters.
- The request fingerprint is the SHA-256 of `method + path + JSON(typed request)`. Marshalling the request is fail-closed: if it fails, an internal error (`apperror.ErrInternal`) is returned rather than proceeding with a weak fingerprint.
- On success, the middleware stores an `idempotency.Request` (scope, key, fingerprint, method, path, operationID) into the request context via the usecase's `WithRequest`, then delegates to the next handler; the usecase layer consumes it downstream.
