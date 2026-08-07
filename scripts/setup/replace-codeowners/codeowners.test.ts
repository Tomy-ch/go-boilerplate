import { describe, expect, it } from "vitest";

import { parseOwners, replaceCodeowners } from "./codeowners";

describe("parseOwners", () => {
  describe("正常系", () => {
    it("空白区切りを所有者の配列へ分解する", () => {
      expect(parseOwners("@example-org/tech-leads @example-org/security")).toEqual([
        "@example-org/tech-leads",
        "@example-org/security",
      ]);
    });

    it("前後の空白と連続空白を落とす", () => {
      expect(parseOwners("  @a \t  @b  ")).toEqual(["@a", "@b"]);
    });
  });

  describe("異常系", () => {
    it("空文字を空配列にする（検証側で弾けるようにする）", () => {
      expect(parseOwners("   ")).toEqual([]);
    });
  });
});

describe("replaceCodeowners", () => {
  describe("正常系", () => {
    it("ルール行の所有者だけを差し替え、パターンと区切り空白を保つ", () => {
      const content = "go.mod                                @Tomy-ch\n";

      expect(replaceCodeowners(content, "@org/team").content).toBe(
        "go.mod                                @org/team\n",
      );
    });

    it("複数の所有者を持つ行も 1 つの所有者欄として置き換える", () => {
      const result = replaceCodeowners("*.go @a @b @c\n", "@org/team");

      expect(result.content).toBe("*.go @org/team\n");
      expect(result.replaced).toBe(1);
    });

    it("行末コメントを残す", () => {
      expect(replaceCodeowners("go.mod @a  # 供給網\n", "@org/team").content).toBe(
        "go.mod @org/team  # 供給網\n",
      );
    });

    it("メールアドレス形式の所有者欄も置き換える", () => {
      expect(replaceCodeowners("go.mod owner@example.com\n", "@org/team").content).toBe(
        "go.mod @org/team\n",
      );
    });

    it("すべてのルール行を置き換えて件数を返す", () => {
      const result = replaceCodeowners("# 見出し\n\ngo.mod @a\ngo.sum @b\n", "@org/team");

      expect(result.replaced).toBe(2);
      expect(result.skippedLines).toEqual([]);
    });

    // 書き換えた行だけ LF になると、CRLF のリポジトリで改行が混在した差分になる。
    it("CRLF の行は CR を保って書き換える", () => {
      expect(replaceCodeowners("go.mod @a\r\ngo.sum @b\r\n", "@org/team").content).toBe(
        "go.mod @org/team\r\ngo.sum @org/team\r\n",
      );
    });
  });

  describe("異常系", () => {
    // ヘッダーは所有者の記載例を含む。コメント行を書き換えると、使い方の説明が
    // 利用者自身の所有者に化けて例として読めなくなる。
    it("コメント行を書き換えない", () => {
      const content = "#   make setup-replace-codeowners OWNERS='@example-org/tech-leads'\ngo.mod @a\n";

      expect(replaceCodeowners(content, "@org/team").content).toBe(
        "#   make setup-replace-codeowners OWNERS='@example-org/tech-leads'\ngo.mod @org/team\n",
      );
    });

    // 所有者を持たないパターン行は継承の打ち消し。所有者を足すと、打ち消しのつもりの
    // 行が所有者を要求する行に反転する。
    it("所有者を持たないパターン行を書き換えない", () => {
      const content = "/vendor/\ngo.mod @a\n";

      const result = replaceCodeowners(content, "@org/team");

      expect(result.content).toBe("/vendor/\ngo.mod @org/team\n");
      expect(result.skippedLines).toEqual([]);
    });

    it("空行を書き換えない", () => {
      expect(replaceCodeowners("\n\ngo.mod @a\n", "@org/team").content).toBe(
        "\n\ngo.mod @org/team\n",
      );
    });

    // 空白だけを境界にすると、エスケープした空白を含むパターンの途中で切って
    // ファイル名の後半を所有者欄と誤認し、パターンそのものを壊す。
    it("所有者の形をしないトークンを所有者欄と認めず、行番号で報告する", () => {
      const result = replaceCodeowners("foo\\ bar.txt @a\ngo.mod @b\n", "@org/team");

      expect(result.content).toBe("foo\\ bar.txt @a\ngo.mod @org/team\n");
      expect(result.skippedLines).toEqual([1]);
      expect(result.replaced).toBe(1);
    });

    it("セクション見出しのような複数語の行を未置換として報告する", () => {
      const result = replaceCodeowners("[My Team]\ngo.mod @b\n", "@org/team");

      expect(result.skippedLines).toEqual([1]);
    });

    // 1 行も置き換えられていないのに成功で終えると、レビューを予約したつもりで
    // 誰にも予約できていない CODEOWNERS がそのまま残る。
    it("所有者を持つルール行が 1 行も無ければ throw する", () => {
      expect(() => replaceCodeowners("# 見出しだけ\n\n/vendor/\n", "@org/team")).toThrow(
        ".github/CODEOWNERS",
      );
    });
  });
});
