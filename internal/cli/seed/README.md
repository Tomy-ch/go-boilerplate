# db-seed

English | [日本語](README.ja.md)

Inserts seed data into the database for development and testing purposes.

## Command

```text
db-seed
```

## Flags

|Flag|Default|Description|
|---|---|---|
|*(none)*|||

## Usage

```bash
./server db-seed
```

## Notes

- **Not intended for production use.** Use only in development and test environments.
- Seed files are loaded from the `database/seed` directory.
- If a target table does not exist, the corresponding seed file is skipped and a warning is logged.
- Always verify seed data contents before running against shared environments.

### Placeholders

Before executing a file, the command expands every `${NAME}` it contains — see
[`database/seed/README.md`](../../../database/seed/README.md). The runner substitutes only the names it
is handed, so which environment-dependent value exists is decided here, at the command (`AUTH_ISSUER` =
the JWT issuer read from the configuration), not by the runner. A placeholder with no value — undefined
or empty — aborts the run rather than being inserted as an empty string.

### Seed objects

After the SQL files, the command uploads everything under `storage/seed`, deriving each object key from
the path relative to that directory — see [`storage/README.md`](../../../storage/README.md). It only
uploads; a column that holds an object key is written by the SQL, like any other column.

Nothing is uploaded when the directory holds no file, or when `OBJECT_STORAGE_ENDPOINT` is empty. An
empty endpoint means SDK-default resolution, i.e. a real AWS S3 account, and seed content must never be
pushed there — which is also why CI, whose endpoint is empty, only ever seeds the database.
