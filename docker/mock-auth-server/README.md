# mock-auth-server (JWT Test Provider)

English | [日本語](README.ja.md)

A **mock OIDC auth server** for local development. It lets the Go API (Resource Server) exercise **JWT signature and claim verification** end-to-end without a real IdP. Scope is **verification up to signature + claims** (Phase 5-A): `iss+sub → User` resolution, Roles, and the Authorization Code Flow are **out of scope** (later phases).

It is intentionally implemented in **TypeScript on Node.js (a different runtime than the Go API)** so that the token issuer and the verifier do not share bugs originating from a single library. The `.ts` sources run directly via Node's native type stripping (Node 24) — no `tsc` build step and no extra runtime dependency; `typescript` is a dev-only dependency for `npm run typecheck`.

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health` | Liveness — `{"status":"ok"}` |
| GET | `/.well-known/openid-configuration` | OIDC Discovery document |
| GET | `/.well-known/jwks.json` | JWKS (public key only: `kty=RSA` `alg=RS256` `use=sig` `kid=mock-key-1`) |
| POST | `/test/token` | Issue a token — `{subject, profile}` → `{access_token, token_type, expires_in}` |
| GET | `/test/users` | List the fixed user fixtures |

`/test/*` are **test-only endpoints**. Set `MOCK_AUTH_TEST_ENDPOINTS=disabled` to make them return `404` in production-like modes.

## Token Profiles (`/test/token`)

Anomalies are provided as **fixed profiles** (not an arbitrary-claim-injection API) for reproducibility. The Go API is expected to accept `valid` and reject every anomaly with `401`.

| Profile | Content | Go API |
| --- | --- | --- |
| `valid` | Standard access token (`typ=at+jwt`, correct `iss`/`aud`/`exp`/`nbf`/`sub`) | accept |
| `expired` | `exp` in the past | 401 |
| `not-yet-valid` | `nbf` in the future | 401 |
| `wrong-issuer` | `iss` mismatch | 401 |
| `wrong-audience` | `aud` mismatch | 401 |
| `missing-subject` | no `sub` | 401 |
| `invalid-signature` | signed with a key absent from the JWKS (same `kid`) | 401 |
| `unsupported-algorithm` | `HS256` (symmetric, outside the RS256 allowlist) | 401 |
| `id-token` | `token_use=id`, `aud=<client_id>`, `typ` not `at+jwt` | 401 |

The valid access-token claims follow the de-facto standard profile: `iss` / `sub` / `aud` / `exp` / `iat` / `nbf` / `jti` / `client_id` / `token_use=access` / `scope`. Access tokens carry a `typ=at+jwt` header (RFC 9068), which the Go verifier uses to reject ID Token misuse.

## Keys & Fixtures

- `keys/mock-key-1.pem` — a **fixed RSA private key** (unchanged across restarts, so issued tokens are reproducible). `kid=mock-key-1`.
- `fixtures/users.json` — the example users (Subject / Email / name in the A range; status is used from Phase 5-B onward). The mock runs even if the file is absent (`/test/token` accepts any `subject`). See `fixtures/README.md`.

## Usage

```sh
docker compose --profile development up mock_auth_server
curl -s http://localhost:4000/.well-known/openid-configuration
curl -s -X POST http://localhost:4000/test/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"user-john-doe","profile":"valid"}'
```

The Go API points `AUTH_ISSUER` at the host URL (`http://localhost:4000`) and `AUTH_JWKS_URL` at the internal service URL (`http://mock_auth_server:4000/.well-known/jwks.json`) — the **issuer is separated from the JWKS fetch URL** so `iss` stays browser/host-resolvable while key fetching uses the container hostname. See `env/README.md` (Auth section).

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `OIDC_PORT` | `4000` | Listen port |
| `OIDC_ISSUER` | `http://localhost:4000` | `iss` claim / OIDC issuer (host-resolvable URL) |
| `OIDC_AUDIENCE` | `go-boilerplate-api` | `aud` claim for access tokens |
| `OIDC_CLIENT_ID` | `go-boilerplate-client` | `client_id` claim / ID Token audience |
| `MOCK_AUTH_TEST_ENDPOINTS` | `enabled` | `disabled` turns `/test/*` into `404` |

## Out of Scope (later phases)

`user_identities` / Roles / deleted・unknown user (Phase 5-B), Key Rotation with multiple `kid` and `/test/rotate-key` (Phase 5-C), and the Authorization Code Flow / Login UI / `/userinfo` / `/logout` (Phase 5-D).
