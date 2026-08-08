import type { Element, Nodes, Root } from "hast";
import { describe, expect, it } from "vitest";
import { parseMarkdownDocument } from "./markdown-document";

function collectElements(node: Nodes): Element[] {
  const found: Element[] = [];

  const visit = (current: Nodes) => {
    if (current.type === "element") {
      found.push(current);
    }

    if ("children" in current) {
      for (const child of current.children) {
        visit(child);
      }
    }
  };

  visit(node);

  return found;
}

function tagNamesOf(root: Root): string[] {
  return collectElements(root).map((element) => element.tagName);
}

function firstElement(root: Root, tagName: string): Element | undefined {
  return collectElements(root).find((element) => element.tagName === tagName);
}

describe("parseMarkdownDocument", () => {
  describe("正常系", () => {
    it("見出しと段落を要素へ変換する", () => {
      const { root } = parseMarkdownDocument("## 節\n\n本文です。\n");

      expect(tagNamesOf(root)).toEqual(["h2", "p"]);
    });
    it("表を構造ごと変換する", () => {
      const { root } = parseMarkdownDocument("| 見出し |\n| --- |\n| 値 |\n");

      expect(tagNamesOf(root)).toContain("table");
      expect(tagNamesOf(root)).toContain("th");
      expect(tagNamesOf(root)).toContain("td");
    });
    it("コードフェンスの言語表記を残す", () => {
      const { root } = parseMarkdownDocument("```ts\nconst a = 1;\n```\n");

      expect(firstElement(root, "code")?.properties.className).toEqual(["language-ts"]);
    });
    it("文書の題を h1 として本文の先頭へ残す", () => {
      const { root } = parseMarkdownDocument("# 題\n\n## 節\n");

      expect(tagNamesOf(root)).toEqual(["h1", "h2"]);
    });
    it("空の Markdown を空の本文にする", () => {
      const { root } = parseMarkdownDocument("");

      expect(root.children).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("Markdown 内に直接書かれた script を落とす", () => {
      const { root } = parseMarkdownDocument("本文\n\n<script>alert(1)</script>\n");

      expect(tagNamesOf(root)).not.toContain("script");
    });
  });
});
