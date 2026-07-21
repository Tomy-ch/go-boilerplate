// oidc.test.ts は Authorization Code Flow + PKCE を app.request（プロセス内）で e2e 検証する。
import { test } from "node:test";
import assert from "node:assert/strict";
import { createApp } from "../router.ts";
import { s256Challenge } from "../pkce.ts";
import { sessionStore } from "../store.ts";

const CLIENT_ID = "go-boilerplate-client";
const REDIRECT_URI = "http://localhost:3000/api/auth/callback";
const VERIFIER = "test-code-verifier-abcdefghijklmnopqrstuvwxyz-0123456789";
const CHALLENGE = s256Challenge(VERIFIER);

function authorizePath(params: Record<string, string>): string {
  const search = new URLSearchParams(params);
  return `/oidc/authorize?${search.toString()}`;
}

function decodeJwtPart(part: string): Record<string, unknown> {
  return JSON.parse(Buffer.from(part, "base64url").toString("utf8")) as Record<string, unknown>;
}

async function issueCode(
  app: ReturnType<typeof createApp>,
  extra: Record<string, string> = {},
): Promise<URL> {
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      scope: "openid profile",
      code_challenge: CHALLENGE,
      code_challenge_method: "S256",
      ...extra,
    }),
  );
  assert.equal(res.status, 302);
  return new URL(res.headers.get("location") ?? "");
}

async function tokenRequest(app: ReturnType<typeof createApp>, form: Record<string, string>): Promise<Response> {
  return app.request("/oidc/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams(form).toString(),
  });
}

async function logoutRequest(app: ReturnType<typeof createApp>, form: Record<string, string>): Promise<Response> {
  return app.request("/oidc/logout", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams(form).toString(),
  });
}

// issueTokens は authorize→token を通し access_token / id_token を取り出す（subject は defaultSubject）。
async function issueTokens(
  app: ReturnType<typeof createApp>,
): Promise<{ access_token: string; id_token: string }> {
  const location = await issueCode(app);
  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: location.searchParams.get("code") ?? "",
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  });
  return (await res.json()) as { access_token: string; id_token: string };
}

test("正常系: authorize→token で access/id token を発行し claim が正しい", async () => {
  const app = createApp();
  const location = await issueCode(app, { state: "state-xyz", nonce: "nonce-123" });
  const code = location.searchParams.get("code");
  assert.ok(code);
  assert.equal(location.searchParams.get("state"), "state-xyz");

  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: code ?? "",
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  });
  assert.equal(res.status, 200);
  const body = (await res.json()) as Record<string, string>;
  assert.equal(body.token_type, "Bearer");
  assert.ok(body.access_token);
  assert.ok(body.id_token);

  const [atHeader, atClaims] = body.access_token.split(".");
  assert.equal(decodeJwtPart(atHeader).typ, "at+jwt");
  assert.equal(decodeJwtPart(atHeader).kid, "mock-key-1");
  assert.equal(decodeJwtPart(atClaims).aud, "go-boilerplate-api");
  assert.equal(decodeJwtPart(atClaims).token_use, "access");

  const idClaims = decodeJwtPart(body.id_token.split(".")[1]);
  assert.equal(idClaims.aud, CLIENT_ID);
  assert.equal(idClaims.token_use, "id");
  assert.equal(idClaims.nonce, "nonce-123");
});

test("異常系: code の再利用は 400 invalid_grant で拒否する", async () => {
  const app = createApp();
  const location = await issueCode(app);
  const code = location.searchParams.get("code") ?? "";
  const form = {
    grant_type: "authorization_code",
    code,
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  };
  assert.equal((await tokenRequest(app, form)).status, 200);
  const reuse = await tokenRequest(app, form);
  assert.equal(reuse.status, 400);
  assert.equal(((await reuse.json()) as Record<string, string>).error, "invalid_grant");
});

test("異常系: code_verifier 不一致は 400 invalid_grant で拒否する", async () => {
  const app = createApp();
  const location = await issueCode(app);
  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: location.searchParams.get("code") ?? "",
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: "wrong-verifier",
  });
  assert.equal(res.status, 400);
  assert.equal(((await res.json()) as Record<string, string>).error, "invalid_grant");
});

