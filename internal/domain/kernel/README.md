# kernel

English | [日本語](README.ja.md)

The domain **shared kernel** (DDD *Shared Kernel*): a deliberately small set of business-semantic
value objects shared across aggregates. Other `internal/domain/**` packages may import `kernel/`;
`kernel/` itself depends only on `pkg/**` and `internal/apperror`, never on an aggregate.

## Why this exists

A value object like `money.Price` is used by more than one aggregate (`product.price` now,
`purchase_details.unit_price` / `purchases.*_amount` next). It cannot live in one aggregate (the
others would have to reach in), and it cannot live in `pkg/` (which forbids business logic). So it
lives here — see [ADR-0104](../../../docs/adr/0104-domain-shared-kernel.md).

## Admission bar (this is NOT a `shared` / `common` junk drawer)

Add a package here **only when all** hold:

- it is a **value object / domain concept**, not an aggregate or an aggregate-specific service;
- it is genuinely used by **two or more aggregates** (or imminently so) — not "might be reused";
- it carries **business semantics** that bar it from `pkg/` (currency, non-negativity, minor-unit,
  tax, …) — if it's a context-independent utility, it belongs in `pkg/`;
- adding it is a **jointly-owned** decision — a kernel change ripples to every dependent aggregate,
  so evolve it conservatively.

If a type does not clear this bar, keep it in its owning aggregate.

## Enforcement

depguard (`maintain_a_sound_domain` in `.golangci-full.yaml`) denies `internal/domain/` for domain
files but allows `internal/domain/kernel`. So domain→kernel is permitted while
domain→other-aggregate is forbidden.

## Packages

- `money` — `Price` value object (non-negative price-scale decimal; owns minor-unit conversion).
  The exact decimal container is `pkg/decimal` ([ADR-0102](../../../docs/adr/0102-exact-decimal-pkg-wrap.md)).
