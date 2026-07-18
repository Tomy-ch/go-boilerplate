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
- `WithUserID()` — returns a copy with the internal UserID resolved (identity resolution is separate from authentication)
- `HasUserID()` — returns true if the internal UserID has been resolved
- `UserID()` — returns the internal user UUID (`ErrUserIDUnresolved` if unresolved)
- `Issuer()` — returns the token issuer (e.g., `"mock"`, an IdP issuer)
- `Scopes()` — returns scope list (optional)
- `Claims()` — returns claims map (optional, for authorization / UI control)

`New()` produces the authenticator result (subject + issuer) with the UserID **unresolved**; resolving the internal user is a separate concern (an `IdentityResolver`-equivalent), applied via `WithUserID()`.

## Errors

|Error|Description|
|---|---|
|`ErrUnauthenticatedSubjectMissing`|Subject is empty (wraps `apperror.ErrUnauthenticated`)|
|`ErrUserIDUnresolved`|Internal UserID is unresolved (wraps `apperror.ErrUnauthenticated`)|
|`ErrTokenMissing`|Token is empty (wraps `apperror.ErrUnauthenticated`)|

## Design Intent

- Represent the "authenticated" state with types
- Push token parsing logic to the outer layer (Infrastructure)
- Separate authentication (subject/issuer extraction) from internal-user resolution (`WithUserID`)

## Implementation

`internal/infrastructure/auth/` provides environment-specific implementations of the `Authenticator` interface, which generates an `Authn` from a `Credential`.
