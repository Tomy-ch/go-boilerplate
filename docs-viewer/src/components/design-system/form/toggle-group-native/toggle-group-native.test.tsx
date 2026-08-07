// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { ToggleGroupNative, ToggleGroupNativeItem } from "./toggle-group-native";

function SingleFixture() {
  return (
    <ToggleGroupNative aria-label="表示通貨">
      <ToggleGroupNativeItem defaultChecked name="currency" value="jpy">
        JPY
      </ToggleGroupNativeItem>
      <ToggleGroupNativeItem name="currency" value="usd">
        USD
      </ToggleGroupNativeItem>
    </ToggleGroupNative>
  );
}

function MultipleFixture() {
  return (
    <ToggleGroupNative aria-label="表示する列">
      <ToggleGroupNativeItem name="columns" type="checkbox" value="price">
        価格
      </ToggleGroupNativeItem>
      <ToggleGroupNativeItem name="columns" type="checkbox" value="stock">
        アーカイブ
      </ToggleGroupNativeItem>
    </ToggleGroupNative>
  );
}

describe("ToggleGroupNative", () => {
  describe("正常系", () => {
    it("名前を持つ group として公開する", () => {
      render(<SingleFixture />);

      const group = screen.getByRole("group", { name: "表示通貨" });

      expect(group).toHaveAttribute("data-slot", "toggle-group-native");
    });

    it("排他選択は radio として公開し、既定は 1 つだけ選択される", () => {
      render(<SingleFixture />);

      const options = screen.getAllByRole("radio");

      expect(options).toHaveLength(2);
      expect(screen.getByRole("radio", { name: "JPY" })).toBeChecked();
      expect(screen.getByRole("radio", { name: "USD" })).not.toBeChecked();
    });

    it("複数選択は checkbox として公開する", () => {
      render(<MultipleFixture />);

      expect(screen.getAllByRole("checkbox")).toHaveLength(2);
    });

    it("form へ送る name と value を native 属性として持つ", () => {
      render(<SingleFixture />);

      const option = screen.getByRole("radio", { name: "USD" });

      expect(option).toHaveAttribute("name", "currency");
      expect(option).toHaveAttribute("value", "usd");
    });

    it("選ぶと排他的に切り替わる", () => {
      render(<SingleFixture />);

      fireEvent.click(screen.getByRole("radio", { name: "USD" }));

      expect(screen.getByRole("radio", { name: "USD" })).toBeChecked();
      expect(screen.getByRole("radio", { name: "JPY" })).not.toBeChecked();
    });

    it("複数選択では同時に選べる", () => {
      render(<MultipleFixture />);

      fireEvent.click(screen.getByRole("checkbox", { name: "価格" }));
      fireEvent.click(screen.getByRole("checkbox", { name: "アーカイブ" }));

      expect(screen.getByRole("checkbox", { name: "価格" })).toBeChecked();
      expect(screen.getByRole("checkbox", { name: "アーカイブ" })).toBeChecked();
    });

    it("input は視覚的に隠しても支援技術と keyboard から到達できる", () => {
      render(<SingleFixture />);

      const option = screen.getByRole("radio", { name: "JPY" });

      expect(option).toHaveClass("sr-only");
      expect(option).not.toHaveAttribute("aria-hidden");
      expect(option).not.toBeDisabled();
    });

    it("選択中の面と大きさを Toggle と同じ token で示す", () => {
      render(<SingleFixture />);

      const label = screen.getByText("JPY");

      expect(label).toHaveClass("has-[:checked]:bg-accent", "h-9");
    });

    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(<SingleFixture />);

      const result = await axe(container, { rules: { "color-contrast": { enabled: false } } });

      expect(result.violations).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("disabled の項目は操作を受け付けない", () => {
      render(
        <ToggleGroupNative aria-label="表示通貨">
          <ToggleGroupNativeItem defaultChecked name="currency" value="jpy">
            JPY
          </ToggleGroupNativeItem>
          <ToggleGroupNativeItem disabled name="currency" value="eur">
            EUR
          </ToggleGroupNativeItem>
        </ToggleGroupNative>,
      );

      const disabled = screen.getByRole("radio", { name: "EUR" });

      expect(disabled).toBeDisabled();
      expect(disabled).not.toBeChecked();
    });
  });
});

describe("ToggleGroupNativeItem", () => {
  describe("正常系", () => {
    it("children を項目のアクセシブルな名前にする", () => {
      render(<ToggleGroupNativeItem name="view" value="list" />);
      render(<ToggleGroupNativeItem name="view" value="grid">一覧</ToggleGroupNativeItem>);

      expect(screen.getByRole("radio", { name: "一覧" })).toBeInTheDocument();
    });

    it("既定では排他選択の radio として公開する", () => {
      render(<ToggleGroupNativeItem name="view" value="list">一覧</ToggleGroupNativeItem>);

      expect(screen.getByRole("radio", { name: "一覧" })).toHaveAttribute("type", "radio");
    });

    it("type を渡すと複数選択の checkbox にできる", () => {
      render(
        <ToggleGroupNativeItem name="view" type="checkbox" value="list">
          一覧
        </ToggleGroupNativeItem>,
      );

      expect(screen.getByRole("checkbox", { name: "一覧" })).toBeInTheDocument();
    });

    it("input は視覚的に隠しても支援技術から到達できる", () => {
      render(<ToggleGroupNativeItem name="view" value="list">一覧</ToggleGroupNativeItem>);

      expect(screen.getByRole("radio", { name: "一覧" })).toHaveClass("sr-only");
    });

    it("選択中の面は Toggle と同じ token で示す", () => {
      render(<ToggleGroupNativeItem name="view" value="list">一覧</ToggleGroupNativeItem>);

      expect(screen.getByText("一覧")).toHaveClass("has-[:checked]:bg-accent");
    });

    it("両端だけを丸めて隣と繋げる", () => {
      render(<ToggleGroupNativeItem name="view" value="list">一覧</ToggleGroupNativeItem>);

      expect(screen.getByText("一覧")).toHaveClass("first:rounded-l-md", "last:rounded-r-md");
    });
  });
});
