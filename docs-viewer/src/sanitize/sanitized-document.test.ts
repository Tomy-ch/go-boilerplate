import type { Element, Nodes, Root } from "hast";
import { describe, expect, it } from "vitest";
import { SanitizedDocument, toDocumentRoot } from "./sanitized-document";

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

function textOf(root: Root): string {
  let text = "";

  const visit = (current: Nodes) => {
    if (current.type === "text") {
      text += current.value;
    }

    if ("children" in current) {
      for (const child of current.children) {
        visit(child);
      }
    }
  };

  visit(root);

  return text;
}

describe("toDocumentRoot", () => {
  describe("正常系", () => {
    it("root ではないノードを root の子として包む", () => {
      const wrapped = toDocumentRoot({ type: "text", value: "本文" });

      expect(wrapped.type).toBe("root");
      expect(wrapped.children).toHaveLength(1);
    });
    it("既に root のノードをそのまま返す", () => {
      const root: Root = { type: "root", children: [] };

      expect(toDocumentRoot(root)).toBe(root);
    });
  });
});

describe("SanitizedDocument", () => {
  describe("正常系", () => {
    it("表を構造ごと通す", () => {
      const { root } = SanitizedDocument.from(
        "<table><thead><tr><th>見出し</th></tr></thead><tbody><tr><td>値</td></tr></tbody></table>",
      );

      expect(tagNamesOf(root)).toEqual(["table", "thead", "tr", "th", "tbody", "tr", "td"]);
    });
    it("コードブロックを pre と code の入れ子のまま通す", () => {
      const { root } = SanitizedDocument.from("<pre><code>const a = 1;</code></pre>");

      expect(tagNamesOf(root)).toEqual(["pre", "code"]);
    });
    it("コードブロックの言語表記を残す", () => {
      const { root } = SanitizedDocument.from(
        '<pre><code class="language-ts">const a = 1;</code></pre>',
      );

      expect(firstElement(root, "code")?.properties.className).toEqual(["language-ts"]);
    });
    it("図を alt 付きで通す", () => {
      const { root } = SanitizedDocument.from('<img src="./diagram.svg" alt="構成図">');
      const image = firstElement(root, "img");

      expect(image?.properties.src).toBe("./diagram.svg");
      expect(image?.properties.alt).toBe("構成図");
    });
    it("alt を持たない図に空の代替テキストを補う", () => {
      const { root } = SanitizedDocument.from('<img src="./diagram.svg">');

      expect(firstElement(root, "img")?.properties.alt).toBe("");
    });
    it("文書の題から細目までの見出し階層をそのまま通す", () => {
      const { root } = SanitizedDocument.from("<h1>題</h1><h2>節</h2><h6>細目</h6>");

      expect(tagNamesOf(root)).toEqual(["h1", "h2", "h6"]);
    });
    it("リポジトリ内を指す相対 link を通す", () => {
      const { root } = SanitizedDocument.from('<a href="./guides/adr.md">ADR</a>');

      expect(firstElement(root, "a")?.properties.href).toBe("./guides/adr.md");
    });
    it("表の結合と揃えの指定を残す", () => {
      const { root } = SanitizedDocument.from(
        '<table><tr><td colspan="2" align="center">値</td></tr></table>',
      );
      const cell = firstElement(root, "td");

      expect(cell?.properties.colSpan).toBe(2);
      expect(cell?.properties.align).toBe("center");
    });
  });

  describe("異常系", () => {
    it("言語表記以外の class を落とす", () => {
      const { root } = SanitizedDocument.from('<pre><code class="attacker">x</code></pre>');

      expect(firstElement(root, "code")?.properties.className).toEqual([]);
    });
    it("script を内容ごと取り除く", () => {
      const { root } = SanitizedDocument.from("<p>本文</p><script>alert(1)</script>");

      expect(tagNamesOf(root)).toEqual(["p"]);
      expect(textOf(root)).toBe("本文");
    });
    it("style を内容ごと取り除く", () => {
      const { root } = SanitizedDocument.from("<style>body{display:none}</style><p>本文</p>");

      expect(textOf(root)).toBe("本文");
    });
    it("許可していないプロトコルの link を落とす", () => {
      const { root } = SanitizedDocument.from('<a href="javascript:alert(1)">押す</a>');

      expect(firstElement(root, "a")?.properties.href).toBeUndefined();
    });
    it("外部ホストへ解決される protocol-relative な link を落とす", () => {
      const { root } = SanitizedDocument.from(
        '<a href="//attacker.example.com">内部パスに見える</a>',
      );

      expect(firstElement(root, "a")?.properties.href).toBeUndefined();
    });
    it("外部ホストへ解決される protocol-relative な図の参照を落とす", () => {
      const { root } = SanitizedDocument.from(
        '<img src="//attacker.example.com/pixel.gif" alt="x">',
      );

      expect(firstElement(root, "img")?.properties.src).toBeUndefined();
    });
    it("許可していない属性を落とす", () => {
      const { root } = SanitizedDocument.from('<p onclick="alert(1)" id="x">本文</p>');

      expect(firstElement(root, "p")?.properties).toEqual({});
    });
    it("コメントを落とす", () => {
      const { root } = SanitizedDocument.from("<p>本文</p><!-- 内部メモ -->");

      expect(root.children).toHaveLength(1);
    });
  });
});
