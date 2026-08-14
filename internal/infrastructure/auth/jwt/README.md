# jwt (JWT Authentication)

English | [日本語](README.ja.md)

This directory contains an `Authenticator` implementation that verifies an access token (JWT). The signing key is resolved either from a **fixed RSA public key** (`New`) or **dynamically from a JWKS endpoint by `kid`** (`NewJWKS`). It is the production-oriented counterpart to the development-only `local` implementation, and covers the **de-facto standard verification core** only.

## Role

- Verify the signature and standard claims of an access token presented as a JWT
- Generate a verified `Authn` (subject + scopes + raw claims) on success
- Normalize every verification failure to `apperror.ErrUnauthenticated` (fail-closed), and leave an
  inability to verify classified as the infrastructure failure it is

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
- `NewJWKS(JWKSParams, httpclient.Client)` — dynamic key resolution from a JWKS endpoint (`kid` lookup, TTL cache, lazy fetch). The JWK Set is parsed with [`github.com/go-jose/go-jose/v4`](https://github.com/go-jose/go-jose) and fetched through the resilient `httpclient` substrate (`net/http` is banned in the infrastructure layer), so the HTTP timeout / retry / circuit breaker / budget come from the `jwks` downstream profile (`NewDownstreamProfile`), not a param. `JWKSParams` embeds `Params` (PublicKeyPEM unused) and adds the JWKS URL, cache TTL, discovery TTL, and unknown-`kid` cooldown. When the JWKS URL is empty, the `jwks_uri` is derived from the issuer via OIDC discovery (issuer strict-match + https + same-origin, cached on its own TTL); a non-empty URL is used verbatim as an override (split-horizon). Fetch is lazy (on first use / cache miss), so no background goroutine or lifecycle binding is needed.
- `NewWithKeyResolver(Params, KeyResolver)` — the underlying seam that accepts any `KeyResolver` (`ResolveKey(ctx, kid) (crypto.PublicKey, error)`, an in-package interface) directly. It decouples key resolution from the JWT library and propagates `ctx` into the fetch; `New` injects a fixed-key resolver, `NewJWKS` a JWKS-backed one, and tests inject a double.

Construction fails when a required parameter (clock / issuer / audience) is missing — these are configuration errors, distinct from authentication errors.

## Error Handling

All verification failures are normalized to the `ErrJWTAuthenticatorInvalidToken` sentinel, which wraps `apperror.ErrUnauthenticated` (fail-closed). The underlying cause (signature mismatch / `exp` / `iss` / `aud` / `typ` etc.) is preserved in the error chain so operators can distinguish the failure reason from logs and traces, while callers only ever see a normalized `401`. This mirrors the `pgerror.NormalizeError` convention in `internal/infrastructure/rdb/pgerror`.

A failure to *carry out* the verification is not normalized. When the signing key cannot be fetched, when the fetched key set is unusable as a whole (malformed JSON, no usable signing key, a duplicate `kid` that disqualifies the document), or when the caller's context ends, the error keeps its `apperror.ErrUnavailable` / `apperror.ErrCanceled` classification and is returned as-is, because it says nothing about the token: reporting it as `401` would tell the client to fix a credential that was never examined. Fail-closed is unaffected — no such error grants access.

The line runs between the document and the `kid`. A `kid` that is simply absent from a valid key set is a verdict about *that* token and stays a `401`; a key set that cannot be used at all blocks every token equally and is a service failure.

## Extension Points

The following are **out of scope** for the standard core and are added by the consuming project when their IdP requires them:

- Cognito access-token dialect (`token_use=access` verification, `aud`→`client_id` substitution — Cognito access tokens carry no `aud`)
- Azure AD `scp` / `roles` claims
- Elliptic-curve keys (`ES256` etc.) — the current constructor parses an RSA public key
- Opaque (non-JWT) access tokens — out of scope; these IdPs are not supported by this implementation

## Notes

- JWKS key resolution (`NewJWKS`) verifies the same standard claim set as the fixed-key path; only the key source differs. JWK parsing is delegated to `go-jose/v4` and the fetch to the `httpclient` substrate; the `kid` lookup + TTL cache + unknown-`kid` refetch (throttled by a cooldown) are kept in-package, and the RSA signing-method guard is still applied on top (key-confusion defense). The JWK Set is filtered to signing keys (`use=sig`, RSA, unique `kid` — a duplicate `kid` rejects the whole document), so a JWKS that publishes multiple `kid`s (key rotation) is handled.
- Key rotation is handled without re-fetching on every request: a known `kid` is served from the TTL cache, and concurrent fetches collapse to one (fetch serialized + cooldown-deduplicated). A `kid` confirmed absent within the current cache generation is remembered in a **negative cache** so repeated bogus/`kid` probes don't each trigger a refetch; the negative set is discarded whenever a successful fetch changes the published key set, and it never applies to a stale cache — so a `kid` added by a rotation is still picked up on the next fetch (bounded by the cache TTL) rather than being permanently rejected. A `kid` dropped by a rotation is rejected once the cache generation refreshes and the key is gone.
- The refetch runs detached from the request that triggered it. Updating the key set is work shared by every request, so letting the lifetime of whichever request happened to trigger it govern the fetch would leave a disconnected attempt unrecorded — and a caller disconnecting on purpose could then drive refetches faster than the cooldown allows. The caller still leaves on its own budget (cancellation surfaces as `499`, an exhausted deadline as `503`); the fetch finishes on the substrate's own timeout and records its outcome, so a disconnected attempt warms the cache for the next request instead of being wasted.
- Internal user-ID resolution (`sub` → application user ID via DB) is a separate concern handled by the identity-resolution phase; this implementation returns a verified-but-unresolved `Authn`.
