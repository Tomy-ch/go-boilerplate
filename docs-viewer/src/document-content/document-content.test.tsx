// @vitest-environment jsdom

import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { highlightCode } from "../code-block/highlight-code";
import { renderMermaid } from "../mermaid-diagram/render-mermaid";
import { parseMarkdownDocument } from "../markdown/markdown-document";
import { SanitizedDocument } from "../sanitize/sanitized-document";
import { DocumentContent } from "./document-content";

// 強調表示と図の描画は本文の構造と別の関心で、それぞれの単体テストが持つ。ここで検査するのは
// コードフェンスをどちらへ振り分けるかだけなので、実物の読み込みは呼び出しへ置き換える。
vi.mock("../code-block/highlight-code", () => ({ highlightCode: vi.fn().mockResolvedValue(null) }));
vi.mock("../mermaid-diagram/render-mermaid", () => ({
  renderMermaid: vi.fn().mockResolvedValue("<svg></svg>"),
}));

describe("DocumentContent", () => {
  describe("正常系", () => {
    it("木の要素を対応する DOM 要素として描画する", () => {
      render(<DocumentContent content={SanitizedDocument.from("<h2>節</h2><p>本文</p>")} />);

      expect(screen.getByRole("heading", { level: 2, name: "節" })).toBeInTheDocument();
      expect(screen.getByText("本文")).toBeInTheDocument();
    });
    it("表を表として描画する", () => {
      const html =
        "<table><thead><tr><th>見出し</th></tr></thead><tbody><tr><td>値</td></tr></tbody></table>";

      render(<DocumentContent content={SanitizedDocument.from(html)} />);

      expect(screen.getByRole("table")).toBeInTheDocument();
      expect(screen.getByRole("columnheader", { name: "見出し" })).toBeInTheDocument();
      expect(screen.getByRole("cell", { name: "値" })).toBeInTheDocument();
    });
    it("link の href を保つ", () => {
      render(<DocumentContent content={SanitizedDocument.from('<a href="./adr.md">ADR</a>')} />);

      expect(screen.getByRole("link", { name: "ADR" })).toHaveAttribute("href", "./adr.md");
    });
    it("ドキュメント用の組版を既定で当てる", () => {
      const { container } = render(
        <DocumentContent content={SanitizedDocument.from("<p>本文</p>")} />,
      );

      expect(container.querySelector('[data-slot="document-content"]')).toHaveClass(
        "typeset",
        "typeset-docs",
      );
    });
    it("className を重ねても組版の class が残る", () => {
      const { container } = render(
        <DocumentContent className="max-w-prose" content={SanitizedDocument.from("<p>本文</p>")} />,
      );
      const root = container.querySelector('[data-slot="document-content"]');

      expect(root).toHaveClass("typeset", "typeset-docs", "max-w-prose");
    });
    it("native div 属性を外枠へ渡す", () => {
      const { container } = render(
        <DocumentContent content={SanitizedDocument.from("<p>本文</p>")} lang="en" />,
      );

      expect(container.querySelector('[data-slot="document-content"]')).toHaveAttribute(
        "lang",
        "en",
      );
    });
    it("mermaid のフェンスを図として描画する", async () => {
      const { container } = render(
        <DocumentContent content={parseMarkdownDocument("```mermaid\ngraph TD; A-->B;\n```")} />,
      );

      await waitFor(() => {
        expect(container.querySelector('[data-slot="mermaid-diagram"] svg')).toBeInTheDocument();
      });
      expect(renderMermaid).toHaveBeenCalledWith(expect.any(String), "graph TD; A-->B;\n");
    });
    it("mermaid 以外のフェンスを強調表示の対象にする", async () => {
      render(<DocumentContent content={parseMarkdownDocument("```go\npackage main\n```")} />);

      await waitFor(() => {
        expect(highlightCode).toHaveBeenCalledWith("package main\n", "go");
      });
      expect(screen.getByText("package main")).toHaveClass("hljs", "language-go");
    });
    it("フェンスでない pre はそのまま描画する", () => {
      const { container } = render(
        <DocumentContent content={SanitizedDocument.from("<pre>整形済みテキスト</pre>")} />,
      );

      expect(container.querySelector("pre")).toHaveTextContent("整形済みテキスト");
      expect(container.querySelector("pre code")).not.toBeInTheDocument();
    });
    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(
        <DocumentContent content={SanitizedDocument.from("<h2>節</h2><p>本文</p>")} />,
      );

      expect(
        (await axe(container, { rules: { "color-contrast": { enabled: false } } })).violations,
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("本文が空の場合は空の枠を描画する", () => {
      const { container } = render(<DocumentContent content={SanitizedDocument.from("")} />);

      expect(container.querySelector('[data-slot="document-content"]')?.childNodes).toHaveLength(0);
    });
  });
});
