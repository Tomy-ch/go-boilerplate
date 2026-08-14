# period

English | [日本語](README.ja.md)

Resolves the **target period** of a purchase read — the range of order dates a list or an aggregate
is restricted to. The package owns two types and the translation between them:

- **`Spec`** — what the caller asked for, straight from the query string: a `Kind` (`all` / `month` /
  `range` / `recent`) plus the fields that kind needs. Nothing is validated yet.
- **`Window`** — the resolved answer: two calendar days, both ends inclusive, or "no filtering at
  all". Its zero value means no filtering, so a caller that never sets a period gets the whole
  history by default.

`Resolve` is the only way to get a filtering `Window`, and it is where a relative request such as
"the last 10 days" becomes two concrete dates.

## Why this sits in the Usecase layer

Two other placements are plausible and both are wrong for this repository.

**Not Domain.** [`internal/domain/README.md`](../../../domain/README.md) lists *Search
specifications* and *Aggregation processing* among what the domain is not responsible for. A target
period is neither an invariant of the purchase aggregate nor a state transition — it is a narrowing
of a question asked about it.

**Not Infrastructure.** Infrastructure executes a read that has already been decided; it does not
decide what the read means. Resolving `recent` into dates *is* a business-visible decision — which
days count — and the API returns those dates to the client. Making that decision inside a query
service would hide it from the layer that owns the response, and would make it unverifiable without
a database. Keeping it here also matches the Time Handling Policy in
[`internal/usecase/README.md`](../../README.md): time is acquired in the Usecase layer, through the
`clock.Clock` boundary, and passed inward.

> The `dashboard` feature resolves its own window inside its query service instead. That is the
> older of the two shapes and the one that diverges from the Time Handling Policy; this package is
> the shape to follow.

`Window` is a **query parameter**, not a reified Specification — it carries dates the user chose, the
same way `purchase.ListFeedParams` carries a keyset boundary. The business criteria that accompany
these reads (for example "cancelled purchases are excluded") stay named predicates where the value
lives, per the departure from Evans recorded in
[`internal/domain/README.md`](../../../domain/README.md).

## Behavior

|`Spec.Kind`|Required fields|Resolved window|
|---|---|---|
|`all` (or unset / unknown)|—|no filtering (`Filtered()` is `false`)|
|`month`|`Month` (`YYYY-MM`)|first day → last day of that month|
|`range`|`From`, `To`|the two dates as given, both ends inclusive|
|`recent`|`Days`|`Days` days before today → today, both ends inclusive|

- Every kind resolves against the injected `*time.Location`, never `time.Local` — the container's
  default is UTC, so relying on it would aggregate a different calendar day than the configuration
  asks for.
- **`now` is converted into the location; `From` / `To` are not.** `now` is an instant that has to be
  read as a local calendar day, while `From` / `To` are calendar days the user already named —
  converting them would shift the date backwards in any location west of UTC.
- The month end is derived as "first day of the next month, minus one day", so 28 / 29 / 31 fall out
  without a table of month lengths.
- `Bounds()` returns the half-open interval `[after, before)` that SQL predicates use, where `before`
  is the day *after* the end date. That is what includes orders placed at any time on the end date.
- Errors are `apperror.ErrInvalidArgument`: a kind's required field missing, a `Month` that does not
  parse, `To` before `From`, or `Days` below 1. The wire contract (OpenAPI) is stricter than this on
  purpose — the `Days` ceiling lives there, and these checks are the safety net for callers that
  reach the usecase directly.

## Usage

```go
// Controller: fill Spec straight from the query parameters, validating nothing.
spec := period.Spec{Kind: period.Kind(*params.Period), Days: ptr.Map(params.Days, ...)}

// Usecase: resolve once, then use the same window for the query and the response.
window, err := period.Resolve(spec, u.clk.Now(), u.loc)
if err != nil {
    return SummaryView{}, err
}
if window.Filtered() {
    after, before := window.Bounds() // [after, before) for the SQL predicate
}
```

Resolve once per request and pass the `Window` around. Re-resolving the same `Spec` later can land on
a different day if the request straddles midnight.

## Test strategy

The package has **no Repository and no Boundary of its own** — `Resolve` takes `now` as an argument
rather than reaching for a clock. The viewpoints in the Usecase layer's Testing Strategy that
concern mocked repositories, boundaries and transactions therefore do not apply here; they are out of
scope rather than missing. What is exercised instead is the calendar arithmetic: each kind's
dispatch, the month-end and leap-year boundaries, the location-dependent notion of "today" (including
a UTC instant that falls on the next day locally), and the half-open upper bound.
