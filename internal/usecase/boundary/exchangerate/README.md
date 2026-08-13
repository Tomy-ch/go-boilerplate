# exchangerate

English | [日本語](README.ja.md)

Provides a `Gateway` interface that acts as a semantic gateway to an external
exchange-rate service (DTO-mode sample).

```go
type Gateway interface {
    GetRate(ctx context.Context, base, quote string) (*Rate, error)
}

type Rate struct {
    Base  string
    Quote string
    Value decimal.Decimal
    Date  string
}
```

`Value` is an exact `pkg/decimal.Decimal`, not a `float64`: the rate is a multiplier on the
money path and a float would corrupt it at ingest ([ADR-0036 (two-scale-quantity-model)](../../../../docs/adr/0036-two-scale-quantity-model.md)).

## Why Abstract?

- Ensure testability of usecases that depend on an external rate provider without real HTTP calls
- Allow mock substitution in tests for deterministic behavior

See [`internal/usecase/boundary/README.md`](../README.md#exchangerate) for why the port is semantic rather than `net/http` / SDK-shaped.

## Implementation

`internal/infrastructure/webapi/exchangerate/` provides the concrete implementation that calls the external service over HTTP.
