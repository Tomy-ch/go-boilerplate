import { describe, expect, it } from "vitest";

import { formatFindings } from "./lint-report";

describe("formatFindings", () => {
  describe("正常系", () => {
    it("違反をファイル見出しと行番号付きで並べる", () => {
      const text = formatFindings([{ file: "a.yaml", line: 3, message: "壊れています" }]);

      expect(text).toBe("  a.yaml\n    :3  壊れています");
    });

    it("同じファイルの違反は見出しを繰り返さない", () => {
      const text = formatFindings([
        { file: "a.yaml", line: 3, message: "一つ目" },
        { file: "a.yaml", line: 9, message: "二つ目" },
      ]);

      expect(text).toBe("  a.yaml\n    :3  一つ目\n    :9  二つ目");
    });

    it("ファイルが変わるところへ空行を挟む", () => {
      const text = formatFindings([
        { file: "a.yaml", line: 3, message: "一つ目" },
        { file: "b.yaml", line: 1, message: "二つ目" },
      ]);

      expect(text).toBe("  a.yaml\n    :3  一つ目\n\n  b.yaml\n    :1  二つ目");
    });

    it("渡された順を並べ替えない", () => {
      const text = formatFindings([
        { file: "b.yaml", line: 1, message: "先" },
        { file: "a.yaml", line: 1, message: "後" },
      ]);

      expect(text.indexOf("b.yaml")).toBeLessThan(text.indexOf("a.yaml"));
    });

    it("同じファイルが離れて現れたら見出しを二度出す", () => {
      const text = formatFindings([
        { file: "a.yaml", line: 1, message: "一つ目" },
        { file: "b.yaml", line: 1, message: "二つ目" },
        { file: "a.yaml", line: 2, message: "三つ目" },
      ]);

      expect(text.split("  a.yaml").length - 1).toBe(2);
    });
  });

  describe("異常系", () => {
    it("違反が無ければ空文字を返す", () => {
      expect(formatFindings([])).toBe("");
    });
  });
});
