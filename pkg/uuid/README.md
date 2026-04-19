# uuid

English | [日本語](README.ja.md)

UUID value object wrapping `github.com/google/uuid`. Generates UUIDv7 and supports database integration.

## Public API

|Function / Method|Description|
|---|---|
|`New()`|Generate UUIDv7|
|`Parse(s)`|Parse UUID from string|
|`NewTestFromSalt(t, salt)`|Generate deterministic UUID for testing|
|`String()`|Return string representation|
|`Bytes()`|Return underlying `[16]byte`|
|`ToPrimitive()`|Convert to `google/uuid.UUID`|
|`IsNil()`|Check if zero value|
|`Equal(v)`|Compare UUIDs|
|`ToPtr()`|Get pointer to UUID|
|`EqualPtr(v)`|Compare via pointer|
|`Scan(src)` / `Value()`|`sql.Scanner` / `driver.Valuer` for DB integration|

## Wraps

`github.com/google/uuid`

## Notes

- `NewTestFromSalt` is for testing only — do not use in production
- sqlc override aligns DB UUID with this type, eliminating manual conversion
