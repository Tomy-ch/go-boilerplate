// oidc-flow.integration.test.ts は実 HTTP でサーバを起動し（@hono/node-server の serve アダプタ経由）、
// 実際のエンドポイントを叩いて疎通する統合動作確認。src/**.test.ts の in-process（app.fetch 直叩き）テストが
// 通らない serve アダプタ層を検証する。
import { after, before, describe, test } from "node:test";
import assert from "node:assert/strict";
import { randomBytes } from "node:crypto";
import { serve } from "@hono/node-server";
import type { ServerType } from "@hono/node-server";
import { createApp } from "../src/router.ts";
import { s256Challenge } from "../src/pkce.ts";

const CLIENT_ID = "go-boilerplate-client";
const REDIRECT_URI = "http://localhost:3000/api/auth/callback";

let server: ServerType;
let base: string;

before(async () => {
  const app = createApp();
  const port = await new Promise<number>((resolve) => {
    server = serve({ fetch: app.fetch, port: 0 }, (info) => resolve(info.port));
  });
  base = `http://localhost:${port}`;
});

after(() => {
  server.close();
});

// runAuthorizationCodeFlow は authorize→token を実 HTTP で完走し、token レスポンスを返す。
async function runAuthorizationCodeFlow(): Promise<{ access_token: string; id_token: string; token_type: string }> {
  const verifier = randomBytes(32).toString("base64url");
  const authorizeUrl = new URL(`${base}/oidc/authorize`);
  authorizeUrl.searchParams.set("client_id", CLIENT_ID);
  authorizeUrl.searchParams.set("redirect_uri", REDIRECT_URI);
  authorizeUrl.searchParams.set("response_type", "code");
  authorizeUrl.searchParams.set("scope", "openid profile email");
  authorizeUrl.searchParams.set("code_challenge", s256Challenge(verifier));
  authorizeUrl.searchParams.set("code_challenge_method", "S256");
  authorizeUrl.searchParams.set("state", "state-xyz");
  authorizeUrl.searchParams.set("nonce", "nonce-abc");

  const authorizeRes = await fetch(authorizeUrl, { redirect: "manual" });
  assert.equal(authorizeRes.status, 302, "authorize は 302 でリダイレクトする");
  const location = authorizeRes.headers.get("location");
  assert.ok(location !== null, "authorize は Location を返す");
  const redirected = new URL(location);
  assert.equal(redirected.searchParams.get("state"), "state-xyz", "state を反映する");
  const code = redirected.searchParams.get("code");
  assert.ok(code !== null, "authorize は code を発行する");

  const tokenRes = await fetch(`${base}/oidc/token`, {
    method: "POST",
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: REDIRECT_URI,
      client_id: CLIENT_ID,
      code_verifier: verifier,
    }),
  });
  assert.equal(tokenRes.status, 200, "token は 200 を返す");
  return tokenRes.json() as Promise<{ access_token: string; id_token: string; token_type: string }>;
}

describe("mock OIDC provider 統合（実 HTTP / @hono/node-server serve）", () => {
  describe("正常系", () => {
    test("health が 200 と status:ok を返す", async () => {
      const res = await fetch(`${base}/health`);
      assert.equal(res.status, 200);
      assert.deepEqual(await res.json(), { status: "ok" });
    });

    test("discovery が issuer と /oidc/* エンドポイントを返す", async () => {
      const res = await fetch(`${base}/.well-known/openid-configuration`);
      assert.equal(res.status, 200);
      const doc = (await res.json()) as Record<string, unknown>;
      assert.equal(doc.issuer, "http://localhost:4000");
      assert.equal(doc.authorization_endpoint, "http://localhost:4000/oidc/authorize");
      assert.equal(doc.token_endpoint, "http://localhost:4000/oidc/token");
      assert.equal(doc.userinfo_endpoint, "http://localhost:4000/oidc/userinfo");
    });

    test("jwks が RS256 / kid=mock-key-1 の署名鍵を返す", async () => {
      const res = await fetch(`${base}/.well-known/jwks.json`);
      assert.equal(res.status, 200);
      const jwks = (await res.json()) as { keys: Array<Record<string, unknown>> };
      assert.equal(jwks.keys.length, 1);
      assert.equal(jwks.keys[0].kid, "mock-key-1");
      assert.equal(jwks.keys[0].alg, "RS256");
      assert.equal(jwks.keys[0].use, "sig");
    });

    test("Authorization Code + PKCE フロー完走 → userinfo が user-john-doe fixture を返す", async () => {
      const token = await runAuthorizationCodeFlow();
      assert.equal(token.token_type, "Bearer");
      assert.ok(token.access_token.length > 0);
      assert.ok(token.id_token.length > 0);

      const res = await fetch(`${base}/oidc/userinfo`, {
        headers: { authorization: `Bearer ${token.access_token}` },
      });
      assert.equal(res.status, 200);
      assert.deepEqual(await res.json(), {
        sub: "user-john-doe",
        email: "john.doe@example.com",
        email_verified: true,
        given_name: "John",
        family_name: "Doe",
        name: "John Doe",
      });
    });

    test("admin/reset 後も admin/users に user-john-doe が残る（fixture は reset 対象外）", async () => {
      const reset = await fetch(`${base}/admin/reset`, { method: "POST" });
      assert.equal(reset.status, 200);
      assert.deepEqual(await reset.json(), { status: "reset" });

      const res = await fetch(`${base}/admin/users`);
      assert.equal(res.status, 200);
      const body = (await res.json()) as { users: Array<{ subject: string }> };
      const john = body.users.find((u) => u.subject === "user-john-doe");
      assert.ok(john !== undefined, "reset 後も user-john-doe が存在する");
      assert.deepEqual(john, {
        subject: "user-john-doe",
        email: "john.doe@example.com",
        given_name: "John",
        family_name: "Doe",
        name: "John Doe",
        status: "active",
      });
    });
  });

  describe("異常系", () => {
    test("userinfo は Bearer 無しを 401 で拒否する", async () => {
      const res = await fetch(`${base}/oidc/userinfo`);
      assert.equal(res.status, 401);
      const body = (await res.json()) as { error: string };
      assert.equal(body.error, "invalid_token");
    });

    test("token は使用済み code の再利用を invalid_grant で拒否する", async () => {
      const verifier = randomBytes(32).toString("base64url");
      const authorizeUrl = new URL(`${base}/oidc/authorize`);
      authorizeUrl.searchParams.set("client_id", CLIENT_ID);
      authorizeUrl.searchParams.set("redirect_uri", REDIRECT_URI);
      authorizeUrl.searchParams.set("response_type", "code");
      authorizeUrl.searchParams.set("scope", "openid");
      authorizeUrl.searchParams.set("code_challenge", s256Challenge(verifier));
      authorizeUrl.searchParams.set("code_challenge_method", "S256");
      const authorizeRes = await fetch(authorizeUrl, { redirect: "manual" });
      const code = new URL(authorizeRes.headers.get("location") ?? "").searchParams.get("code") ?? "";

      const form = () =>
        new URLSearchParams({
          grant_type: "authorization_code",
          code,
          redirect_uri: REDIRECT_URI,
          client_id: CLIENT_ID,
          code_verifier: verifier,
        });
      const first = await fetch(`${base}/oidc/token`, { method: "POST", body: form() });
      assert.equal(first.status, 200, "初回の交換は成功する");
      const second = await fetch(`${base}/oidc/token`, { method: "POST", body: form() });
      assert.equal(second.status, 400, "同一 code の再利用は拒否される");
      assert.equal(((await second.json()) as { error: string }).error, "invalid_grant");
    });
  });
});
