import { describe, expect, it } from "vitest";

import { renderHelp } from "./help";

function source(content: string, path = ".makefiles/sample.mk") {
  return [{ path, content }];
}

describe("renderHelp", () => {
  describe("正常系", () => {
    it("見出し 2 行から始める", () => {
      const { lines } = renderHelp([]);

      expect(lines).toEqual(["📦 Makeターゲット一覧", "-------------------------------------------"]);
    });

    it("カテゴリ行の前に空行を挟む", () => {
      const { lines } = renderHelp(source("## DB関連"));

      expect(lines.slice(2)).toEqual(["", "📂 DB関連"]);
    });

    it("ターゲット名を左詰めで揃え、説明を続ける", () => {
      const { lines } = renderHelp(source(".PHONY: serve ## 開発サーバを起動する"));

      expect(lines.at(-1)).toBe(`🛠  ${"serve".padEnd(24)} 開発サーバを起動する`);
    });

    it("1 行に複数ターゲットを書いた場合は全件を一覧に出す", () => {
      const { lines } = renderHelp(source(".PHONY: up down ## 起動と停止"));

      expect(lines.slice(2)).toEqual([
        `🛠  ${"up".padEnd(24)} 起動と停止`,
        `🛠  ${"down".padEnd(24)} 起動と停止`,
      ]);
    });

    it("複数ファイルを渡された順に連結する", () => {
      const { lines } = renderHelp([
        { path: "a.mk", content: ".PHONY: a ## A" },
        { path: "b.mk", content: ".PHONY: b ## B" },
      ]);

      expect(lines.slice(2)).toEqual([
        `🛠  ${"a".padEnd(24)} A`,
        `🛠  ${"b".padEnd(24)} B`,
      ]);
    });

    it("ターゲット名が桁幅を超えても切り詰めない", () => {
      const target = "very-long-target-name-exceeding-the-column";
      const { lines } = renderHelp(source(`.PHONY: ${target} ## 説明`));

      expect(lines.at(-1)).toBe(`🛠  ${target} 説明`);
    });

    it("レシピ行やコメント行は一覧にも警告にも出さない", () => {
      const { lines, undocumented } = renderHelp(source(["\techo hi", "# ただのコメント"].join("\n")));

      expect(lines).toHaveLength(2);
      expect(undocumented).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("説明コメントの無い .PHONY 行を一覧から外して報告する", () => {
      const { lines, undocumented } = renderHelp(source(".PHONY: hidden"));

      expect(lines).toHaveLength(2);
      expect(undocumented).toEqual([".makefiles/sample.mk: .PHONY: hidden"]);
    });

    it("報告にはファイルパスを添える（編集先へ辿れるようにする）", () => {
      const { undocumented } = renderHelp(source(".PHONY: hidden", ".makefiles/go/test.mk"));

      expect(undocumented[0]).toContain(".makefiles/go/test.mk");
    });

    it("## を持つが説明が空の行は一覧側で受ける", () => {
      const { lines, undocumented } = renderHelp(source(".PHONY: blank ##"));

      expect(undocumented).toEqual([]);
      expect(lines.at(-1)).toBe(`🛠  ${"blank".padEnd(24)} `);
    });
  });
});