test("異常系: 未登録 redirect_uri は 400（リダイレクト不能）で拒否する", async () => {
  const app = createApp();
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: "http://evil.example.com/callback",
      scope: "openid",
      code_challenge: CHALLENGE,
      code_challenge_method: "S256",
    }),
  );
  assert.equal(res.status, 400);
});

test("異常系: code_challenge_method!=S256 は 302 error redirect（state を反映）", async () => {
  const app = createApp();
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      scope: "openid",
      code_challenge: CHALLENGE,
      code_challenge_method: "plain",
      state: "state-abc",
    }),
  );
  assert.equal(res.status, 302);
  const location = new URL(res.headers.get("location") ?? "");
  assert.equal(location.searchParams.get("error"), "invalid_request");
  assert.equal(location.searchParams.get("state"), "state-abc");
});

test("異常系: token で redirect_uri 不一致は 400 invalid_grant で拒否する", async () => {
  const app = createApp();
  const location = await issueCode(app);
  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: location.searchParams.get("code") ?? "",
    redirect_uri: "http://localhost:3000/other",
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  });
  assert.equal(res.status, 400);
});

test("正常系: userinfo は Bearer access token で whitelist フィールドのみ返す", async () => {
  const app = createApp();
  const { access_token } = await issueTokens(app);
  const res = await app.request("/oidc/userinfo", {
    headers: { authorization: `Bearer ${access_token}` },
  });
  assert.equal(res.status, 200);
  const info = (await res.json()) as Record<string, unknown>;
  assert.equal(info.sub, "user-john-doe");
  assert.equal(info.email, "john.doe@example.com");
  assert.equal(info.email_verified, true);
  assert.equal(info.name, "John Doe");
  assert.deepEqual(
    Object.keys(info).sort(),
    ["email", "email_verified", "family_name", "given_name", "name", "sub"],
  );
});

test("異常系: userinfo は ID Token を 401 で拒否する", async () => {
  const app = createApp();
  const { id_token } = await issueTokens(app);
  const res = await app.request("/oidc/userinfo", {
    headers: { authorization: `Bearer ${id_token}` },
  });
  assert.equal(res.status, 401);
});

test("異常系: userinfo は Bearer 欠如を 401 で拒否する", async () => {
  const app = createApp();
  const res = await app.request("/oidc/userinfo");
  assert.equal(res.status, 401);
});

test("正常系: logout は登録済み post_logout_redirect_uri へ 302（state 反映）", async () => {
  const app = createApp();
  const res = await logoutRequest(app, {
    post_logout_redirect_uri: "http://localhost:3000",
    state: "logout-state",
  });
  assert.equal(res.status, 302);
  const location = new URL(res.headers.get("location") ?? "");
  assert.equal(location.origin, "http://localhost:3000");
  assert.equal(location.searchParams.get("state"), "logout-state");
});

test("異常系: logout は未登録 post_logout_redirect_uri を 400 で拒否する", async () => {
  const app = createApp();
  const res = await logoutRequest(app, { post_logout_redirect_uri: "http://evil.example.com" });
  assert.equal(res.status, 400);
});

test("正常系: logout は post_logout_redirect_uri 未指定で 200 を返す", async () => {
  const app = createApp();
  const res = await logoutRequest(app, {});
  assert.equal(res.status, 200);
  assert.equal(((await res.json()) as Record<string, string>).status, "logged_out");
});

test("正常系: bypass/session が session を作成し、logout(id_token_hint) が破棄する", async () => {
  sessionStore.clear();
  const app = createApp();
  const created = await app.request("/bypass/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ subject: "user-john-doe" }),
  });
  assert.equal(created.status, 200);
  assert.equal(sessionStore.size, 1);

  const { id_token } = await issueTokens(app);
  await logoutRequest(app, { id_token_hint: id_token });
  assert.equal(sessionStore.size, 0);
});
