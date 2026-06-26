# exec

English | [日本語](README.ja.md)

Provides a thin wrapper around external process execution so callers depend on an interface instead of `os/exec` directly.

## Wraps

- `os/exec.CommandContext`

## Notes

- Uses `os` / `os/exec`, so depguard's `reject_dangerous_os` is relaxed for this package (`!**/pkg/exec/**.go`).
- Must NOT be imported from the domain / usecase layers (enforced by depguard); process execution belongs to outer layers.
- Inject `Runner` to test; production wires `OS{}`. A generated mock lives under `mock/`.
