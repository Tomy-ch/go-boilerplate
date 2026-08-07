// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import type { PortalGroup, PortalReferenceLink } from "../docs-json/docs-json";
import { PortalSidebar } from "./portal-sidebar";

const groups: PortalGroup[] = [
  {
    title: "Architecture",
    slug: "architecture",
    sections: [{ id: "adr", slug: "adr", title: "ADR", items: [] }],
  },
  {
    title: "Get Started",
    slug: "get-started",
    sections: [{ id: "setup", slug: "setup", title: "Setup", items: [] }],
  },
];

const referenceLinks: PortalReferenceLink[] = [
  { sectionId: "coverage", title: "Coverage", path: "./coverage/index.html" },
];

describe("PortalSidebar", () => {
  describe("正常系", () => {
    it("group の見出しは開閉だけを担い link を持たない", () => {
      render(<PortalSidebar activeGroupSlug="architecture" groups={groups} referenceLinks={[]} />);

      expect(screen.getByText("Architecture")).toBeInTheDocument();
      expect(screen.queryByRole("link", { name: "Architecture" })).not.toBeInTheDocument();
    });
    it("section への導線に group と section の両方を含める", () => {
      render(<PortalSidebar activeGroupSlug="architecture" groups={groups} referenceLinks={[]} />);

      expect(screen.getByRole("link", { name: "ADR" })).toHaveAttribute(
        "href",
        "#/architecture/adr",
      );
    });
    it("表示中の group を開いた状態で描画する", () => {
      const { container } = render(
        <PortalSidebar activeGroupSlug="architecture" groups={groups} referenceLinks={[]} />,
      );
      const items = container.querySelectorAll("details");

      expect(items[0]?.open).toBe(true);
      expect(items[1]?.open).toBe(false);
    });
    it("常設リンクを別タブへの link として並べる", () => {
      render(
        <PortalSidebar
          activeGroupSlug="architecture"
          groups={groups}
          referenceLinks={referenceLinks}
        />,
      );
      const link = screen.getByRole("link", { name: "Coverage" });

      expect(link).toHaveAttribute("target", "_blank");
      expect(link).toHaveAttribute("rel", "noopener noreferrer");
    });
    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(
        <PortalSidebar
          activeGroupSlug="architecture"
          groups={groups}
          referenceLinks={referenceLinks}
        />,
      );

      expect(
        (await axe(container, { rules: { "color-contrast": { enabled: false } } })).violations,
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("表示中の group が無ければ全て閉じた状態にする", () => {
      const { container } = render(
        <PortalSidebar activeGroupSlug={null} groups={groups} referenceLinks={[]} />,
      );

      expect([...container.querySelectorAll("details")].every((item) => !item.open)).toBe(true);
    });
    it("常設リンクが無ければその区画を描画しない", () => {
      render(<PortalSidebar activeGroupSlug="architecture" groups={groups} referenceLinks={[]} />);

      expect(screen.queryByRole("heading", { name: "Reference" })).not.toBeInTheDocument();
    });
  });
});
