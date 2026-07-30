# auth

English | [日本語](README.ja.md)

Provides interfaces and value objects for authentication.

## Credential Details

`Credential` is a scheme-neutral representation of the inbound credential.

- `Scheme()` — returns the auth scheme (e.g., `"Bearer"`; see `SchemeBearer`)
- `Token()` — returns the token

## Authn Details

The identity core is Subject (token `sub`), Issuer (token issuer) and UserID (internal user id).

- `Subject()` — returns the authenticated subject (token `sub`)
- `WithUserID()` — returns a copy with the internal UserID resolved (identity resolution is separate from authentication); a zero-value UUID returns `ErrUserIDZero`
- `HasUserID()` — returns true if the internal UserID has been resolved
- `UserID()` — returns the internal user UUID (`ErrUserIDUnresolved` if unresolved)
- `Issuer()` — returns the token issuer (e.g., `"mock"`, an IdP issuer)
- `Scopes()` — returns scope list (optional)
- `Claims()` — returns claims map (optional, for authorization / UI control)

`New()` produces the authenticator result (subject + issuer) with the UserID **unresolved**; resolving the internal user is a separate concern (`IdentityResolver`), applied via `WithUserID()`.

`WithUserID()` rejects a zero-value UUID, so a resolved UserID is guaranteed to be non-zero. A zero value identifies no user, and letting it through would make every consumer that compares the caller against stored data (ownership checks in particular) responsible for rejecting it on its own.

## IdentityResolver Details

`IdentityResolver` resolves an authenticated external identity (`Issuer` + `Subject`) to an internal user, returning a copy of the `Authn` with the UserID resolved.

- `Resolve(ctx, *Authn) (*Authn, error)` — looks the identity up (by `Issuer` + `Subject`) and returns the `Authn` with `WithUserID()` applied
- No matching identity → `ErrIdentityNotFound`; the resolved user is unavailable (e.g. soft-deleted) → `ErrUserUnavailable` (both fail closed)
- Applied after authentication succeeds, so the token-verification concern (`Authenticator`) stays independent of user lookup

## Errors

|Error|Description|
|---|---|
|`ErrUnauthenticatedSubjectMissing`|Subject is empty (wraps `apperror.ErrUnauthenticated`)|
|`ErrUserIDUnresolved`|Internal UserID is unresolved (wraps `apperror.ErrUnauthenticated`)|
|`ErrUserIDZero`|`WithUserID()` was given a zero-value UUID (wraps `apperror.ErrUnauthenticated`)|
|`ErrTokenMissing`|Token is empty (wraps `apperror.ErrUnauthenticated`)|
|`ErrIdentityNotFound`|No internal user matches the issuer + subject (wraps `apperror.ErrUnauthenticated`)|
|`ErrUserUnavailable`|The resolved user is unavailable, e.g. soft-deleted (wraps `apperror.ErrUnauthenticated`)|

## Design Intent

- Represent the "authenticated" state with types
- Push token parsing logic to the outer layer (Infrastructure)
- Separate authentication (subject/issuer extraction) from internal-user resolution (`IdentityResolver` / `WithUserID`)
- Hold the "a resolved UserID is non-zero" invariant at the boundary that produces it, rather than in each consumer

## Implementation

`internal/infrastructure/auth/` provides environment-specific implementations of the `Authenticator` interface, which generates an `Authn` from a `Credential`. The default `IdentityResolver` (`internal/infrastructure/auth/identity/`) is a passthrough that leaves the UserID unresolved; provide a project-specific implementation to resolve the internal user from your own user store.
