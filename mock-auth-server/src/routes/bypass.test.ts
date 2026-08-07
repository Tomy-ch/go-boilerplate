// bypass.test.ts は /bypass/* のリクエストボディ解釈（壊れた JSON・空ボディ・省略時の既定）を検証する。
// dev-gate と正常系の Token 発行は router.test.ts が受け持つ。
import { describe, expect, it } from "vitest";
import { createApp } from "../router.ts";
import { defaultSubject } from "../users.ts";
import { sessionStore } from "../store.ts";

// post は指定パスへ生のボディ文字列を POST し、[status, body] を返す。
async function post(path: string, body: string): Promise<[number, Record<string, unknown>]> {
  const res = await createApp().request(path, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body,
  });
  return [res.status, (await res.json()) as Record<string, unknown>];
}

describe("bypassRoutes", () => {
  describe("正常系", () => {
    it("POST /bypass/token は空ボディなら既定の subject / profile で発行する", async () => {
      const [status, body] = await post("/bypass/token", "");
      expect(status).toBe(200);
      expect(body.token_type).toBe("Bearer");
      expect(typeof body.access_token).toBe("string");
    });

    it("POST /bypass/session は subject 省略時に既定 subject の session を作る", async () => {
      const [status, body] = await post("/bypass/session", "");
      expect(status).toBe(200);
      expect(body.subject).toBe(defaultSubject);
      expect(sessionStore.get(body.session_id as string)?.subject).toBe(defaultSubject);
    });
  });

  describe("異常系", () => {
    it("POST /bypass/token は JSON として壊れたボディを 400 で拒否する", async () => {
      const [status, body] = await post("/bypass/token", "{ not json");
      expect(status).toBe(400);
      expect(body.error).toBe("invalid_request");
    });

    it("POST /bypass/session は JSON として壊れたボディを 400 で拒否する", async () => {
      const [status, body] = await post("/bypass/session", "{ not json");
      expect(status).toBe(400);
      expect(body.error).toBe("invalid_request");
    });
  });
});
