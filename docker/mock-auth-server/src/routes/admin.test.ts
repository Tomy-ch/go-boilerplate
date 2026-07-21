// admin.test.ts は /admin/reset の揮発ストア初期化、keys/rotate の 501、dev-gate を検証する。
import { test } from "node:test";
import assert from "node:assert/strict";
import { createApp } from "../router.ts";
import { codeStore, sessionStore } from "../store.ts";

test("正常系: admin/reset は揮発ストア（code / session）を初期化する", async () => {
  const app = createApp();
  codeStore.set("code-1", {
    clientId: "go-boilerplate-client",
    redirectUri: "http://localhost:3000/api/auth/callback",
    subject: "user-john-doe",
    scope: "openid",
    codeChallenge: "challenge",
  });
  sessionStore.set("session-1", { subject: "user-john-doe" });

  const res = await app.request("/admin/reset", { method: "POST" });
  assert.equal(res.status, 200);
  assert.equal(((await res.json()) as Record<string, string>).status, "reset");
  assert.equal(codeStore.size, 0);
  assert.equal(sessionStore.size, 0);
});

test("契約: admin/keys/rotate は未実装スタブとして 501 を返す", async () => {
  const app = createApp();
  const res = await app.request("/admin/keys/rotate", { method: "POST" });
  assert.equal(res.status, 501);
});

test("異常系: dev-gate 無効時は admin/reset を 404 で秘匿する", async () => {
  const app = createApp({ devEndpointsEnabled: false });
  const res = await app.request("/admin/reset", { method: "POST" });
  assert.equal(res.status, 404);
});
