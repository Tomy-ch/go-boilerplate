# local (Simple Authentication for Local Development)

English | [日本語](README.ja.md)

This directory contains a simple `Authenticator` implementation intended for use in local development and CI / test environments. It is a lightweight implementation for emulating external authentication providers, and is **not intended for production use**.

## Role

- Provide authentication information (via `Authenticator` interface) for quick verification during development
- Return a simply authenticated subject (`Authn`) as a substitute for production authentication services

## Public API

|Function / Type|Description|
|---|---|
|`New()`|Create a `LocalMockAuthenticator` (returns `authbd.Authenticator`)|
|`Authenticate(ctx, cred)`|Extract subject from token and return `Authn`|
|`ErrLocalMockAuthenticatorInvalidToken`|Error for invalid/empty token|

## Token Format

```text
Authorization: Bearer debug:user123
```

- Prefix `debug:` is stripped to extract the subject
- Subject: `user123`, Provider: `mock`
- No signature verification is performed

## Replacing for Production

Edit `provideAuthenticator` in `internal/di/module/core/auth.go` to switch the DI-registered implementation based on environment (local / stg / prd).

## Notes

- This implementation does not guarantee security (no signature verification, token expiry check, or replay prevention)
- Do not use in production
- Security-related settings or hardcoded values in this code must not be carried into production environments
