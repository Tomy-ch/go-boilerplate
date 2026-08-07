// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { useId } from "react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { Input } from "./input";

const handleChange = () => undefined;

function InvalidInputFixture() {
  const inputId = useId();
  const errorId = `${inputId}-error`;

  return (
    <div>
      <label htmlFor={inputId}>メールアドレス</label>
      <Input
        aria-describedby={errorId}
        aria-invalid={true}
        id={inputId}
        name="email"
        type="email"
      />
      <p id={errorId}>メールアドレスの形式で入力してください。</p>
    </div>
  );
}

describe("Input", () => {
  describe("正常系", () => {
    it("native input 属性を渡して単一行の入力欄を表示する", () => {
      render(<Input defaultValue="user@example.com" name="email" type="email" />);

      const input = screen.getByDisplayValue("user@example.com");

      expect(input).toHaveAttribute("data-slot", "input");
      expect(input).toHaveAttribute("name", "email");
      expect(input).toHaveAttribute("type", "email");
    });

    it("disabled 状態では native の操作不能属性を渡す", () => {
      render(<Input disabled value="変更できません" onChange={handleChange} />);

      expect(screen.getByDisplayValue("変更できません")).toBeDisabled();
    });

    it("aria-invalid で検証エラーの状態を表せる", () => {
      render(<Input aria-invalid="true" aria-label="メールアドレス" />);

      expect(screen.getByRole("textbox", { name: "メールアドレス" })).toHaveAttribute(
        "aria-invalid",
        "true",
      );
    });

    it("ラベルとエラー文を関連付けた入力は a11y 自動検査に違反しない", async () => {
      const { container } = render(<InvalidInputFixture />);

      const result = await axe(container, { rules: { "color-contrast": { enabled: false } } });

      expect(result.violations).toEqual([]);
    });
  });
});
