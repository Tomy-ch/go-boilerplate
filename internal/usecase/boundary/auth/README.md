# auth

English | [日本語](README.ja.md)

Provides interfaces and value objects for authentication.

## Public API

|Type / Function|Description|
|---|---|
|`Authenticator`|Interface: generate `Authn` from `Credential`|
|`Authn`|Authentication result (subject / id / provider / scopes / claims)|
|`New(subject, provider, scopes, claims)`|Create `Authn` (empty subject → `ErrUnauthorizedSubjectMissing`)|
|`Credential`|Value object holding access token|
|`NewCredential(accessToken)`|Create `Credential` (empty token → `ErrArgumentTokenMissing`)|

## Authn Details

- `Subject()` — returns the authenticated subject (e.g., userID)
- `HasID()` — returns true if subject was parseable as UUID
- `ID()` — returns UUID (error if not parseable)
- `Provider()` — returns the auth provider name (e.g., "mock", "google")
- `Scopes()` — returns scope list (optional)
- `Claims()` — returns claims map (optional, for authorization / UI control)

## Errors

|Error|Description|
|---|---|
|`ErrUnauthorizedSubjectMissing`|Subject is empty (wraps `apperror.ErrUnauthenticated`)|
|`ErrInvalidIDMissing`|Subject cannot be parsed as UUID (wraps `apperror.ErrValidation`)|
|`ErrArgumentTokenMissing`|Access token is empty (wraps `apperror.ErrInvalidArgument`)|

## Design Intent

- Represent the "authenticated" state with types
- Push token parsing logic to the outer layer (Infrastructure)
- Keep subject normalization (trim) and UUID conversion encapsulated

## Implementation

`internal/infrastructure/auth/` provides environment-specific implementations.
