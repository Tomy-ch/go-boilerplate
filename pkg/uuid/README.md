# uuid

English | [日本語](README.ja.md)

UUID value object wrapping `github.com/google/uuid`. Generates UUIDv7 and supports database integration.

## Wraps

`github.com/google/uuid`

## Notes

- `NewTestFromSalt` is for testing only — do not use in production
- sqlc override aligns DB UUID with this type, eliminating manual conversion
