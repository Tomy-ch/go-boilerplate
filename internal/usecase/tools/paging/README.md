# paging

English | [日本語](README.ja.md)

Provides common pagination value objects. Two strategies are offered as **equally valid application policies** (pagination is a Usecase-tier concern, not a domain rule):

- **Offset-based (`Page`)** — converts 1-based page/perPage into limit/offset. Simple, allows random page access, but degrades on deep pages (large `OFFSET` scans).
- **Cursor-based / keyset (`Cursor`)** — carries an opaque cursor (the previous page's last-row sort keys) for `WHERE (sort_keys) < (:cursor)` queries. Stable and fast even on deep pages; recommended for large datasets and infinite scroll. The package owns only **transport (encode/decode), validation, and limit policy**; interpreting the keys back into typed sort columns (e.g. RFC3339 → time, UUID string → uuid) is the **query layer's** responsibility.

The package also owns the **fetch-count policy on its own** (`Limit` / `LimitPolicy`), so a read
with no pagination at all — a top-N list such as a ranking or a low-stock dashboard card — shares
the same "unspecified → default, over the ceiling → clamp" rule instead of re-implementing it.
`Page` and `Cursor` are built on `Limit` with the package's own `defaultPerPage` / `maxPerPage`;
top-N callers pass their own `LimitPolicy` (its named fields keep the default and the ceiling from
being swapped at a call site).

## Constants

|Constant|Value|Description|
|---|---|---|
|`defaultPerPage`|50|Default items per page (`Page` / `Cursor`)|
|`maxPerPage`|200|Maximum items per page (`Page` / `Cursor`)|
|`minPage`|1|Minimum page number|
|`maxPage`|10,000|Maximum page number|

## Behavior

**Fetch count (`Limit`)**

- `first` ≤ 0 or nil → uses `policy.Default`
- `first` > `policy.Max` → clamped to `policy.Max`
- `Value32()` clamps at `math.MaxInt32` for safe int32 conversion

**Offset-based (`Page`)**

- `perPage` ≤ 0 or nil → uses `defaultPerPage` (50)
- `perPage` > `maxPerPage` → clamped to `maxPerPage` (200)
- `page` ≤ 0 or nil → uses `minPage` (1)
- `page` > `maxPage` → returns `apperror.ErrInvalidArgument`
- `Limit32()` / `Offset32()` clamp values for safe int32 conversion

**Cursor-based (`Cursor`)**

- `first` ≤ 0 or nil → uses `defaultPerPage` (50)
- `first` > `maxPerPage` → clamped to `maxPerPage` (200)
- `after` nil or empty → first page (`HasCursor()` is `false`, `Keys()` empty)
- `after` malformed (bad base64 / JSON, or empty keyset) → returns `apperror.ErrInvalidArgument`
- keyset has **no page-number ceiling** — that is the whole point of cursor pagination, so there is no `maxPage`-style error
- Cursor string format is **opaque**: `base64url(JSON string array)`. Treat it as a black box on the client side.

## Usage

### Top-N (no pagination)

```go
var lowStockLimitPolicy = paging.LimitPolicy{Default: 20, Max: 100}

limit := paging.NewLimit(req.Limit, lowStockLimitPolicy)
// limit.Value() == 20 when req.Limit is nil; limit.Value32() feeds the SQL LIMIT.
```

### Offset-based

```go
pg, err := paging.NewPageFrom1Based(ptr.To(2), ptr.To(20))
// pg.Limit() == 20, pg.Offset() == 20
```

### Cursor-based

```go
// 1) Parse the incoming request (first page: after == nil).
cur, err := paging.NewCursor(req.After, req.First)

// 2) Query layer: fetch limit+1 rows. If cur.HasCursor(), apply a keyset
//    predicate using cur.Keys() (interpret strings into typed columns):
//      WHERE (created_at, id) < ($1, $2) ORDER BY created_at DESC, id DESC
//      LIMIT cur.Limit32() + 1
//    Fetching one extra row tells you whether a next page exists.

// 3) Build the next cursor from the LAST visible row's sort keys:
next := paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano), last.ID.String())
// Return `next` only when the extra row was present; otherwise "" (end of list).
```

> Sort keys must be **unique and totally ordered** as a tuple (append a tie-breaker like the primary key), otherwise rows can be skipped or duplicated across pages.

## Test coverage exception

The following uncovered branch is exempt from the near-100% expectation as an **infallible
defensive branch**; no contrived test or extra implementation is added to reach it:

- `cursor.go` `EncodeCursor` — the `json.Marshal(keys)` error return. `keys` is a
  `[]string`, which the encoder can always marshal, so the error path is unreachable.

**Governance:** coverage exceptions are **not added at will** — a new entry requires an
appropriate approver's (e.g. architect) sign-off.
