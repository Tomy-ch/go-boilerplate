// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { useCallback, useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { Toggle, toggleVariants } from "./toggle";

function ControlledFixture() {
  const [pressed, setPressed] = useState(false);
  const toggle = useCallback(() => setPressed((current) => !current), []);

  return (
    <Toggle onClick={toggle} pressed={pressed}>
      折り返す
    </Toggle>
  );
}

describe("Toggle", () => {
  describe("正常系", () => {
    it("押下状態を aria-pressed で公開する", () => {
      render(<Toggle pressed>折り返す</Toggle>);

      const toggle = screen.getByRole("button", { name: "折り返す", pressed: true });

      expect(toggle).toHaveAttribute("data-slot", "toggle");
    });

    it("未押下のときは aria-pressed が false になる", () => {
      render(<Toggle pressed={false}>折り返す</Toggle>);

      expect(screen.getByRole("button", { name: "折り返す" })).toHaveAttribute(
        "aria-pressed",
        "false",
      );
    });

    it("状態が変わってもアクセシブルな名前は変えない", () => {
      render(<ControlledFixture />);

      const toggle = screen.getByRole("button", { name: "折り返す" });
      fireEvent.click(toggle);

      expect(screen.getByRole("button", { name: "折り返す", pressed: true })).toBe(toggle);
    });

    it("呼び出し元が保持する state を反映する", () => {
      render(<ControlledFixture />);

      fireEvent.click(screen.getByRole("button", { name: "折り返す" }));

      expect(screen.getByRole("button", { name: "折り返す" })).toHaveAttribute(
        "aria-pressed",
        "true",
      );
    });

    it("既定では form を送信しない type=button になる", () => {
      render(<Toggle pressed={false}>折り返す</Toggle>);

      expect(screen.getByRole("button", { name: "折り返す" })).toHaveAttribute("type", "button");
    });

    it("native form へ載せるため type と name / value を上書きできる", () => {
      render(
        <Toggle name="density" pressed={false} type="submit" value="compact">
          表示密度
        </Toggle>,
      );

      const toggle = screen.getByRole("button", { name: "表示密度" });

      expect(toggle).toHaveAttribute("type", "submit");
      expect(toggle).toHaveAttribute("name", "density");
      expect(toggle).toHaveAttribute("value", "compact");
    });

    it("variant と size で見た目を切り替える", () => {
      render(
        <Toggle pressed={false} size="lg" variant="outline">
          折り返す
        </Toggle>,
      );

      expect(screen.getByRole("button", { name: "折り返す" })).toHaveClass("border-input", "h-10");
    });

    it("icon だけの場合も aria-label で名前を与えられる", () => {
      render(
        <Toggle aria-label="折り返す" pressed>
          <svg aria-hidden="true" />
        </Toggle>,
      );

      expect(screen.getByRole("button", { name: "折り返す", pressed: true })).toBeInTheDocument();
    });

    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(<Toggle pressed>折り返す</Toggle>);

      const result = await axe(container, { rules: { "color-contrast": { enabled: false } } });

      expect(result.violations).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("disabled のとき操作を受け付けない", () => {
      const onClick = vi.fn();
      render(
        <Toggle disabled onClick={onClick} pressed={false}>
          折り返す
        </Toggle>,
      );

      const toggle = screen.getByRole("button", { name: "折り返す" });
      fireEvent.click(toggle);

      expect(toggle).toBeDisabled();
      expect(onClick).not.toHaveBeenCalled();
    });
  });
});

describe("toggleVariants", () => {
  describe("正常系", () => {
    it("指定が無ければ既定の面と寸法を当てる", () => {
      const classes = toggleVariants();

      expect(classes).toContain("bg-transparent");
      expect(classes).toContain("h-9");
    });

    it("variant を指定すると既定の面だけを差し替える", () => {
      const classes = toggleVariants({ variant: "outline" });

      expect(classes).toContain("border-input");
      expect(classes).toContain("h-9");
    });

    it("size を指定すると既定の寸法だけを差し替える", () => {
      const classes = toggleVariants({ size: "sm" });

      expect(classes).toContain("h-8");
      expect(classes).not.toContain("h-9");
    });

    it("選択中の面を aria-pressed と data-state の両方で指定する", () => {
      const classes = toggleVariants();

      expect(classes).toContain("aria-pressed:bg-accent");
      expect(classes).toContain("data-[state=on]:bg-accent");
    });
  });
});
