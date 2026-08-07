import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { MermaidDiagram } from "./mermaid-diagram";
import { renderMermaid } from "./render-mermaid";

vi.mock("./render-mermaid", () => ({ renderMermaid: vi.fn() }));

const renderMermaidMock = vi.mocked(renderMermaid);

describe("MermaidDiagram", () => {
  describe("正常系", () => {
    it("描画された SVG を本文へ差し込む", async () => {
      renderMermaidMock.mockResolvedValue("<svg><title>図</title></svg>");

      const { container } = render(<MermaidDiagram code="graph TD; A-->B;" />);

      await waitFor(() => {
        expect(container.querySelector('[data-slot="mermaid-diagram"] svg')).toBeInTheDocument();
      });
    });
    it("描画が済むまで何も出さない", () => {
      renderMermaidMock.mockReturnValue(new Promise(() => undefined));

      const { container } = render(<MermaidDiagram code="graph TD; A-->B;" />);

      expect(container.querySelector('[data-slot="mermaid-diagram"]')).not.toBeInTheDocument();
    });
    it("選択子として書ける id を mermaid へ渡す", async () => {
      renderMermaidMock.mockResolvedValue("<svg></svg>");

      render(<MermaidDiagram code="graph TD; A-->B;" />);

      await waitFor(() => {
        expect(renderMermaidMock).toHaveBeenCalledWith(
          expect.stringMatching(/^mermaid[a-zA-Z0-9]+$/),
          "graph TD; A-->B;",
        );
      });
    });
  });

  describe("異常系", () => {
    it("描画できない図は定義をそのまま出す", async () => {
      renderMermaidMock.mockRejectedValue(new Error("文法が誤っています"));

      render(<MermaidDiagram code="graph TD; A--" />);

      expect(await screen.findByText("graph TD; A--")).toBeInTheDocument();
    });
    it("閉じた後に描画が失敗しても差し替えない", async () => {
      let fail: (error: Error) => void = () => undefined;
      renderMermaidMock.mockReturnValue(
        new Promise<string>((_, reject) => {
          fail = reject;
        }),
      );

      const { unmount } = render(<MermaidDiagram code="graph TD; A--" />);

      unmount();
      fail(new Error("文法が誤っています"));

      await waitFor(() => {
        expect(document.body).not.toHaveTextContent("graph TD; A--");
      });
    });
  });
});
