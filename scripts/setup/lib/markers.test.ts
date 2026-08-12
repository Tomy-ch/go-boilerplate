import { describe, expect, it } from "vitest";

import { stripMarkers } from "./markers";

function doc(...lines: string[]): string {
  return lines.join("\n");
}

describe("stripMarkers", () => {
  describe("正常系", () => {
    it("マーカー名を変えても同じ規則で除去する", () => {
      const content = doc("keep", "# setup-localize:begin", "drop", "# setup-localize:end", "keep2");

      expect(stripMarkers(content, "setup-localize").content).toBe(doc("keep", "keep2"));
    });

    it("別名のマーカーには反応しない", () => {
      const content = doc("keep", "# sample-api:begin", "keep2", "# sample-api:end");

      expect(stripMarkers(content, "setup-localize").removed).toBe(0);
    });

    it("行マーカーはその行だけを落とす", () => {
      expect(stripMarkers(doc("a # setup-localize:line", "b"), "setup-localize").content).toBe("b");
    });

    it("markdown コメントのマーカーも拾う", () => {
      const content = doc("| row | <!-- setup-localize:line --> |", "keep");

      expect(stripMarkers(content, "setup-localize").content).toBe("keep");
    });

    it("入れ子のブロックを深さで数える", () => {
      const content = doc(
        "keep",
        "# m:begin",
        "# m:begin",
        "drop",
        "# m:end",
        "drop2",
        "# m:end",
        "keep2",
      );

      expect(stripMarkers(content, "m").content).toBe(doc("keep", "keep2"));
    });

    it("除去した行数をマーカー行込みで数える", () => {
      expect(stripMarkers(doc("# m:begin", "drop", "# m:end"), "m").removed).toBe(3);
    });

    it("置換マーカーは有効側を落とし退避側をアンコメントする", () => {
      const content = doc(
        "# m:replace-begin",
        "  active()",
        "# m:replace-with",
        "  // = substitute()",
        "# m:replace-end",
      );

      expect(stripMarkers(content, "m").content).toBe("  substitute()");
    });

    it("HTML コメントの退避行をアンコメントして残す", () => {
      const content = doc(
        "<!-- m:replace-begin -->",
        "upstream only",
        "<!-- m:replace-with -->",
        "<!-- = general form -->",
        "<!-- m:replace-end -->",
      );

      expect(stripMarkers(content, "m").content).toBe("general form");
    });

    it("HTML コメントの退避行が空なら空行として残す", () => {
      const content = doc(
        "<!-- m:replace-begin -->",
        "upstream only",
        "<!-- m:replace-with -->",
        "<!-- = first -->",
        "<!-- = -->",
        "<!-- = second -->",
        "<!-- m:replace-end -->",
      );

      expect(stripMarkers(content, "m").content).toBe(doc("first", "", "second"));
    });

    it("HTML コメントの退避行でもインデントを保つ", () => {
      const content = doc("# m:replace-begin", "# m:replace-with", "  <!-- = nested -->", "# m:replace-end");

      expect(stripMarkers(content, "m").content).toBe("  nested");
    });

    it("`//` の退避行は行末の空白を保つ", () => {
      const content = doc("# m:replace-begin", "# m:replace-with", "// = break here  ", "# m:replace-end");

      expect(stripMarkers(content, "m").content).toBe("break here  ");
    });

    it("消した跡で空行が隣り合わないよう畳む", () => {
      const content = doc("keep", "", "# m:begin", "drop", "# m:end", "", "keep2");

      expect(stripMarkers(content, "m").content).toBe(doc("keep", "", "keep2"));
    });

    it("引用の間のブロックを抜いたら空行でなく引用の段落区切りを置く", () => {
      const content = doc("> A", "", "# m:begin", "> sample", "# m:end", "", "> B");

      expect(stripMarkers(content, "m").content).toBe(doc("> A", ">", "> B"));
    });

    // 冒頭のブロックを抜くと、繕う相手（直前の行）が 1 行も無い状態で継ぎ目に出る。
    // 先頭に空行を残すと本文が 1 行下がってしまうので、そのまま落とす。
    it("先頭のブロックを抜いたら残った空行ごと落とす", () => {
      const content = doc("# m:begin", "drop", "# m:end", "", "keep");

      expect(stripMarkers(content, "m").content).toBe("keep");
    });

    it("先頭のブロックの直後が引用でも段落区切りを差し込まない", () => {
      const content = doc("# m:begin", "drop", "# m:end", "", "> A");

      expect(stripMarkers(content, "m").content).toBe("> A");
    });

    it("継ぎ目の片側だけが引用なら空行のままにする", () => {
      const content = doc("> A", "", "# m:begin", "sample", "# m:end", "", "text");

      expect(stripMarkers(content, "m").content).toBe(doc("> A", "", "text"));
    });

    it("末尾が引用の継ぎ目で終わっても空行を落とさない", () => {
      const content = doc("> A", "", "# m:begin", "> sample", "# m:end", "");

      expect(stripMarkers(content, "m").content).toBe(doc("> A", ""));
    });

    it("行マーカーの跡でも同じように畳む", () => {
      const content = doc("keep", "", "drop # m:line", "", "keep2");

      expect(stripMarkers(content, "m").content).toBe(doc("keep", "", "keep2"));
    });

    it("消していない箇所の連続空行は畳まない", () => {
      const content = doc("keep", "", "", "keep2");

      expect(stripMarkers(content, "m").content).toBe(content);
    });

    it("コードフェンス内の連続空行を壊さない", () => {
      const content = doc("```sh", "a", "", "", "b", "```", "", "# m:begin", "drop", "# m:end");

      expect(stripMarkers(content, "m").content).toBe(doc("```sh", "a", "", "", "b", "```", ""));
    });

    it("正規表現メタ文字を含むマーカー名も literal として扱う", () => {
      expect(stripMarkers(doc("keep", "# a.b:line"), "a.b").removed).toBe(1);
      expect(stripMarkers(doc("keep", "# a.b:line"), "axb").removed).toBe(0);
    });
  });

  describe("異常系", () => {
    it("閉じられていないブロックを検出する", () => {
      expect(() => stripMarkers(doc("# m:begin", "x"), "m")).toThrow(/m:begin/);
    });

    it("対応しない end を検出する", () => {
      expect(() => stripMarkers(doc("# m:end"), "m")).toThrow(/m:end/);
    });

    it("入れ子の replace ブロックを拒否する", () => {
      const content = doc("# m:replace-begin", "# m:replace-begin");

      expect(() => stripMarkers(content, "m")).toThrow(/入れ子/);
    });

    it("replace-with の無い replace-end を拒否する", () => {
      expect(() => stripMarkers(doc("# m:replace-end"), "m")).toThrow(/replace-begin/);
    });

    it("退避側に退避コメント以外の行があれば拒否する", () => {
      const content = doc("# m:replace-begin", "# m:replace-with", "  raw()", "# m:replace-end");

      expect(() => stripMarkers(content, "m")).toThrow(/いずれかで書いてください/);
    });

    // 閉じ忘れを通すと、除去後に `<!-- = ` が本文へ残る。
    it("閉じられていない HTML コメントの退避行を拒否する", () => {
      const content = doc("# m:replace-begin", "# m:replace-with", "<!-- = unclosed", "# m:replace-end");

      expect(() => stripMarkers(content, "m")).toThrow(/いずれかで書いてください/);
    });
  });
});
