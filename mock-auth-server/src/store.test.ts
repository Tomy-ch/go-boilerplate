// store.test.ts は TTLStore の TTL・単回使用・sweep を検証する（clock を注入して時間を制御する）。
import { describe, expect, it } from "vitest";
import { TTLStore, codeStore, resetAll, sessionStore, sweepAll } from "./store.ts";

// putLiveEntries は code / session ストアに鮮度内のエントリを 1 件ずつ置く。
function putLiveEntries(): void {
  codeStore.set("code", {
    clientId: "go-boilerplate-client",
    redirectUri: "http://localhost:3000/api/auth/callback",
    subject: "user-john-doe",
    scope: "openid",
    codeChallenge: "challenge",
  });
  sessionStore.set("session", { subject: "user-john-doe" });
}

describe("TTLStore", () => {
  describe("正常系", () => {
    it("take は単回使用で、2 回目は undefined を返す", () => {
      let now = 0;
      const store = new TTLStore<string>(1000, () => now);
      store.set("code", "value");
      expect(store.take("code")).toBe("value");
      expect(store.take("code")).toBe(undefined);
      expect(store.size).toBe(0);
    });

    it("get は値を保持し、繰り返し取得できる", () => {
      let now = 0;
      const store = new TTLStore<string>(1000, () => now);
      store.set("session", "value");
      expect(store.get("session")).toBe("value");
      expect(store.get("session")).toBe("value");
      expect(store.size).toBe(1);
    });

    it("sweep は期限切れのみを回収し、鮮度内は残す", () => {
      let now = 0;
      const store = new TTLStore<string>(1000, () => now);
      store.set("old", "1");
      now = 500;
      store.set("new", "2");
      now = 1000;
      store.sweep();
      expect(store.get("old")).toBe(undefined);
      expect(store.get("new")).toBe("2");
      expect(store.size).toBe(1);
    });

    it("clear は全エントリを破棄する", () => {
      const store = new TTLStore<string>(1000, () => 0);
      store.set("a", "1");
      store.set("b", "2");
      store.clear();
      expect(store.size).toBe(0);
    });

    it("delete は指定 key だけを破棄する", () => {
      const store = new TTLStore<string>(1000, () => 0);
      store.set("a", "1");
      store.set("b", "2");
      store.delete("a");
      expect(store.get("a")).toBe(undefined);
      expect(store.get("b")).toBe("2");
    });

    it("deleteWhere は述語に一致しないエントリを残す", () => {
      const store = new TTLStore<{ subject: string }>(1000, () => 0);
      store.set("a", { subject: "user-a" });
      store.set("b", { subject: "user-b" });

      store.deleteWhere((value) => value.subject === "user-a");

      expect(store.take("a")).toBe(undefined);
      expect(store.take("b")).toEqual({ subject: "user-b" });
    });
  });

  describe("異常系", () => {
    it("TTL 経過後は get が undefined を返し、エントリを失効させる", () => {
      let now = 0;
      const store = new TTLStore<string>(1000, () => now);
      store.set("code", "value");
      now = 999;
      expect(store.get("code")).toBe("value");
      now = 1000;
      expect(store.get("code")).toBe(undefined);
      expect(store.size).toBe(0);
    });

    it("期限切れ後の take は undefined を返す", () => {
      let now = 0;
      const store = new TTLStore<string>(1000, () => now);
      store.set("code", "value");
      now = 2000;
      expect(store.take("code")).toBe(undefined);
    });

    it("未知の key は undefined を返す", () => {
      const store = new TTLStore<string>(1000, () => 0);
      expect(store.get("missing")).toBe(undefined);
      expect(store.take("missing")).toBe(undefined);
    });

    it("未知の key の delete は何も壊さない", () => {
      const store = new TTLStore<string>(1000, () => 0);
      store.set("a", "1");
      store.delete("missing");
      expect(store.size).toBe(1);
    });
  });
});

describe("sweepAll", () => {
  describe("正常系", () => {
    it("定期回収は鮮度内の code / session を落とさない", () => {
      resetAll();
      putLiveEntries();
      sweepAll();
      expect(codeStore.size).toBe(1);
      expect(sessionStore.size).toBe(1);
      resetAll();
    });
  });
});

describe("resetAll", () => {
  describe("正常系", () => {
    it("code / session 双方を初期化する", () => {
      putLiveEntries();
      resetAll();
      expect(codeStore.size).toBe(0);
      expect(sessionStore.size).toBe(0);
    });
  });
});
