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

      expect(() => stripMarkers(content, "m")).toThrow(/で始めてください/);
    });
  });
});
