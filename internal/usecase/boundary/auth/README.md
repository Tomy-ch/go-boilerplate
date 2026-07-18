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
- `HasUserID()` — returns true if subject was parseable as UUID
- `UserID()` — returns the internal user UUID (error if not parseable)
- `Issuer()` — returns the token issuer (e.g., `"mock"`, an IdP issuer)
- `Scopes()` — returns scope list (optional)
- `Claims()` — returns claims map (optional, for authorization / UI control)

## Errors

|Error|Description|
|---|---|
|`ErrUnauthenticatedSubjectMissing`|Subject is empty (wraps `apperror.ErrUnauthenticated`)|
|`ErrSubjectNotUUID`|Subject cannot be parsed as UUID (wraps `apperror.ErrValidation`)|
|`ErrTokenMissing`|Token is empty (wraps `apperror.ErrInvalidArgument`)|

## Design Intent

- Represent the "authenticated" state with types
- Push token parsing logic to the outer layer (Infrastructure)
- Keep subject normalization (trim) and UUID conversion encapsulated

## Implementation

`internal/infrastructure/auth/` provides environment-specific implementations of the `Authenticator` interface, which generates an `Authn` from a `Credential`.
