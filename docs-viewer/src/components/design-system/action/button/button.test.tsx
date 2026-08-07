// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";
import { Button, buttonVariants } from "./button";
import { BUTTON_SIZE, BUTTON_VARIANT } from "./button.definition";

describe("Button", () => {
  describe("正常系", () => {
    it("既定の操作ボタンを表示する", () => {
      render(<Button>保存する</Button>);

      expect(screen.getByRole("button", { name: "保存する" })).toBeVisible();
    });

    it("asChild で子要素にボタンの表現を付与する", () => {
      render(
        <Button asChild variant={BUTTON_VARIANT.OUTLINE}>
          <a href="https://github.com/">設定へ進む</a>
        </Button>,
      );

      expect(screen.getByRole("link", { name: "設定へ進む" })).toHaveAttribute(
        "href",
        "https://github.com/",
      );
    });

    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(<Button>保存する</Button>);

      expect(
        (await axe(container, { rules: { "color-contrast": { enabled: false } } })).violations,
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("disabled のときは操作を受け付けない", () => {
      const onClick = vi.fn();

      render(
        <Button disabled onClick={onClick}>
          保存する
        </Button>,
      );
      fireEvent.click(screen.getByRole("button", { name: "保存する" }));

      expect(screen.getByRole("button", { name: "保存する" })).toBeDisabled();
      expect(onClick).not.toHaveBeenCalled();
    });
  });
});

describe("buttonVariants", () => {
  describe("正常系", () => {
    it("指定が無ければ既定の面と寸法を当てる", () => {
      const classes = buttonVariants();

      expect(classes).toContain("bg-foreground");
      expect(classes).toContain("h-10");
    });

    it("variant を指定すると既定の面だけを差し替える", () => {
      const classes = buttonVariants({ variant: BUTTON_VARIANT.OUTLINE });

      expect(classes).toContain("border-border");
      expect(classes).not.toContain("bg-foreground/85");
      expect(classes).toContain("h-10");
    });

    it("size を指定すると既定の寸法だけを差し替える", () => {
      const classes = buttonVariants({ size: BUTTON_SIZE.SMALL });

      expect(classes).toContain("h-8");
      expect(classes).not.toContain("h-10");
      expect(classes).toContain("bg-foreground");
    });

    it("面と寸法に関わらず操作可能な見た目の基底は残す", () => {
      for (const variant of Object.values(BUTTON_VARIANT)) {
        expect(buttonVariants({ variant })).toContain("cursor-pointer");
        expect(buttonVariants({ variant })).toContain("disabled:opacity-50");
      }
    });
  });
});
