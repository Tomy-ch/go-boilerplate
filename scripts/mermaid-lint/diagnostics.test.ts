import { describe, expect, it } from "vitest";

import { errorMessage, formatFailures, isDependencyMissing, summarize } from "./diagnostics";

describe("errorMessage", () => {
  describe("正常系", () => {
    it("Error のメッセージを返す", () => {
      expect(errorMessage(new Error("boom"))).toBe("boom");
    });

    it("前後の空白を落とす", () => {
      expect(errorMessage(new Error("  boom  "))).toBe("boom");
    });

    it("Error 以外の値は文字列化して返す", () => {
      expect(errorMessage("plain string")).toBe("plain string");
    });
  });

  describe("異常系", () => {
    it("メッセージが空の Error は文字列化側へ落とす", () => {
      expect(errorMessage(new Error(""))).toBe("Error");
    });

    it("undefined でも空行にせず値を残す", () => {
      expect(errorMessage(undefined)).toBe("undefined");
    });
  });
});

describe("isDependencyMissing", () => {
  describe("正常系", () => {
    it("ERR_MODULE_NOT_FOUND を依存の欠落として扱う", () => {
      const err = Object.assign(new Error("whatever"), { code: "ERR_MODULE_NOT_FOUND" });

      expect(isDependencyMissing(err)).toBe(true);
    });

    it("code を持たない Cannot find package も依存の欠落として扱う", () => {
      expect(isDependencyMissing(new Error("Cannot find package 'mermaid'"))).toBe(true);
    });

    it("Cannot find module も大小文字を問わず拾う", () => {
      expect(isDependencyMissing(new Error("cannot find MODULE 'linkedom'"))).toBe(true);
    });
  });

  describe("異常系", () => {
    it("無関係な例外は依存の欠落として扱わない", () => {
      expect(isDependencyMissing(new Error("Parse error on line 3"))).toBe(false);
    });

    it("別の code を持つ例外は依存の欠落として扱わない", () => {
      const err = Object.assign(new Error("nope"), { code: "EACCES" });

      expect(isDependencyMissing(err)).toBe(false);
    });
  });
});

describe("formatFailures", () => {
  describe("正常系", () => {
    it("ファイル・行・ブロック番号を見出しにして詳細をぶら下げる", () => {
      const text = formatFailures([{ rel: "a.md", startLine: 12, index: 1, msg: "bad" }]);

      expect(text).toBe("  a.md:12  (block #1)\n    bad\n");
    });

    it("複数行のメッセージを行ごとに字下げする", () => {
      const text = formatFailures([{ rel: "a.md", startLine: 1, index: 2, msg: "one\ntwo" }]);

      expect(text).toBe("  a.md:1  (block #2)\n    one\n    two\n");
    });

    it("複数の失敗を空行で区切って並べる", () => {
      const text = formatFailures([
        { rel: "a.md", startLine: 1, index: 1, msg: "x" },
        { rel: "b.md", startLine: 2, index: 1, msg: "y" },
      ]);

      expect(text).toBe("  a.md:1  (block #1)\n    x\n\n  b.md:2  (block #1)\n    y\n");
    });
  });

  describe("異常系", () => {
    it("失敗が無ければ空文字を返す", () => {
      expect(formatFailures([])).toBe("");
    });
  });
});

describe("summarize", () => {
  describe("正常系", () => {
    it("検証したブロック数とファイル数を並べる", () => {
      expect(summarize(10, 3, 0)).toBe("10 ブロック / 3 ファイル");
    });

    it("読めず skip したファイルがあれば件数を添える", () => {
      expect(summarize(10, 3, 2)).toBe("10 ブロック / 3 ファイル（読めず skip: 2 件）");
    });
  });

  describe("異常系", () => {
    it("1 ブロックも無ければ 0 件として報告する", () => {
      expect(summarize(0, 0, 0)).toBe("0 ブロック / 0 ファイル");
    });
  });
});
