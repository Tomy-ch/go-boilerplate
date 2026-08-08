// admin.test.ts は /admin/reset（揮発ストア + 鍵ストア初期化）、/admin/keys/rotate（宣言的操作）、dev-gate を検証する。
import { describe, expect, it, beforeEach } from "vitest";
import { createApp } from "../router.ts";
import { codeStore, sessionStore } from "../store.ts";
import { keyStore, PRIMARY_KID, ROTATION_KID, RETIRED_KID, RotationError } from "../keys.ts";
import { rotationFailure } from "./admin.ts";

// 鍵ストアはモジュール singleton のため、各テスト前に Phase1 へ戻して独立性を担保する。
beforeEach(() => {
  keyStore.reset();
});

// rotate は /admin/keys/rotate を叩き、[status, body] を返す。
async function rotate(app: ReturnType<typeof createApp>, action: string, kid: string): Promise<[number, Record<string, unknown>]> {
  const res = await app.request("/admin/keys/rotate", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ action, kid }),
  });
  return [res.status, (await res.json()) as Record<string, unknown>];
}

describe("adminRoutes", () => {
  describe("正常系", () => {
    it("admin/reset は揮発ストア（code / session）を初期化する", async () => {
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
      expect(res.status).toBe(200);
      expect(((await res.json()) as Record<string, string>).status).toBe("reset");
      expect(codeStore.size).toBe(0);
      expect(sessionStore.size).toBe(0);
    });

    it("admin/reset は鍵ストアを Phase1（公開 = 署名 = PRIMARY_KID のみ）へ戻す", async () => {
      const app = createApp();
      keyStore.addKey(ROTATION_KID);
      keyStore.promote(ROTATION_KID);

      const res = await app.request("/admin/reset", { method: "POST" });
      expect(res.status).toBe(200);
      expect(keyStore.state()).toEqual({ signing_kid: PRIMARY_KID, published_kids: [PRIMARY_KID] });
    });

    it("rotate の add-key → promote → retire で Phase1→2→3 を再現する", async () => {
      const app = createApp();

      // Phase1（初期）
      expect(keyStore.state()).toEqual({ signing_kid: PRIMARY_KID, published_kids: [PRIMARY_KID] });

      // Phase2: 新鍵追加（署名鍵は不変）→ 署名鍵昇格
      const [addStatus, addBody] = await rotate(app, "add-key", ROTATION_KID);
      expect(addStatus).toBe(200);
      expect(addBody).toEqual({ signing_kid: PRIMARY_KID, published_kids: [PRIMARY_KID, ROTATION_KID] });
      const [promoteStatus, promoteBody] = await rotate(app, "promote", ROTATION_KID);
      expect(promoteStatus).toBe(200);
      expect(promoteBody).toEqual({ signing_kid: ROTATION_KID, published_kids: [PRIMARY_KID, ROTATION_KID] });

      // Phase3: 旧鍵退役
      const [retireStatus, retireBody] = await rotate(app, "retire", PRIMARY_KID);
      expect(retireStatus).toBe(200);
      expect(retireBody).toEqual({ signing_kid: ROTATION_KID, published_kids: [ROTATION_KID] });
    });

    it("JWKS は公開集合の変化を反映する", async () => {
      const app = createApp();
      await rotate(app, "add-key", ROTATION_KID);

      const res = await app.request("/.well-known/jwks.json");
      const jwks = (await res.json()) as { keys: Array<{ kid: string }> };
      expect(jwks.keys.map((k) => k.kid)).toEqual([PRIMARY_KID, ROTATION_KID]);
    });

    it("admin/keys/rotate は 501 スタブではなく実装済みで 200 を返す", async () => {
      const app = createApp();
      const [status] = await rotate(app, "add-key", ROTATION_KID);
      expect(status).toBe(200);
    });
  });

  describe("異常系", () => {
    it("未知 kid の add-key は 400", async () => {
      const app = createApp();
      const [status, body] = await rotate(app, "add-key", "no-such-key");
      expect(status).toBe(400);
      expect(body.error).toBe("invalid_request");
    });

    it("退役鍵 kid の add-key は 400（回転プール外）", async () => {
      const app = createApp();
      const [status] = await rotate(app, "add-key", RETIRED_KID);
      expect(status).toBe(400);
    });

    it("既に公開済み kid の add-key は 400（二重公開）", async () => {
      const app = createApp();
      const [status] = await rotate(app, "add-key", PRIMARY_KID);
      expect(status).toBe(400);
    });

    it("未公開 kid の promote は 400", async () => {
      const app = createApp();
      const [status] = await rotate(app, "promote", ROTATION_KID);
      expect(status).toBe(400);
    });

    it("未公開 kid の retire は 400", async () => {
      const app = createApp();
      const [status] = await rotate(app, "retire", ROTATION_KID);
      expect(status).toBe(400);
    });

    it("現署名鍵の retire は 400（署名不能を防ぐ）", async () => {
      const app = createApp();
      const [status] = await rotate(app, "retire", PRIMARY_KID);
      expect(status).toBe(400);
    });

    it("不正な action / 欠落フィールドは 400", async () => {
      const app = createApp();
      const [badAction] = await rotate(app, "flip", PRIMARY_KID);
      expect(badAction).toBe(400);

      const res = await app.request("/admin/keys/rotate", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ action: "add-key" }),
      });
      expect(res.status).toBe(400);
    });

    it("dev-gate 無効時は admin/reset を 404 で秘匿する", async () => {
      const app = createApp({ devEndpointsEnabled: false });
      const res = await app.request("/admin/reset", { method: "POST" });
      expect(res.status).toBe(404);
    });

    it("dev-gate 無効時は admin/keys/rotate を 404 で秘匿する", async () => {
      const app = createApp({ devEndpointsEnabled: false });
      const res = await app.request("/admin/keys/rotate", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ action: "add-key", kid: ROTATION_KID }),
      });
      expect(res.status).toBe(404);
    });

    it("admin/keys/rotate は JSON として壊れたボディを 400 で拒否する", async () => {
      const res = await createApp().request("/admin/keys/rotate", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: "{ not json",
      });
      expect(res.status).toBe(400);
      expect(((await res.json()) as Record<string, unknown>).error).toBe("invalid_request");
    });

    it("空ボディの rotate は action を欠くものとして拒否する", async () => {
      const app = createApp();

      const res = await app.request("/admin/keys/rotate", { method: "POST" });

      expect(res.status).toBe(400);
    });
  });
});

describe("rotationFailure", () => {
  describe("正常系", () => {
    it("RotationError は 400 応答ボディへ写像する", () => {
      expect(rotationFailure(new RotationError("kid is not rotatable: x"))).toEqual({
        error: "invalid_request",
        error_description: "kid is not rotatable: x",
      });
    });
  });

  describe("異常系", () => {
    it("RotationError 以外は握り潰さず投げ返す", () => {
      const unexpected = new Error("boom");
      expect(() => rotationFailure(unexpected)).toThrow(unexpected);
    });
  });
});
