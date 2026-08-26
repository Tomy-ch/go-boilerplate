import { describe, expect, it } from "vitest";

import { highlightCode } from "./highlight-code";

describe("highlightCode", () => {
  describe("正常系", () => {
    it("知っている言語を scope class 付きの HTML へ変換する", async () => {
      const html = await highlightCode("package main", "go");

      expect(html).toContain("hljs-keyword");
      expect(html).toContain("package");
    });
    it("記号を escape してから強調表示する", async () => {
      const html = await highlightCode('const x = "<script>"', "javascript");

      expect(html).not.toContain("<script>");
      expect(html).toContain("&lt;script&gt;");
    });
  });

  describe("異常系", () => {
    it("言語表記が無ければ変換しない", async () => {
      expect(await highlightCode("package main", null)).toBeNull();
    });
    it("highlight.js が知らない言語は変換しない", async () => {
      expect(await highlightCode("graph TD;", "mermaid")).toBeNull();
    });
  });
});
