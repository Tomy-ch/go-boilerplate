// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { useId } from "react";
import { describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { Label } from "../label/label";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  InputGroupText,
  InputGroupTextarea,
} from "./input-group";
import { INPUT_GROUP_ADDON_ALIGN, INPUT_GROUP_BUTTON_SIZE } from "./input-group.definition";

type FormSubmitHandler = NonNullable<ComponentProps<"form">["onSubmit"]>;

function UnitInputGroup() {
  const quantityId = useId();
  const noteId = useId();

  return (
    <div>
      <Label htmlFor={quantityId}>数量</Label>
      <InputGroup>
        <InputGroupInput
          aria-describedby={noteId}
          id={quantityId}
          inputMode="numeric"
          name="quantity"
        />
        <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.INLINE_END}>
          <InputGroupText>kg</InputGroupText>
        </InputGroupAddon>
      </InputGroup>
      <p id={noteId}>小数点以下は切り捨てます。</p>
    </div>
  );
}

describe("InputGroup", () => {
  describe("正常系", () => {
    it("入力欄と addon を一つの group としてまとめる", () => {
      render(<UnitInputGroup />);

      const [group, addon] = screen.getAllByRole("group");
      expect(group).toContainElement(screen.getByRole("textbox"));
      expect(group).toContainElement(addon ?? null);
      expect(screen.getByRole("textbox")).toHaveAttribute("name", "quantity");
    });

    it("入力欄は Label のアクセシブルな名前と補足の説明を持つ", () => {
      render(<UnitInputGroup />);

      const control = screen.getByRole("textbox");
      expect(control).toHaveAccessibleName("数量");
      expect(control).toHaveAccessibleDescription("小数点以下は切り捨てます。");
    });

    it("align を指定すると addon の配置を data 属性で表す", () => {
      const { container } = render(
        <InputGroup>
          <InputGroupAddon>
            <InputGroupText>前</InputGroupText>
          </InputGroupAddon>
          <InputGroupInput aria-label="値" name="value" />
          <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.BLOCK_END}>
            <InputGroupText>後</InputGroupText>
          </InputGroupAddon>
        </InputGroup>,
      );

      const addons = container.querySelectorAll("[data-slot='input-group-addon']");
      expect(addons[0]).toHaveAttribute("data-align", INPUT_GROUP_ADDON_ALIGN.INLINE_START);
      expect(addons[1]).toHaveAttribute("data-align", INPUT_GROUP_ADDON_ALIGN.BLOCK_END);
    });

    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(<UnitInputGroup />);

      expect(
        (await axe(container, { rules: { "color-contrast": { enabled: false } } })).violations,
      ).toEqual([]);
    });
  });
});

describe("InputGroupAddon", () => {
  describe("正常系", () => {
    it("addon の空白を押すと同じ枠内の入力欄へ focus が移る", () => {
      render(<UnitInputGroup />);

      fireEvent.click(screen.getByText("kg"));

      expect(screen.getByRole("textbox")).toHaveFocus();
    });

    it("addon の空白を押すと複数行の入力欄へも focus が移る", () => {
      render(
        <InputGroup>
          <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.BLOCK_START}>
            <InputGroupText>補足</InputGroupText>
          </InputGroupAddon>
          <InputGroupTextarea aria-label="補足" name="note" rows={3} />
        </InputGroup>,
      );

      fireEvent.click(screen.getByText("補足", { selector: "span" }));

      expect(screen.getByRole("textbox")).toHaveFocus();
    });
  });

  describe("異常系", () => {
    it("addon 内の button を押したときは入力欄へ focus を移さない", () => {
      const handleClick = vi.fn();
      render(
        <InputGroup>
          <InputGroupInput aria-label="キーワード" name="keyword" />
          <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.INLINE_END}>
            <InputGroupButton onClick={handleClick}>消去</InputGroupButton>
          </InputGroupAddon>
        </InputGroup>,
      );

      fireEvent.click(screen.getByRole("button", { name: "消去" }));

      expect(handleClick).toHaveBeenCalledOnce();
      expect(screen.getByRole("textbox")).not.toHaveFocus();
    });

    it("入力欄を持たない枠の addon を押しても何も起きない", () => {
      render(
        <InputGroup>
          <InputGroupAddon>
            <InputGroupText>単位のみ</InputGroupText>
          </InputGroupAddon>
        </InputGroup>,
      );

      fireEvent.click(screen.getByText("単位のみ"));

      expect(document.body).toHaveFocus();
    });

    it("枠の外に置いた addon を押しても何も起きない", () => {
      render(
        <InputGroupAddon>
          <InputGroupText>単体</InputGroupText>
        </InputGroupAddon>,
      );

      fireEvent.click(screen.getByText("単体"));

      expect(document.body).toHaveFocus();
    });
  });
});

