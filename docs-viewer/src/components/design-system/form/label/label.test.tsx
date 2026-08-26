// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { useId } from "react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { Label } from "./label";

function LabelFixture() {
  const inputId = useId();

  return (
    <div>
      <Label htmlFor={inputId}>表示名</Label>
      <input id={inputId} name="display-name" />
    </div>
  );
}

describe("Label", () => {
  describe("正常系", () => {
    it("htmlFor で対応する control の項目名を関連付ける", () => {
      render(<LabelFixture />);

      const input = screen.getByRole("textbox", { name: "表示名" });

      expect(input).toHaveAttribute("name", "display-name");
      expect(screen.getByText("表示名")).toHaveAttribute("data-slot", "label");
    });

    it("className を既定の見た目へ追加できる", () => {
      render(<Label className="text-xs">補足</Label>);

      expect(screen.getByText("補足")).toHaveClass("text-xs");
    });

    it("control と関連付けたラベルは a11y 自動検査に違反しない", async () => {
      const { container } = render(<LabelFixture />);

      const result = await axe(container, { rules: { "color-contrast": { enabled: false } } });

      expect(result.violations).toEqual([]);
    });
  });
});
