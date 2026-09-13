# datetime

Provides the wire representation of a time that this layer sends outward.

## Role

This package holds the one rule for spelling an instant on the wire: UTC, RFC 3339 with nanosecond
precision. Centralizing it means the delivery envelope and every event payload inside one frame read
the same way, instead of each builder deciding separately.

The rule exists because the alternative is not stable. `time.Time` carries a location, and the offset
a bare `Format` writes is therefore a function of the deployment's `TZ` — the same instant spells
`2026-09-01T10:00:00Z` in one environment and `2026-09-01T19:00:00+09:00` in another. A published
contract cannot have that property: a subscriber whose schema accepts only `Z` passes or fails
depending on where the API happens to run.

It resolves nothing and parses nothing. Reading a time back is the concern of whoever owns that
format — a paging cursor parses its own token, because that token is opaque and round-trips within
the deployment that issued it.

## Behavior

- Any location → the same instant, spelled in UTC
- The zero `time.Time` → spelled by the same rule (`0001-01-01T00:00:00Z`); whether a value is absent
  is the caller's judgment, not this package's

## Usage

```go
payload := created{
    OrderedAt: datetime.FormatUTC(p.OrderedAt()),
}
```

`internal/architest` fails an `<aggregate>/event/` package that formats a time itself instead of
calling this package — see `internal/usecase/README.md` § `event/payload_parity.yaml`.
