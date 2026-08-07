// clients.test.ts は登録済み OAuth クライアントの読み込みと参照を検証する。fixture が壊れていても
// mock が起動できること（空配列へのフォールバック）が主眼で、fixture の中身自体は検証対象にしない。
import { describe, expect, it } from "vitest";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { clients, findClient, loadClients } from "./clients.ts";

// fixtureFile は指定内容を書いた一時ファイルのパスを返す。
function fixtureFile(content: string): string {
  const path = join(mkdtempSync(join(tmpdir(), "mock-auth-clients-")), "clients.json");
  writeFileSync(path, content, "utf8");
  return path;
}

describe("loadClients", () => {
  describe("正常系", () => {
    it("JSON として読めるファイルの内容をそのまま返す", () => {
      const entries = [{ client_id: "some-client", scopes: ["openid"] }];
      expect(loadClients(fixtureFile(JSON.stringify(entries)))).toEqual(entries);
    });
  });

  describe("異常系", () => {
    it("存在しないパスは空配列を返す", () => {
      expect(loadClients(join(tmpdir(), "no-such-dir", "clients.json"))).toEqual([]);
    });

    it("JSON として壊れたファイルは空配列を返す", () => {
      expect(loadClients(fixtureFile("{ not json"))).toEqual([]);
    });
  });
});

describe("findClient", () => {
  describe("正常系", () => {
    it("登録済み client_id のクライアントを返す", () => {
      const clientId = clients[0].client_id;
      expect(findClient(clientId)?.client_id).toBe(clientId);
    });
  });

  describe("異常系", () => {
    it("未登録の client_id は undefined を返す", () => {
      expect(findClient("not-registered-client")).toBe(undefined);
    });
  });
});
