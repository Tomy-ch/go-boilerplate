# clifs

English | [日本語](README.ja.md)

Shared filesystem-operation wrapper for CLI commands. Lets callers depend on an interface instead of `os` directly so the orchestration logic is unit-testable.

## Public API

|Type / Method|Description|
|---|---|
|`FS`|Interface abstracting filesystem operations|
|`OS`|`FS` implementation backed by `os` / `path/filepath`|
|`FS.ReadFile(name) ([]byte, error)`|Read a file|
|`FS.WriteFile(name, data, perm) error`|Write a file|
|`FS.Glob(pattern) ([]string, error)`|Glob match|

## Notes

- Lives under `internal/cli` because `os` usage is only permitted there (and in `cmd` / `config` / `scripts`) by depguard.
- Inject `FS` to test; production wires `OS{}`. A generated mock lives under `mock/`.
