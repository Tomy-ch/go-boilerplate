# cliexec

English | [日本語](README.ja.md)

Shared external-process execution wrapper for CLI commands. Lets callers depend on an interface instead of `os/exec` directly so the orchestration logic is unit-testable.

## Public API

|Type / Method|Description|
|---|---|
|`Runner`|Interface abstracting command execution|
|`OS`|`Runner` implementation backed by `os/exec`|
|`Runner.Output(ctx, dir, name, args) ([]byte, error)`|Run a command in `dir`, return stdout; stderr goes to `os.Stderr`|

## Notes

- Lives under `internal/cli` because `os` / `os/exec` usage is only permitted there (and in `cmd` / `config` / `scripts`) by depguard.
- Inject `Runner` to test; production wires `OS{}`. A generated mock lives under `mock/`.
