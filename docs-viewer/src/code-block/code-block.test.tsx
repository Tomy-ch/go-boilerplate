import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CodeBlock } from "./code-block";
import { highlightCode } from "./highlight-code";

vi.mock("./highlight-code", () => ({ highlightCode: vi.fn() }));

const highlightCodeMock = vi.mocked(highlightCode);

describe("CodeBlock", () => {
  describe("正常系", () => {
    it("読み込みが済むまで素のテキストで描く", () => {
      highlightCodeMock.mockReturnValue(new Promise(() => undefined));

      const { container } = render(<CodeBlock code="package main" language="go" />);

      expect(container.querySelector("pre code")).toHaveTextContent("package main");
    });
    it("強調表示済みの HTML へ差し替える", async () => {
      highlightCodeMock.mockResolvedValue('<span class="hljs-keyword">package</span> main');

      const { container } = render(<CodeBlock code="package main" language="go" />);

      await waitFor(() => {
        expect(container.querySelector(".hljs-keyword")).toHaveTextContent("package");
      });
    });
    it("言語表記を class として残す", () => {
      highlightCodeMock.mockResolvedValue(null);

      const { container } = render(<CodeBlock code="SELECT 1" language="sql" />);

      expect(container.querySelector("pre code")).toHaveClass("hljs", "language-sql");
    });
    it("言語表記が無ければ language- の class を付けない", () => {
      highlightCodeMock.mockResolvedValue(null);

      const { container } = render(<CodeBlock code="text" language={null} />);
      const code = container.querySelector("pre code");

      expect(code).toHaveClass("hljs");
      expect(code?.className).toBe("hljs");
    });
  });

  describe("異常系", () => {
    it("強調表示できない言語でも中身をそのまま出す", async () => {
      highlightCodeMock.mockResolvedValue(null);

      render(<CodeBlock code="graph TD;" language="mermaid" />);

      expect(await screen.findByText("graph TD;")).toBeInTheDocument();
    });
    it("閉じた後に読み込みが済んでも差し替えない", async () => {
      let complete: (html: string) => void = () => undefined;
      highlightCodeMock.mockReturnValue(
        new Promise<string | null>((resolve) => {
          complete = resolve;
        }),
      );

      const { unmount } = render(<CodeBlock code="package main" language="go" />);

      unmount();
      complete('<span class="hljs-keyword">package</span> main');

      await waitFor(() => {
        expect(document.querySelector(".hljs-keyword")).not.toBeInTheDocument();
      });
    });
    it("強調表示に失敗しても中身をそのまま出す", async () => {
      highlightCodeMock.mockRejectedValue(new Error("読み込めません"));

      render(<CodeBlock code="package main" language="go" />);

      expect(await screen.findByText("package main")).toBeInTheDocument();
    });
  });
});
