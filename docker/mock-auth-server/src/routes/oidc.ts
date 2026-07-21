// oidc.ts は OAuth2 / OIDC 標準エンドポイント（/oidc/*）を提供する。
// authorize（Authorization Code Flow + PKCE S256）・token（code 単回消費 + PKCE 検証で access/id token 発行）・
// userinfo（Bearer access token 必須・whitelist claim）・logout（RP-Initiated）を実装する。
import { Hono } from "hono";
import * as zod from "zod";
import { randomBytes } from "node:crypto";
import { config } from "../config.ts";
import { clients, findClient } from "../clients.ts";
import { defaultSubject, findUser } from "../users.ts";
import { codeStore, sessionStore } from "../store.ts";
import {
  issueAccessToken,
  issueIdToken,
  verifyAccessToken,
  subjectFromToken,
  ACCESS_TTL_SECONDS,
} from "../tokens.ts";
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
  // 要求 scope はクライアントに許可された scope の部分集合でなければならない（scope 昇格の防止）。
  const requestedScopes = q.scope.split(" ").filter((s) => s !== "");
  if (!requestedScopes.every((s) => client.scopes.includes(s))) {
    return redirectError("invalid_scope", "requested scope is not allowed for this client");
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
  const idToken = await issueIdToken(config, record.subject, record.nonce, record.clientId);
  logEvent("token_issued", { client_id: record.clientId, subject: record.subject, kid: KID });

  return c.json({
    access_token: accessToken,
    token_type: "Bearer",
    expires_in: ACCESS_TTL_SECONDS,
    scope: record.scope,
    id_token: idToken,
  });
});

// GET /oidc/userinfo — Bearer access token を必須とし、対応 subject の claim を whitelist で返す。
// ID Token（typ != at+jwt）は verifyAccessToken の typ 検証で拒否される。
oidcRoutes.get("/oidc/userinfo", async (c) => {
  const authz = c.req.header("authorization") ?? "";
  const match = /^Bearer (.+)$/.exec(authz);
  if (match === null) {
    return c.json({ error: "invalid_token", error_description: "missing bearer access token" }, 401);
  }

  let subject: string;
  try {
    const payload = await verifyAccessToken(config, match[1]);
    subject = typeof payload.sub === "string" ? payload.sub : "";
  } catch {
    logEvent("oidc_error", { endpoint: "userinfo", error: "invalid_token" });
    return c.json({ error: "invalid_token", error_description: "access token verification failed" }, 401);
  }

  // sub は必須。fixture に User があれば whitelist フィールドも返す（additionalProperties:false）。
  const user = findUser(subject);
  if (user === undefined) {
    return c.json({ sub: subject });
  }
  return c.json({
    sub: user.subject,
    email: user.email,
    email_verified: true,
    given_name: user.given_name,
    family_name: user.family_name,
    name: user.name,
  });
});

// POST /oidc/logout — RP-Initiated Logout。id_token_hint の subject の session を破棄し、
// post_logout_redirect_uri が登録済みなら 302（state 反映）、未指定なら 200、未登録なら 400。
oidcRoutes.post("/oidc/logout", async (c) => {
  const form = await c.req.parseBody();
  const idTokenHint = typeof form.id_token_hint === "string" ? form.id_token_hint : undefined;
  const postLogout = typeof form.post_logout_redirect_uri === "string" ? form.post_logout_redirect_uri : undefined;
  const state = typeof form.state === "string" ? form.state : undefined;

  if (idTokenHint !== undefined) {
    const subject = subjectFromToken(idTokenHint);
    if (subject !== undefined) {
      sessionStore.deleteWhere((s) => s.subject === subject);
    }
  }
  logEvent("logout", {});

  if (postLogout === undefined) {
    return c.json({ status: "logged_out" });
  }
  const registered = clients.some((cl) => cl.post_logout_redirect_uris.includes(postLogout));
  if (!registered) {
    return c.json({ error: "invalid_request", error_description: "unregistered post_logout_redirect_uri" }, 400);
  }
  const url = new URL(postLogout);
  if (state !== undefined) {
    url.searchParams.set("state", state);
  }
  return c.redirect(url.toString(), 302);
});
