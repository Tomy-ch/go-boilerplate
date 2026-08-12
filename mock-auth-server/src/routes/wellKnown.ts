// wellKnown.ts は OIDC Discovery（/.well-known/openid-configuration）と JWKS を提供する。
import { Hono } from "hono";
import { config } from "../config.ts";
import { keyStore, ALG } from "../keys.ts";

// issuer / jwks_uri は Go 側認証が依存する契約のため不変（バイト等価）に保つ。
function discoveryDocument() {
  return {
    issuer: config.issuer,
    jwks_uri: `${config.issuer}/.well-known/jwks.json`,
    authorization_endpoint: `${config.issuer}/oidc/authorize`,
    token_endpoint: `${config.issuer}/oidc/token`,
    userinfo_endpoint: `${config.issuer}/oidc/userinfo`,
    end_session_endpoint: `${config.issuer}/oidc/logout`,
    response_types_supported: ["code"],
    grant_types_supported: ["authorization_code"],
    subject_types_supported: ["public"],
    id_token_signing_alg_values_supported: [ALG],
    scopes_supported: ["openid", "profile", "email", "api.read", "api.write"],
    // public client（PKCE）のみを許可するため none。
    token_endpoint_auth_methods_supported: ["none"],
    code_challenge_methods_supported: ["S256"],
    claims_supported: ["sub", "iss", "aud", "exp", "iat", "nbf", "email", "name"],
  };
}

export const wellKnownRoutes = new Hono();

wellKnownRoutes.get("/.well-known/openid-configuration", (c) => c.json(discoveryDocument()));
// JWKS は現在の公開集合から都度組み立てる（鍵ローテーションを反映）。
wellKnownRoutes.get("/.well-known/jwks.json", (c) => c.json(keyStore.jwks()));
