# conv

English | [日本語](README.ja.md)

Boundary helpers that convert OpenAPI-generated types into domain types, used **only** by the controller layer.

## Why

OpenAPI-generated types (`github.com/oapi-codegen/runtime/types`) must not leak below the controller layer. Centralizing the conversions here keeps that import confined to the boundary — `usecase` / `domain` never depend on generated types.

## Notes

- `UUID` does **not** return an error. The value is already format-validated by echo's binding, so conversion always succeeds. If it somehow cannot be parsed, that is an unreachable invariant violation (a bug), so it **panics** rather than returning an error — keeping handlers free of dead error branches.
- Do not add a bypass constructor to `pkg/uuid` just to skip this conversion; the panic-on-invalid assertion is intentional and is exercised in tests via the string-input helper.
