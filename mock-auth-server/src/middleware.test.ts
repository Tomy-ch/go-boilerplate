// middleware.test.ts は dev endpoint を保護する devGate を、最小の Hono アプリへ載せて検証する。
// 無効時に返すのが 403 ではなく 404 であること（存在そのものを秘匿する）が守りたい契約。
import { Hono } from "hono";
import { describe, expect, it } from "vitest";
import { devGate } from "./middleware.ts";

// appWithGate は devGate 配下に 1 本だけルートを持つアプリを組み立てる。
function appWithGate(enabled: boolean): Hono {
  const app = new Hono();
  app.use("/dev/*", devGate(enabled));
  app.get("/dev/thing", (c) => c.json({ reached: true }));
  return app;
}

describe("devGate", () => {
  describe("正常系", () => {
    it("有効なら後続のハンドラへ通す", async () => {
      const res = await appWithGate(true).request("/dev/thing");

      expect(res.status).toBe(200);
      expect(await res.json()).toEqual({ reached: true });
    });

    it("配下に無いパスは有効・無効いずれでも素通しする", async () => {
      const app = appWithGate(false);
      app.get("/open", (c) => c.json({ reached: true }));

      const res = await app.request("/open");

      expect(res.status).toBe(200);
    });
  });

  describe("異常系", () => {
    it("無効なら後続のハンドラへ渡さず 404 で存在を秘匿する", async () => {
      const res = await appWithGate(false).request("/dev/thing");

      expect(res.status).toBe(404);
      expect(await res.json()).toEqual({ error: "not_found" });
    });
  });
});
