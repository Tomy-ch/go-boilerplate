# uuid

English | [日本語](README.ja.md)

UUID value object wrapping `github.com/google/uuid`. Generates UUIDv7 and supports database integration.

## Wraps

`github.com/google/uuid`

## Notes

- Wire representation is a **JSON string** (`"b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f"`): the value object holds only an unexported array, so the default struct encoding would emit `{}` and lose the value. `UnmarshalJSON` rejects any non-string JSON value and leaves the receiver unchanged on `null`.
- `NewTestFromSalt` is for testing only — do not use in production
- sqlc override aligns DB UUID with this type, eliminating manual conversion
