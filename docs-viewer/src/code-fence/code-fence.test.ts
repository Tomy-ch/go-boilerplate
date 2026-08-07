import type { Element, ElementContent } from "hast";
import { describe, expect, it } from "vitest";

import { readCodeFence } from "./code-fence";

function element(tagName: string, properties: Element["properties"], children: ElementContent[]) {
  return { type: "element", tagName, properties, children } satisfies Element;
}

function text(value: string): ElementContent {
  return { type: "text", value };
}

function codeFence(className: string[] | undefined, code: string): Element {
  return element("pre", {}, [element("code", className ? { className } : {}, [text(code)])]);
}

describe("readCodeFence", () => {
  describe("正常系", () => {
    it("言語表記と中身を取り出す", () => {
      expect(readCodeFence(codeFence(["language-go"], "package main\n"))).toEqual({
        language: "go",
        code: "package main\n",
      });
    });
    it("言語表記が無いフェンスでは言語を未指定にする", () => {
      expect(readCodeFence(codeFence(undefined, "text\n"))).toEqual({
        language: null,
        code: "text\n",
      });
    });
    it("language- 以外の class が混ざっても言語表記を選び出す", () => {
      expect(readCodeFence(codeFence(["hljs", "language-sql"], "SELECT 1"))).toEqual({
        language: "sql",
        code: "SELECT 1",
      });
    });
    it("入れ子の要素を含む中身もテキストとして連結する", () => {
      const node = element("pre", {}, [
        element("code", { className: ["language-go"] }, [
          text("var "),
          element("span", {}, [text("x")]),
          text(" int"),
        ]),
      ]);

      expect(readCodeFence(node)).toEqual({ language: "go", code: "var x int" });
    });
    it("code の前後にある空白だけの text を無視する", () => {
      const node = element("pre", {}, [
        text("\n"),
        element("code", { className: ["language-go"] }, [text("x")]),
        text("\n"),
      ]);

      expect(readCodeFence(node)).toEqual({ language: "go", code: "x" });
    });
  });

  describe("異常系", () => {
    it("code を子に持たない pre はフェンスとして扱わない", () => {
      expect(readCodeFence(element("pre", {}, [text("整形済みテキスト")]))).toBeNull();
    });
    it("code が複数ある pre はフェンスとして扱わない", () => {
      const node = element("pre", {}, [
        element("code", {}, [text("a")]),
        element("code", {}, [text("b")]),
      ]);

      expect(readCodeFence(node)).toBeNull();
    });
    it("子を持たない pre はフェンスとして扱わない", () => {
      expect(readCodeFence(element("pre", {}, []))).toBeNull();
    });
    it("language- の後ろが空の class は言語表記として採らない", () => {
      expect(readCodeFence(codeFence(["language-"], "x"))).toEqual({ language: null, code: "x" });
    });
  });
});

describe("textOf を経由するノード種別", () => {
  describe("異常系", () => {
    // sanitize 後は text だけになるが、comment のような別種が残る木を渡されても
    // 中身を落とさず、未知の種別だけを空として飛ばす。
    it("text でも element でもないノードは空として飛ばす", () => {
      const fence = element("pre", {}, [
        element("code", { className: ["language-go"] }, [
          { type: "comment", value: "消える" } as never,
          text("kept"),
        ]),
      ]);

      expect(readCodeFence(fence)).toEqual({ language: "go", code: "kept" });
    });
  });
});
