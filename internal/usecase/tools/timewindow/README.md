# timewindow

English | [日本語](README.ja.md)

Provides the half-open interval that bounds an aggregation or filter by ordered time.

## Role

This package holds the one shape every period-bounded read shares: a lower bound that is included, an
upper bound that is not, and the rule that rejects an empty interval. Centralizing it means every
period-bounded read agrees on what "no bound" and "this instant" mean, instead of each usecase
deciding separately.

It resolves nothing. A relative period — "today", "this month", "the last 30 days" — is resolved by the
caller into instants before it reaches this package, so the server holds neither a clock nor a calendar and
the same URL always names the same interval. That is the whole reason the type carries instants rather
than calendar days: a calendar day is meaningless without a timezone, and a timezone in the aggregation
path is what let one endpoint's "month" mean something different from another's.

The interval is half-open rather than closed because an inclusive upper bound cannot express "up to and
including this instant" at any finite precision — pick one and two adjacent windows either overlap on the
boundary instant or leave a gap around it. A half-open upper bound lets them meet exactly, so windows can
be tiled without double-counting and without losing a row between them.

## Behavior

- Both bounds omitted → the zero `Window`, meaning the whole period
- One bound omitted → that side is unbounded
- `Before` at or before `After` → `apperror.ErrInvalidArgument`

An empty interval is rejected rather than allowed to return zero rows, because a caller cannot otherwise
tell "nothing matched" from "I passed the bounds the wrong way round".

## Usage

```go
window, err := timewindow.New(timewindow.Bounds{After: params.OrderedAfter, Before: params.OrderedBefore})
if err != nil {
    return nil, err
}
// window.After() / window.Before() return nil for an unbounded side.
```
