// router.test.ts は Hono アプリを app.request（プロセス内）で叩き、既存エンドポイントの挙動と dev-gate を検証する。
import { describe, expect, it } from "vitest";
import { createApp } from "./router.ts";
import { loadConfig } from "./config.ts";

it("GET /health は 200 と {status:ok} を返す", async () => {
  const app = createApp();
  const res = await app.request("/health");
  expect(res.status).toBe(200);
  expect(await res.json()).toEqual({ status: "ok" });
});

it("GET /.well-known/openid-configuration は issuer/jwks_uri を不変で返す", async () => {
  const app = createApp();
  const res = await app.request("/.well-known/openid-configuration");
  expect(res.status).toBe(200);
  const doc = (await res.json()) as Record<string, unknown>;
  const cfg = loadConfig();
  expect(doc.issuer).toBe(cfg.issuer);
  expect(doc.jwks_uri).toBe(`${cfg.issuer}/.well-known/jwks.json`);
  expect(doc.authorization_endpoint).toBe(`${cfg.issuer}/oidc/authorize`);
  expect(doc.token_endpoint_auth_methods_supported).toEqual(["none"]);
});

it("GET /.well-known/jwks.json は公開鍵（kid=mock-key-1 / RS256）を返す", async () => {
  const app = createApp();
  const res = await app.request("/.well-known/jwks.json");
  expect(res.status).toBe(200);
  const jwks = (await res.json()) as { keys: Array<Record<string, unknown>> };
  expect(jwks.keys[0]?.kid).toBe("mock-key-1");
  expect(jwks.keys[0]?.alg).toBe("RS256");
});

it("POST /bypass/token は valid Profile で access token を発行する", async () => {
  const app = createApp();
  const res = await app.request("/bypass/token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ subject: "user-john-doe", profile: "valid" }),
  });
  expect(res.status).toBe(200);
  const body = (await res.json()) as Record<string, unknown>;
  expect(body.token_type).toBe("Bearer");
  expect(typeof body.access_token).toBe("string");
});

it("POST /bypass/token は未知の profile を 400 で拒否する", async () => {
  const app = createApp();
  const res = await app.request("/bypass/token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ profile: "no-such-profile" }),
  });
  expect(res.status).toBe(400);
});

it("GET /admin/users は固定 User 一覧を返す", async () => {
  const app = createApp();
  const res = await app.request("/admin/users");
  expect(res.status).toBe(200);
  const body = (await res.json()) as { users: Array<{ subject: string }> };
  expect(body.users.length > 0).toBeTruthy();
  expect(typeof body.users[0].subject).toBe("string");
});

it("dev-gate 無効時は /bypass/token ・ /admin/users を 404 で秘匿する", async () => {
  const app = createApp({ devEndpointsEnabled: false });
  const bypass = await app.request("/bypass/token", { method: "POST" });
  expect(bypass.status).toBe(404);
  const admin = await app.request("/admin/users");
  expect(admin.status).toBe(404);
});

it("未定義パスは 404 を返す", async () => {
  const app = createApp();
  const res = await app.request("/no/such/path");
  expect(res.status).toBe(404);
});
