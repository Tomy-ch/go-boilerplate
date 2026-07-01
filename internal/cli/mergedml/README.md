# merge-dml

English | [日本語](README.ja.md)

Merges multiple DML SQL files under `database/dml/<type>/<category>/` into single consolidated files for sqlc code generation. Output files are written to `database/gen/<category>_<type>.gen.sql`.

## Command

```text
merge-dml --type <type> [flags]
```

## Flags

|Flag|Default|Description|
|---|---|---|
|`--type`|*(required)*|Target DML type (e.g. `repository`, `query_service`)|
|`--work-dir`|`/app`|Working directory (project root)|

## Usage

```bash
# Merge all repository DML files
./server merge-dml --type repository

# Merge query service DML files with a custom work directory
./server merge-dml --type query_service --work-dir /app
```

## Notes

- Categories are merged in parallel. The concurrency level is `runtime.NumCPU()` clamped to the range `[2, 4]` (lower bound `2` avoids serializing I/O-bound work; upper cap `4` prevents monopolizing the CPU inside Docker/CI).
- SQL files within each category are sorted by path before merging to produce stable output. A `-- === source: <path> ===` header comment is inserted before each source file's contents so the origin stays traceable.
- Stale generated files are automatically cleaned up when their source category no longer contains SQL files.
- Output paths are validated to stay within `database/gen/` to prevent accidental writes elsewhere.
