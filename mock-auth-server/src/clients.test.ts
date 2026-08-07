// clients.test.ts は登録済み OAuth クライアントの読み込みと参照を検証する。fixture が壊れていても
// mock が起動できること（空配列へのフォールバック）が主眼で、fixture の中身自体は検証対象にしない。
import { describe, expect, it } from "vitest";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { clients, findClient, loadClients } from "./clients.ts";

// brokenFixture は指定内容を書いた一時ファイルのパスを返す。
function brokenFixture(content: string): string {
  const path = join(mkdtempSync(join(tmpdir(), "mock-auth-clients-")), "clients.json");
  writeFileSync(path, content, "utf8");
  return path;
}

it("loadClients: 存在しないパスは空配列を返す", () => {
  expect(loadClients(join(tmpdir(), "no-such-dir", "clients.json"))).toEqual([]);
});

it("loadClients: JSON として壊れたファイルは空配列を返す", () => {
  expect(loadClients(brokenFixture("{ not json"))).toEqual([]);
});

it("findClient: 登録済み client_id のクライアントを返す", () => {
  const clientId = clients[0].client_id;
  expect(findClient(clientId)?.client_id).toBe(clientId);
});

it("findClient: 未登録の client_id は undefined を返す", () => {
  expect(findClient("not-registered-client")).toBe(undefined);
});
