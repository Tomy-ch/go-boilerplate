# tools

English | [日本語](README.ja.md)

`internal/usecase/tools` contains **small reusable utilities for the Usecase layer**.

## Subdirectories

|Package|Description|Details|
|---|---|---|
|`paging/`|Pagination (page/perPage → limit/offset conversion)|[README](paging/README.md)|
|`search/`|Search keyword tokenization (split, dedup, limit)|[README](search/README.md)|

## Design Policy

- Utilities shared across multiple Usecases
- No business logic — only mechanical transformations
- No Infrastructure dependencies
