# mock-auth-server (JWT Test Provider)

English | [日本語](README.ja.md)

A **mock OIDC auth server** for local development. It lets the Go API (Resource Server) exercise **JWT signature/claim verification and the Authorization Code Flow (+ PKCE S256)** end-to-end without a real IdP. Login UI / consent / refresh token / Roles are out of scope.

It is intentionally implemented in **TypeScript on Node.js (a different runtime than the Go API)** so that the token issuer and the verifier do not share bugs originating from a single library. The `.ts` sources run directly via Node's native type stripping (Node 24) — no `tsc` build step; the HTTP layer is [Hono](https://hono.dev/). `typescript` is a dev-only dependency for `npm run typecheck`, and `node --test` runs the suite.

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health` | Liveness — `{"status":"ok"}` |
| GET | `/.well-known/openid-configuration` | OIDC Discovery document |
| GET | `/.well-known/jwks.json` | JWKS (public key only: `kty=RSA` `alg=RS256` `use=sig` `kid=mock-key-1`) |
| GET | `/oidc/authorize` | Authorization endpoint (Code Flow + PKCE S256) — `302` with `code`, or error `redirect` / `400` |
| POST | `/oidc/token` | Token endpoint (`authorization_code` + PKCE) — issues `access_token` + `id_token` |
| GET | `/oidc/userinfo` | UserInfo (Bearer access token; whitelist claims). Rejects ID Tokens with `401` |
| POST | `/oidc/logout` | RP-Initiated Logout — validates `post_logout_redirect_uri`, echoes `state` |
| POST | `/bypass/token` | Issue a token by fixed profile — `{subject, profile}` → `{access_token, ...}` |
| POST | `/bypass/session` | Create a login session directly — `{subject}` → `{session_id, subject}` |
| GET | `/admin/users` | List the fixed user fixtures |
| POST | `/admin/reset` | Reset the volatile stores (code / session) |
| POST | `/admin/keys/rotate` | Key rotation — contract only, returns `501` |

`/bypass/*` and `/admin/*` are **dev-only endpoints**: set `MOCK_AUTH_DEV_ENDPOINTS=disabled` to make them return `404`. The server also **refuses to start when `NODE_ENV=production`** (this mock must never run in production). Registered clients (`fixtures/clients.json`) are public + PKCE-required; `redirect_uri` / `post_logout_redirect_uri` are matched exactly.

## Token Profiles (`/bypass/token`)

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
- `fixtures/users.json` — the example users (Subject / Email / name; `status` is not yet consumed). The mock runs even if the file is absent (`/bypass/token` accepts any `subject`). See `fixtures/README.md`.
- `fixtures/clients.json` — the registered OAuth clients (public client, PKCE required, allowed `redirect_uris` / `post_logout_redirect_uris`).

## Usage

```sh
docker compose --profile development up mock_auth_server
curl -s http://localhost:4000/.well-known/openid-configuration
curl -s -X POST http://localhost:4000/bypass/token \
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
| `MOCK_AUTH_DEV_ENDPOINTS` | `enabled` | `disabled` turns `/bypass/*` · `/admin/*` into `404` |
| `NODE_ENV` | (unset) | `production` makes the server exit immediately (must not run in production) |

## OpenAPI & Codegen

The HTTP surface is defined OpenAPI-first under `openapi/` (bundled to `openapi/openapi.gen.yaml`); `src/generated/schemas.ts` holds the zod schemas generated by [orval](https://orval.dev/). Regenerate with `make gen-mock-auth-oapi`; both artifacts are committed and CI checks for drift.

## Not Yet Implemented

Key rotation with multiple `kid` (`/admin/keys/rotate` is a `501` stub), and Roles / `user_identities` / deleted・unknown-user handling.
