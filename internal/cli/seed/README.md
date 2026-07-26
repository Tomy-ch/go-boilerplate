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

### Product images

After the SQL files, the command uploads the sample product images in `storage/seed/products` and writes
each stored key into the matching row's `image_path`. The file name is the key, so the two cannot drift —
see [`storage/README.md`](../../../storage/README.md) for the naming contract.

Nothing is uploaded when the directory holds no image, or when `OBJECT_STORAGE_ENDPOINT` is empty. An
empty endpoint means SDK-default resolution, i.e. a real AWS S3 account, and sample images must never be
pushed there — which is also why CI, whose endpoint is empty, only ever seeds the database.
