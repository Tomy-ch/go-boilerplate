// users.test.ts は固定 User Fixture の読み込みと参照を検証する。fixture が壊れていても mock が
// 起動できること（空配列へのフォールバック）が主眼で、fixture の中身自体は検証対象にしない。
import { describe, expect, it } from "vitest";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { defaultSubject, findUser, firstSubject, loadUsers, users } from "./users.ts";

// brokenFixture は指定内容を書いた一時ファイルのパスを返す。
function brokenFixture(content: string): string {
  const path = join(mkdtempSync(join(tmpdir(), "mock-auth-users-")), "users.json");
  writeFileSync(path, content, "utf8");
  return path;
}

it("loadUsers: 存在しないパスは空配列を返す", () => {
  expect(loadUsers(join(tmpdir(), "no-such-dir", "users.json"))).toEqual([]);
});

it("loadUsers: JSON として壊れたファイルは空配列を返す", () => {
  expect(loadUsers(brokenFixture("{ not json"))).toEqual([]);
});

it("findUser: 登録済み subject の User を返す", () => {
  const subject = users[0].subject;
  expect(findUser(subject)?.subject).toBe(subject);
});

it("findUser: 未登録の subject は undefined を返す", () => {
  expect(findUser("user-not-registered")).toBe(undefined);
});

it("firstSubject: 先頭の subject を既定として採用する", () => {
  expect(firstSubject(users)).toBe(users[0].subject);
  expect(defaultSubject).toBe(users[0].subject);
});

it("firstSubject: User が 1 人も居なければ中立な既定へフォールバックする", () => {
  expect(firstSubject([])).toBe("user-example");
});
