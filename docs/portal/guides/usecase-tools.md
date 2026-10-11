# tools

`internal/usecase/tools` contains **small reusable utilities for the Usecase layer**.

## Subdirectories

|Package|Description|Details|
|---|---|---|
|`paging/`|Pagination (page/perPage → limit/offset conversion) and the shared fetch-count policy, top-N reads included|[README](paging/README.md)|
|`search/`|Search keyword tokenization (split, dedup, limit)|[README](search/README.md)|
|`money/`|Money arithmetic (integer minor-unit, half-up rate application)|[README](money/README.md)|
|`timewindow/`|Half-open ordered-time interval `[After, Before)` and its emptiness rule|[README](timewindow/README.md)|
|`datetime/`|Wire representation of an outbound time (UTC, RFC 3339 nanosecond)|[README](datetime/README.md)|

## Design Policy

- Utilities shared across multiple Usecases
- No business logic — only mechanical transformations
- No Infrastructure dependencies
- Before adding a package here, derive its shape from the existing package that plays the **same
  mechanical role**. Two shapes live here: a request parameter normalized into a value object
  (`paging` / `timewindow` — unexported fields, a validating constructor, a call site in the handler),
  and a pure transformation that owns no type (`search` / `money` / `datetime`).
  See *New Type Derivation* in [docs/rules.md](../../../docs/rules.md).
