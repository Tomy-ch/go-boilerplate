// router.test.ts は Hono アプリを app.request（プロセス内）で叩き、既存エンドポイントの挙動と dev-gate を検証する。
import { test } from "node:test";
import assert from "node:assert/strict";
import { createApp } from "./router.ts";

test("GET /health は 200 と {status:ok} を返す", async () => {
  const app = createApp();
  const res = await app.request("/health");
  assert.equal(res.status, 200);
  assert.deepEqual(await res.json(), { status: "ok" });
});

test("GET /.well-known/openid-configuration は issuer/jwks_uri を不変で返す", async () => {
  const app = createApp();
  const res = await app.request("/.well-known/openid-configuration");
  assert.equal(res.status, 200);
  const doc = (await res.json()) as Record<string, unknown>;
  assert.equal(doc.issuer, "http://localhost:4000");
  assert.equal(doc.jwks_uri, "http://localhost:4000/.well-known/jwks.json");
  assert.equal(doc.authorization_endpoint, "http://localhost:4000/oidc/authorize");
  assert.deepEqual(doc.token_endpoint_auth_methods_supported, ["none"]);
});

test("GET /.well-known/jwks.json は公開鍵（kid=mock-key-1 / RS256）を返す", async () => {
  const app = createApp();
  const res = await app.request("/.well-known/jwks.json");
  assert.equal(res.status, 200);
  const jwks = (await res.json()) as { keys: Array<Record<string, unknown>> };
  assert.equal(jwks.keys[0]?.kid, "mock-key-1");
  assert.equal(jwks.keys[0]?.alg, "RS256");
});

test("POST /bypass/token は valid Profile で access token を発行する", async () => {
  const app = createApp();
  const res = await app.request("/bypass/token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ subject: "user-john-doe", profile: "valid" }),
  });
  assert.equal(res.status, 200);
  const body = (await res.json()) as Record<string, unknown>;
  assert.equal(body.token_type, "Bearer");
  assert.equal(typeof body.access_token, "string");
});

test("POST /bypass/token は未知の profile を 400 で拒否する", async () => {
  const app = createApp();
  const res = await app.request("/bypass/token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ profile: "no-such-profile" }),
  });
  assert.equal(res.status, 400);
});

test("GET /admin/users は固定 User 一覧を返す", async () => {
  const app = createApp();
  const res = await app.request("/admin/users");
  assert.equal(res.status, 200);
  const body = (await res.json()) as { users: unknown[] };
  assert.ok(Array.isArray(body.users));
});

test("dev-gate 無効時は /bypass/token ・ /admin/users を 404 で秘匿する", async () => {
  const app = createApp({ devEndpointsEnabled: false });
  const bypass = await app.request("/bypass/token", { method: "POST" });
  assert.equal(bypass.status, 404);
  const admin = await app.request("/admin/users");
  assert.equal(admin.status, 404);
});

test("未定義パスは 404 を返す", async () => {
  const app = createApp();
  const res = await app.request("/no/such/path");
  assert.equal(res.status, 404);
});
