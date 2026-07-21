// oidc.ts は OAuth2 / OIDC 標準エンドポイント（/oidc/*）を提供する。
// authorize（Authorization Code Flow + PKCE）と token（code 単回消費 + PKCE S256 検証）を実装する。
// userinfo / logout は後続 Increment で実装する（契約は openapi/ に定義済みのため 501 = 定義済み・未実装）。
import { Hono } from "hono";
import * as zod from "zod";
import { randomBytes } from "node:crypto";
import { config } from "../config.ts";
import { findClient } from "../clients.ts";
import { defaultSubject } from "../users.ts";
import { codeStore } from "../store.ts";
import { issueAccessToken, issueIdToken, ACCESS_TTL_SECONDS } from "../tokens.ts";
import { KID } from "../keys.ts";
import { verifyS256 } from "../pkce.ts";
import { logEvent } from "../log.ts";

export const oidcRoutes = new Hono();

// GET /oidc/authorize — 認可リクエスト（Authorization Code Flow + PKCE S256）。
// 検証順: (1) client_id / redirect_uri を完全一致で検証しリダイレクト不能なら 400（no-redirect）、
// (2) 以降の不正は redirect_uri へ 302 error redirect（state を反映）、(3) 成功なら code を発行して 302。
// Login UI は対象外のため subject は固定（自動ログイン相当）。
oidcRoutes.get("/oidc/authorize", (c) => {
  const q = c.req.query();
  const clientId = q.client_id;
  const redirectUri = q.redirect_uri;

  const client = clientId ? findClient(clientId) : undefined;
  if (client === undefined || redirectUri === undefined || !client.redirect_uris.includes(redirectUri)) {
    logEvent("oidc_error", { endpoint: "authorize", error: "invalid_request", client_id: clientId });
    return c.json({ error: "invalid_request", error_description: "unknown client_id or unregistered redirect_uri" }, 400);
  }

  const state = q.state;
  const redirectError = (error: string, description: string) => {
    const url = new URL(redirectUri);
    url.searchParams.set("error", error);
    url.searchParams.set("error_description", description);
    if (state !== undefined) {
      url.searchParams.set("state", state);
    }
    logEvent("oidc_error", { endpoint: "authorize", error, client_id: clientId });
    return c.redirect(url.toString(), 302);
  };

  if (q.response_type !== "code") {
    return redirectError("unsupported_response_type", "response_type must be code");
  }
  if (q.scope === undefined || q.scope === "") {
    return redirectError("invalid_request", "scope is required");
  }
  if (q.code_challenge === undefined || q.code_challenge === "") {
    return redirectError("invalid_request", "code_challenge is required");
  }
  if (q.code_challenge_method !== "S256") {
    return redirectError("invalid_request", "code_challenge_method must be S256");
  }

  const code = randomBytes(32).toString("base64url");
  codeStore.set(code, {
    clientId,
    redirectUri,
    subject: defaultSubject,
    scope: q.scope,
    codeChallenge: q.code_challenge,
    nonce: q.nonce,
    state,
  });
  logEvent("code_issued", { client_id: clientId, subject: defaultSubject });

  const url = new URL(redirectUri);
  url.searchParams.set("code", code);
  if (state !== undefined) {
    url.searchParams.set("state", state);
  }
  return c.redirect(url.toString(), 302);
});

// tokenBodySchema は /oidc/token の form-urlencoded 入力を runtime 検証する（zod I/O 検証）。
const tokenBodySchema = zod.object({
  grant_type: zod.string(),
  code: zod.string(),
  redirect_uri: zod.string(),
  client_id: zod.string(),
  code_verifier: zod.string(),
});

// POST /oidc/token — authorization_code グラント。code を単回 consume し、PKCE(S256) を検証して
// access token + id token を発行する（nonce を id token に反映）。public client のため client 認証は none。
oidcRoutes.post("/oidc/token", async (c) => {
  const parsed = tokenBodySchema.safeParse(await c.req.parseBody());
  if (!parsed.success) {
    return c.json({ error: "invalid_request", error_description: "missing or malformed token request" }, 400);
  }
  const body = parsed.data;

  if (body.grant_type !== "authorization_code") {
    return c.json({ error: "unsupported_grant_type", error_description: "only authorization_code is supported" }, 400);
  }

  // code を単回消費する（再利用・期限切れは take が undefined を返す）。
  const record = codeStore.take(body.code);
  if (record === undefined) {
    logEvent("oidc_error", { endpoint: "token", error: "invalid_grant" });
    return c.json({ error: "invalid_grant", error_description: "code is invalid, expired, or already used" }, 400);
  }
  if (body.client_id !== record.clientId || body.redirect_uri !== record.redirectUri) {
    logEvent("oidc_error", { endpoint: "token", error: "invalid_grant", client_id: body.client_id });
    return c.json({ error: "invalid_grant", error_description: "client_id or redirect_uri mismatch" }, 400);
  }
  if (!verifyS256(body.code_verifier, record.codeChallenge)) {
    logEvent("oidc_error", { endpoint: "token", error: "invalid_grant", client_id: record.clientId });
    return c.json({ error: "invalid_grant", error_description: "PKCE verification failed" }, 400);
  }

  const accessToken = await issueAccessToken(config, record.subject, record.scope);
  const idToken = await issueIdToken(config, record.subject, record.nonce);
  logEvent("token_issued", { client_id: record.clientId, subject: record.subject, kid: KID });

  return c.json({
    access_token: accessToken,
    token_type: "Bearer",
    expires_in: ACCESS_TTL_SECONDS,
    scope: record.scope,
    id_token: idToken,
  });
});

// userinfo / logout は後続 Increment で実装する。
oidcRoutes.get("/oidc/userinfo", (c) => c.json({ error: "not_implemented" }, 501));
oidcRoutes.post("/oidc/logout", (c) => c.json({ error: "not_implemented" }, 501));
