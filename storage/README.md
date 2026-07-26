# storage

Local seed content for the object-storage substrate — the counterpart of `database/seed` for the
S3-compatible bucket. `database/` bootstraps rows; this directory bootstraps objects.

## Directory

|Path|Role|
|---|---|
|`seed/products/`|Sample product images, uploaded by `make db-seed`|

## `seed/products/`

**The file name is the object key.** A file named `{product ID}.{ext}` is stored under
`products/{product ID}.{ext}`, and the seeder writes that same string into the matching row's
`image_path`. Key and column therefore cannot drift: to add an image, drop in a file whose base name
is the product's UUID as it appears in `database/seed/*.sql`, and re-run `make db-seed`.

- Accepted extensions: `.webp` (preferred) / `.png` / `.jpg` / `.jpeg`. Anything else is skipped with a warning; dotfiles are ignored
- One image per product — the schema holds a single `image_path`
- A file whose UUID matches no row is skipped with a warning, so an image can be staged before its row exists
- Keep the files small. Everyone who clones this template pulls them
- The seeder only ever writes `image_path`. Deleting a file does not clear the column it already set, and
  clearing it automatically would wipe images uploaded through the API in the same local database — reset
  such a row by hand, or rebuild the database (`make db-local-reinit`)

Production keys are `products/{uuid}.{ext}` too, but assigned per upload (see
`internal/usecase/product`), so a replaced image never overwrites an existing key. Seed keys reuse the
product's own ID instead, which is what makes them reproducible from the file name alone.

## When nothing is uploaded

The seeder is a no-op — success, not an error — when the directory holds no image, or when
`OBJECT_STORAGE_ENDPOINT` is empty. An empty endpoint means SDK-default resolution, i.e. a real AWS S3
account, and sample images must never be pushed there.
