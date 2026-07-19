# jwt (JWT Authentication)

English | [日本語](README.ja.md)

This directory contains an `Authenticator` implementation that verifies an access token (JWT). The signing key is resolved either from a **fixed RSA public key** (`New`) or **dynamically from a JWKS endpoint by `kid`** (`NewJWKS`). It is the production-oriented counterpart to the development-only `local` implementation, and covers the **de-facto standard verification core** only.

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

## Constructors

The signing key is resolved through an injected `keyResolver`; the claim-verification logic is shared across every constructor.

- `New(Params)` — fixed RSA public key. `Params` carries the verification parameters (public key PEM, issuer, audience, allowed algorithms, leeway, expected type, clock). Fails when the PEM is invalid.
- `NewJWKS(JWKSParams, httpclient.Client)` — dynamic key resolution from a JWKS endpoint (`kid` lookup, TTL cache, lazy fetch). The JWK Set is parsed with [`github.com/go-jose/go-jose/v4`](https://github.com/go-jose/go-jose) and fetched through the resilient `httpclient` substrate (`net/http` is banned in the infrastructure layer), so the HTTP timeout / retry / circuit breaker / budget come from the `jwks` downstream profile (`NewDownstreamProfile`), not a param. `JWKSParams` embeds `Params` (PublicKeyPEM unused) and adds the JWKS URL and cache TTL. Fetch is lazy (on first use / cache miss), so no background goroutine or lifecycle binding is needed.
- `NewWithKeyfunc(Params, keyResolver)` — the underlying seam that accepts any `jwt.Keyfunc` resolver directly (used to inject a JWKS-backed resolver or a test double).

Construction fails when a required parameter (clock / issuer / audience) is missing — these are configuration errors, distinct from authentication errors.

## Error Handling

All verification failures are normalized to the `ErrJWTAuthenticatorInvalidToken` sentinel, which wraps `apperror.ErrUnauthenticated` (fail-closed). The underlying cause (signature mismatch / `exp` / `iss` / `aud` / `typ` etc.) is preserved in the error chain so operators can distinguish the failure reason from logs and traces, while callers only ever see a normalized `401`. This mirrors the `pgerror.NormalizeError` convention in `internal/infrastructure/rdb/pgerror`.

## Extension Points

The following are **out of scope** for the standard core and left to the template consumer to add when their IdP requires them:

- Cognito access-token dialect (`token_use=access` verification, `aud`→`client_id` substitution — Cognito access tokens carry no `aud`)
- Azure AD `scp` / `roles` claims
- Elliptic-curve keys (`ES256` etc.) — the current constructor parses an RSA public key
- Opaque (non-JWT) access tokens — out of scope; these IdPs are not supported by this implementation

## Notes

- JWKS key resolution (`NewJWKS`) verifies the same standard claim set as the fixed-key path; only the key source differs. JWK parsing is delegated to `go-jose/v4` and the fetch to the `httpclient` substrate; the `kid` lookup + TTL cache are kept in-package, and the RSA signing-method guard is still applied on top (key-confusion defense). Multi-`kid` rotation is a later phase.
- Internal user-ID resolution (`sub` → application user ID via DB) is a separate concern handled by the identity-resolution phase; this implementation returns a verified-but-unresolved `Authn`.
