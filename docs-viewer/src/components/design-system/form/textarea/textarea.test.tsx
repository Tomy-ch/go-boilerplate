// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { useId } from "react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { Textarea } from "./textarea";

const handleChange = () => undefined;

function InvalidTextareaFixture() {
  const textareaId = useId();
  const errorId = `${textareaId}-error`;

  return (
    <div>
      <label htmlFor={textareaId}>配送に関する補足</label>
      <Textarea
        aria-describedby={errorId}
        aria-invalid={true}
        id={textareaId}
        name="delivery-note"
      />
      <p id={errorId}>100 文字以内で入力してください。</p>
    </div>
  );
}

describe("Textarea", () => {
  describe("正常系", () => {
    it("native textarea 属性を渡して複数行の入力欄を表示する", () => {
      render(
        <Textarea defaultValue="玄関前への置き配を希望します。" name="delivery-note" rows={4} />,
      );

      const textarea = screen.getByDisplayValue("玄関前への置き配を希望します。");

      expect(textarea).toHaveAttribute("data-slot", "textarea");
      expect(textarea).toHaveAttribute("name", "delivery-note");
      expect(textarea).toHaveAttribute("rows", "4");
    });

    it("disabled 状態では native の操作不能属性を渡す", () => {
      render(<Textarea disabled value="変更できません" onChange={handleChange} />);

      expect(screen.getByDisplayValue("変更できません")).toBeDisabled();
    });

    it("aria-invalid で検証エラーの状態を表せる", () => {
      render(<Textarea aria-invalid="true" aria-label="配送に関する補足" />);

      expect(screen.getByRole("textbox", { name: "配送に関する補足" })).toHaveAttribute(
        "aria-invalid",
        "true",
      );
    });

    it("ラベルとエラー文を関連付けた入力は a11y 自動検査に違反しない", async () => {
      const { container } = render(<InvalidTextareaFixture />);

      const result = await axe(container, { rules: { "color-contrast": { enabled: false } } });

      expect(result.violations).toEqual([]);
    });
  });
});