describe("InputGroupButton", () => {
  describe("正常系", () => {
    it("type='submit' を指定すると入力欄の name と値を送信する", () => {
      const submitted = vi.fn();
      const handleSubmit: FormSubmitHandler = (event) => {
        event.preventDefault();
        submitted(Object.fromEntries(new FormData(event.currentTarget)));
      };
      render(
        <form onSubmit={handleSubmit}>
          <InputGroup>
            <InputGroupInput aria-label="キーワード" defaultValue="靴" name="keyword" />
            <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.INLINE_END}>
              <InputGroupButton size={INPUT_GROUP_BUTTON_SIZE.SMALL} type="submit" variant="outline">
                検索
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
        </form>,
      );

      fireEvent.click(screen.getByRole("button", { name: "検索" }));

      expect(submitted).toHaveBeenCalledWith({ keyword: "靴" });
    });

    it("アイコンだけの button は aria-label をアクセシブルな名前にする", () => {
      render(
        <InputGroup>
          <InputGroupInput aria-label="キーワード" name="keyword" />
          <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.INLINE_END}>
            <InputGroupButton aria-label="入力を消去" size={INPUT_GROUP_BUTTON_SIZE.ICON_EXTRA_SMALL}>
              <svg aria-hidden="true" viewBox="0 0 16 16" />
            </InputGroupButton>
          </InputGroupAddon>
        </InputGroup>,
      );

      expect(screen.getByRole("button")).toHaveAccessibleName("入力を消去");
    });
  });

  describe("異常系", () => {
    it("既定では form を送信しない", () => {
      const handleSubmit = vi.fn<FormSubmitHandler>((event) => {
        event.preventDefault();
      });
      render(
        <form onSubmit={handleSubmit}>
          <InputGroup>
            <InputGroupInput aria-label="キーワード" defaultValue="靴" name="keyword" />
            <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.INLINE_END}>
              <InputGroupButton>消去</InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
        </form>,
      );

      fireEvent.click(screen.getByRole("button", { name: "消去" }));

      expect(handleSubmit).not.toHaveBeenCalled();
    });
  });
});

describe("InputGroupInput", () => {
  describe("正常系", () => {
    it("native 属性と invalid の指定をそのまま入力欄へ渡す", () => {
      render(
        <InputGroup>
          <InputGroupInput
            aria-invalid="true"
            aria-label="数量"
            defaultValue="-1"
            inputMode="numeric"
            name="quantity"
            required
          />
        </InputGroup>,
      );

      const control = screen.getByRole("textbox", { name: "数量" });
      expect(control).toBeRequired();
      expect(control).toBeInvalid();
      expect(control).toHaveAttribute("inputmode", "numeric");
      expect(control).toHaveValue("-1");
    });
  });

  describe("異常系", () => {
    it("disabled の入力欄は操作を受け付けない", () => {
      render(
        <InputGroup data-disabled="true">
          <InputGroupInput aria-label="数量" disabled name="quantity" />
        </InputGroup>,
      );

      expect(screen.getByRole("textbox", { name: "数量" })).toBeDisabled();
    });
  });
});

describe("InputGroupTextarea", () => {
  describe("正常系", () => {
    it("複数行の入力欄として native 属性を渡す", () => {
      render(
        <InputGroup>
          <InputGroupTextarea aria-label="補足" name="note" rows={4} />
        </InputGroup>,
      );

      const control = screen.getByRole("textbox", { name: "補足" });
      expect(control.tagName).toBe("TEXTAREA");
      expect(control).toHaveAttribute("rows", "4");
      expect(control).toHaveAttribute("name", "note");
    });
  });
});

describe("InputGroupText", () => {
  describe("正常系", () => {
    it("単位や記号を span として addon 内へ置く", () => {
      render(<InputGroupText>円</InputGroupText>);

      expect(screen.getByText("円").tagName).toBe("SPAN");
    });

    it("入力欄より弱い前景色で描き、名前として読ませない", () => {
      render(<InputGroupText>円</InputGroupText>);

      expect(screen.getByText("円")).toHaveClass("text-muted-foreground");
    });

    it("子のアイコンに大きさの指定が無ければ既定の大きさへ揃える", () => {
      render(<InputGroupText>円</InputGroupText>);

      expect(screen.getByText("円")).toHaveClass("[&_svg:not([class*='size-'])]:size-4");
    });

    it("呼び出し元の className を足しても既定の見た目を落とさない", () => {
      render(<InputGroupText className="font-mono">円</InputGroupText>);

      expect(screen.getByText("円")).toHaveClass("text-sm", "font-mono");
    });
  });
});
