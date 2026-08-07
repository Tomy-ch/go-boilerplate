// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import type { PortalItem } from "../docs-json/docs-json";
import { PortalCardGrid } from "./portal-card-grid";

const document: PortalItem = {
  name: "ADR 0001",
  path: "./guides/0001.md",
  source: "docs/adr/0001.md",
  lang: "ja",
};

const generated: PortalItem = { name: "Coverage", path: "./coverage/index.html", lang: "all" };

describe("PortalCardGrid", () => {
  describe("正常系", () => {
    it("Markdown の項目を押せる操作として描画する", () => {
      render(<PortalCardGrid items={[document]} onOpenDocument={vi.fn()} />);

      expect(screen.getByRole("button", { name: "ADR 0001" })).toBeInTheDocument();
    });
    it("Markdown 以外の項目を別タブへの link として描画する", () => {
      render(<PortalCardGrid items={[generated]} onOpenDocument={vi.fn()} />);
      const link = screen.getByRole("link", { name: "Coverage" });

      expect(link).toHaveAttribute("href", "./coverage/index.html");
      expect(link).toHaveAttribute("target", "_blank");
      expect(link).toHaveAttribute("rel", "noopener noreferrer");
    });
    it("Markdown の項目を押すと呼び出し元へ項目を渡す", () => {
      const onOpenDocument = vi.fn();

      render(<PortalCardGrid items={[document]} onOpenDocument={onOpenDocument} />);
      fireEvent.click(screen.getByRole("button", { name: "ADR 0001" }));

      expect(onOpenDocument).toHaveBeenCalledWith(document);
    });
    it("出所が分かる項目は出所を添える", () => {
      render(<PortalCardGrid items={[document]} onOpenDocument={vi.fn()} />);

      expect(screen.getByText("docs/adr/0001.md")).toBeInTheDocument();
    });
    it("出所を持たない項目は経路を添える", () => {
      render(<PortalCardGrid items={[generated]} onOpenDocument={vi.fn()} />);

      expect(screen.getByText("./coverage/index.html")).toBeInTheDocument();
    });
    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(
        <PortalCardGrid items={[document, generated]} onOpenDocument={vi.fn()} />,
      );

      expect(
        (await axe(container, { rules: { "color-contrast": { enabled: false } } })).violations,
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("項目が無ければ空の一覧を描画する", () => {
      render(<PortalCardGrid items={[]} onOpenDocument={vi.fn()} />);

      expect(screen.getByRole("list").children).toHaveLength(0);
    });
  });
});
