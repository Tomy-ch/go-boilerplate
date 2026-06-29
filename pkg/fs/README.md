# fs

English | [日本語](README.ja.md)

Provides a thin wrapper around filesystem operations so callers depend on an interface instead of `os` directly.

## Wraps

- `os.ReadFile` / `os.WriteFile`
- `path/filepath.Glob`

## Notes

- Uses `os`, so depguard's `reject_dangerous_os` is relaxed for this package (`!**/pkg/fs/**.go`).
- Must NOT be imported from the domain / usecase layers (enforced by depguard); filesystem I/O belongs to outer layers.
- Inject `FS` to test; production wires `OS{}`. A generated mock lives under `mock/`.
