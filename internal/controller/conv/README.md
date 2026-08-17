# conv

English | [日本語](README.ja.md)

Boundary helpers that convert OpenAPI-generated types into domain types, used **only** by the controller layer.

## Why

OpenAPI-generated types (`github.com/oapi-codegen/runtime/types`) must not leak below the controller layer. Centralizing the conversions here keeps that import confined to the boundary — `usecase` / `domain` never depend on generated types.

## Notes

- `UUID` does **not** return an error. `openapi_types.UUID` is a value type (an already-validated 16-byte array), so converting it to the domain `pkg/uuid.UUID` via `uuid.FromPrimitive` is unconditional and cannot fail — no error branch, no panic. This keeps handlers free of dead error branches.
- `Email` returns a plain `string`; `EmailPtr` returns a `*string` and maps `nil` input to `nil`.
- `Int16sPtr` **does** return an error, unlike the helpers above. It narrows an `int32` query array to `int16`,
  and the target DB column is `SMALLINT`: a value outside that range is rejected rather than silently truncated.
  It also maps both `nil` and an empty array to `nil`, so "no filter" has a single representation downstream.
