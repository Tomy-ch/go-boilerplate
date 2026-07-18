# jwt (Fixed Public Key JWT Authentication)

English | [日本語](README.ja.md)

This directory contains an `Authenticator` implementation that verifies an access token (JWT) with a **fixed RSA public key**. It is the production-oriented counterpart to the development-only `local` implementation, and covers the **de-facto standard verification core** only.

## Role

- Verify the signature and standard claims of an access token presented as a JWT
- Generate a verified `Authn` (subject + scopes + raw claims) on success
- Normalize every verification failure to `apperror.ErrUnauthenticated` (fail-closed)

## Verification Scope (Standard Core)

This implementation intentionally supports the **de-facto standard profile** only:

- Asymmetric signature via an **algorithm allowlist** (default `["RS256"]`, injectable). `alg=none` and symmetric algorithms such as `HS256` are always rejected to prevent key-confusion attacks.
- Signature verification with a fixed RSA public key (PEM parsed at construction; parse failure is a construction error).
- Claim validation: `iss` / `aud` / `exp` / `nbf` / `sub`. `exp` is required, and `aud` is required (standard profile).
- Clock-skew tolerance via an injectable `Leeway` (default 60s).
- Optional `typ` header check (`ExpectedType`, e.g. `at+jwt` per RFC 9068) to reject ID Token misuse; disabled when empty.
- Scope extraction from the OAuth2 standard `scope` claim (space-delimited string → `[]string`).

Time is injected via the `clock.Clock` boundary so that `exp` / `nbf` validation is deterministic in tests.

## Constructor

`New(Params)` returns the `Authenticator` (or a construction error). `Params` carries the verification parameters (public key PEM, issuer, audience, allowed algorithms, leeway, expected type, clock). Construction fails when the PEM is invalid or a required parameter (clock / issuer / audience) is missing — these are configuration errors, distinct from authentication errors.

## Error Handling

All verification failures are normalized to the `ErrJWTAuthenticatorInvalidToken` sentinel, which wraps `apperror.ErrUnauthenticated` (fail-closed). The underlying cause (signature mismatch / `exp` / `iss` / `aud` / `typ` etc.) is preserved in the error chain so operators can distinguish the failure reason from logs and traces, while callers only ever see a normalized `401`. This mirrors the `pgerror.NormalizeError` convention in `internal/infrastructure/rdb/pgerror`.

## Extension Points

The following are **out of scope** for the standard core and left to the template consumer to add when their IdP requires them:

- Cognito access-token dialect (`token_use=access` verification, `aud`→`client_id` substitution — Cognito access tokens carry no `aud`)
- Azure AD `scp` / `roles` claims
- Elliptic-curve keys (`ES256` etc.) — the current constructor parses an RSA public key
- Opaque (non-JWT) access tokens — out of scope; these IdPs are not supported by this implementation

## Notes

- JWKS-based dynamic key resolution is not handled here; it replaces the fixed public key in a later phase.
- Internal user-ID resolution (`sub` → application user ID via DB) is a separate concern handled by the identity-resolution phase; this implementation returns a verified-but-unresolved `Authn`.
