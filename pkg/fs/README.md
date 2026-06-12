# fs

English | [日本語](README.ja.md)

Provides a thin wrapper around filesystem operations so callers depend on an interface instead of `os` directly.

## Public API

|Type / Method|Description|
|---|---|
|`FS`|Interface abstracting filesystem operations|
|`OS`|`FS` implementation backed by `os` / `path/filepath`|
|`FS.ReadFile(name) ([]byte, error)`|Read a file|
|`FS.WriteFile(name, data, perm) error`|Write a file|
|`FS.Glob(pattern) ([]string, error)`|Glob match|

## Wraps

- `os.ReadFile` / `os.WriteFile`
- `path/filepath.Glob`

## Notes

- Uses `os`, so depguard's `reject_dangerous_os` is relaxed for this package (`!**/pkg/fs/**.go`).
- Must NOT be imported from the domain / usecase layers (enforced by depguard); filesystem I/O belongs to outer layers.
- Inject `FS` to test; production wires `OS{}`. A generated mock lives under `mock/`.
