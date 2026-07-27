# storage

Local seed content for the object-storage substrate — the counterpart of `database/seed` for the
S3-compatible bucket. `database/` bootstraps rows; this directory bootstraps objects.

## The directory layout is the key layout

`make db-seed` uploads everything under `seed/` and **derives each object key from the path relative to
`seed/`**, so the tree built here is the tree that appears in the bucket:

```text
storage/seed/<prefix>/<file>   →   <prefix>/<file>
```

A subdirectory is simply a key prefix; adding one puts its files under that prefix. Nothing in the
seeder is bound to a particular prefix — it never interprets file names and never touches the database.

- Files sit one level below `seed/`; a file placed directly in `seed/` has no prefix and is skipped
- Accepted extensions are the ones the seeder can label with a MIME type (`.webp` / `.png` / `.jpg` /
  `.jpeg` / `.svg` / `.pdf`). Anything else is skipped with a warning rather than served as the wrong
  type; dotfiles are ignored
- Keep the files small. Everyone who clones this template pulls them

## Rows that reference an object

The seeder only uploads. A column holding an object key is written by the SQL in `database/seed`, like
any other column — so the key in the SQL and the file name here have to agree. Keeping both in the same
commit is what stops them from drifting.

## What is not uploaded

Nothing happens — success, not an error — when `seed/` holds no file, or when `OBJECT_STORAGE_ENDPOINT`
is empty. An empty endpoint means SDK-default resolution, i.e. a real AWS S3 account, and seed content
must never be pushed there. That is also why CI, whose endpoint is empty, only ever seeds the database.

No `Cache-Control` is set on the uploaded objects. A seed file is replaceable in place under the same
key, so it is not immutable; the caching policy for content uploaded through the API is decided by that
upload path instead.
