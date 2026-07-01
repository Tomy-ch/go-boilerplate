# authz

English | [日本語](README.ja.md)

Provides interfaces and value objects for authorization (authz) — the counterpart to authentication (`auth`).

## Authorizer

`Authorize(ctx, *auth.Authn, Action, *Resource) error` decides whether the authenticated subject may perform `action` on `resource`. Returns `nil` to allow, or an error wrapping `apperror.ErrPermissionDenied` (`ErrForbidden`, HTTP 403) to deny.

## Types

- `Action` — the operation being authorized (e.g. `ActionUserDelete` = `"user:delete"`).
- `Resource` — the target resource. Carries `Kind()` and an optional `OwnerID()`, so ownership-based (object-level) decisions are expressible.

## Errors

|Error|Description|
|---|---|
|`ErrForbidden`|Authorization denied (wraps `apperror.ErrPermissionDenied`, HTTP 403)|

## Design Intent

- Make the authorization decision a **swappable policy (PDP)** rather than scattering ad-hoc checks across usecases.
- Pass the full `auth.Authn` (subject / scopes / claims) plus the target `Resource`, so both RBAC (roles from claims) and ownership (subject == OwnerID) models are expressible. This is why this `authz` boundary intentionally depends on the sibling `auth` boundary (the only inter-boundary dependency in this layer) — the decision needs the whole authenticated subject, not just its id.

### Convention: how a usecase receives the caller

- A usecase method that performs **object-level authorization** (acts on a resource named by the request, e.g. `GetUser` / `UpdateUser` / `DeleteUser` keyed by a path `user_id`) takes the full `*auth.Authn` and calls `Authorizer.Authorize(...)` first.
- A usecase method that only needs the **caller's own identity** as data (e.g. `CreateUser` / `ChangePassword`, which act on the authenticated user itself) takes a scalar `uuid.UUID` extracted in the controller (`authn.ID()`) instead — there is no separate object to authorize against.

## Implementation

`internal/infrastructure/authz/` provides implementations of the `Authorizer` interface. The default `allowall` implementation grants everything and is restricted to non-production environments.
