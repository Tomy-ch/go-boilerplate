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
- Pass the full `auth.Authn` (subject / scopes / claims) plus the target `Resource`, so both RBAC (roles from claims) and ownership (subject == OwnerID) models are expressible.

## Implementation

`internal/infrastructure/authz/` provides implementations of the `Authorizer` interface. The default `allowall` implementation grants everything and is restricted to non-production environments.
