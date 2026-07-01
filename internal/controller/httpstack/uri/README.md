# uri

English | [日本語](README.ja.md)

Removes trailing slashes from request URIs.

## Role

Treating `/path` and `/path/` as the same route avoids duplicate-path bugs and inconsistent client behaviour. Normalizing the trailing slash in one middleware before routing means every handler sees a single canonical path form, so route matching and handler logic never have to account for the variant.
