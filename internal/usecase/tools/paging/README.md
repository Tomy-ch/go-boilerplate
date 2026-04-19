# paging

English | [日本語](README.ja.md)

Provides common pagination structures and logic for converting 1-based page/perPage parameters into limit/offset values.

## Public API

|Function / Method|Description|
|---|---|
|`NewPagingFrom1Based(page, perPage *int)`|Create `Paging` from 1-based page number and per-page count|
|`Limit()` / `Limit32()`|Return the retrieval limit (int / int32)|
|`Offset()` / `Offset32()`|Return the offset (int / int32)|

## Constants

|Constant|Value|Description|
|---|---|---|
|`defaultPerPage`|50|Default items per page|
|`maxPerPage`|200|Maximum items per page|
|`minPage`|1|Minimum page number|
|`maxPage`|10,000|Maximum page number|

## Behavior

- `perPage` ≤ 0 or nil → uses `defaultPerPage` (50)
- `perPage` > `maxPerPage` → clamped to `maxPerPage` (200)
- `page` ≤ 0 or nil → uses `minPage` (1)
- `page` > `maxPage` → returns `apperror.ErrInvalidArgument`
- `Limit32()` / `Offset32()` clamp values for safe int32 conversion

## Usage

```go
pg, err := paging.NewPagingFrom1Based(ptr.To(2), ptr.To(20))
// pg.Limit() == 20, pg.Offset() == 20
```
