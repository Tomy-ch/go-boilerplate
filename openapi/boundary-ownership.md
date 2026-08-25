# Input Boundary Value Ownership

English | [日本語](boundary-ownership.ja.md)

A constraint like `maxLength: 50` looks like a single fact, but the **same boundary value can live in several layers, owned by different concerns**. This guide defines who owns what, so that an OpenAPI constraint is **not mistaken for the domain's business rule**.

> [!IMPORTANT]
> OpenAPI's `minLength` / `maxLength` / `minimum` / `maximum` / `pattern` express the **wire contract** — what the HTTP API accepts and promises over the network. They are **not** the source of truth for the domain's business rules. The numbers may legitimately differ. Do not read an OpenAPI constraint as "this is the domain limit."

## Two different owners for the "same" number

|Concern|Owner|Where it lives|What it means|
|---|---|---|---|
|**Wire contract**|OpenAPI|`openapi/components/schemas/*.yaml`|What the API accepts (request) / promises (response) over HTTP|
|**Business rule**|domain|`internal/domain/<aggregate>/constant.go`|What value the business considers valid|
|**Storage capacity**|DB|`database/migrations/*.sql`|Physical column limit (e.g. `VARCHAR(100)`)|

These often share a number by coincidence, but they answer **different questions** and are changed for **different reasons**. OpenAPI does not own the domain's value; it only declares the contract the wire must satisfy.

## The direction invariant

The relationship between the layers is **asymmetric by direction**:

```text
OpenAPI request constraint  ⊆  domain rule  ⊆  OpenAPI response capacity
        (tightest)                                    (loosest)
```

- **Request** — OpenAPI may be *stricter* than domain. The request-validation middleware (`internal/controller/httpstack/oapi/`) rejects out-of-range input **before** the domain sees it, so a stricter wire limit is the safe direction.
- **Response** — the OpenAPI response constraint must be a **superset** of what the domain can emit. If the domain (or any non-HTTP write path) can produce a value the response schema forbids, the server emits a contract violation that **nothing on the server catches** (there is no runtime response validation — see [`internal/controller/httpstack/oapi/README`](../internal/controller/httpstack/oapi)). The only place it surfaces is the client's generated validation (e.g. `orval` + `zod`), which is the wrong side to discover a server-side contract break.

## Rules for maintainers

- **Do not** copy a domain constant into OpenAPI (or vice versa) and assume they must stay equal. Decide each value from its own concern.
- **Do** keep the direction invariant: `request ⊆ domain ⊆ response capacity`.
- When a request constraint is **stricter** than the domain rule, that is fine (middleware rejects first).
- When tightening a **response** constraint, confirm no write path can emit a value outside it.
- If you want CI to guard the invariant, add a test that reads `openapi.gen.yaml` and the domain constants and asserts `request ≤ domain ≤ response`.

## Where each value lives

|Layer|Path|
|---|---|
|OpenAPI request / response constraints|`openapi/components/schemas/*.yaml`|
|domain boundary constants|`internal/domain/<aggregate>/constant.go`|
|DB column limits|`database/migrations/*.sql`|
|Client-side validation (consumer's own concern)|generated from this spec (e.g. `orval` + `zod`)|
