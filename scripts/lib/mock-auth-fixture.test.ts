import { describe, expect, it } from "vitest";

import { DEFAULT_USERS, renderFixture } from "./mock-auth-fixture";

describe("DEFAULT_USERS", () => {
  describe("正常系", () => {
    it("mock を起動できる最小の 1 件を持つ", () => {
      expect(DEFAULT_USERS).toHaveLength(1);
    });

    it("全ユーザーが mock-auth-server の必須項目を埋めている", () => {
      for (const user of DEFAULT_USERS) {
        expect(Object.keys(user).sort()).toEqual(
          ["email", "family_name", "given_name", "name", "status", "subject"],
        );
      }
    });

    it("認証できる状態で書き出す", () => {
      expect(DEFAULT_USERS.every((user) => user.status === "active")).toBe(true);
    });
  });

  describe("異常系", () => {
    // このスクリプトは `make setup-remove-sample-api` から呼ばれ、デモの固定ユーザーを
    // 消すのが目的である。既定値そのものにデモ由来の値が残っていれば、撤去は成功したのに
    // サンプルが残る。
    it("デモ由来の人名・ドメインを含まない", () => {
      const serialized = JSON.stringify(DEFAULT_USERS).toLowerCase();

      for (const demo of ["john", "doe", "jane", "example.org", "boilerplate"]) {
        expect(serialized).not.toContain(demo);
      }
    });
  });
});

describe("renderFixture", () => {
  describe("正常系", () => {
    it("2 スペース字下げの JSON を末尾改行付きで返す", () => {
      const rendered = renderFixture([]);

      expect(rendered).toBe("[]\n");
    });

    it("書き出した内容を読み戻せる", () => {
      expect(JSON.parse(renderFixture(DEFAULT_USERS))).toEqual(DEFAULT_USERS);
    });
  });
});
