// oidc.test.ts は Authorization Code Flow + PKCE を app.request（プロセス内）で e2e 検証する。
import { test } from "node:test";
import assert from "node:assert/strict";
import { createApp } from "../router.ts";
import { s256Challenge } from "../pkce.ts";

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
