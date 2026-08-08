// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "./accordion";

function ExampleAccordion() {
  return (
    <Accordion>
      <AccordionItem>
        <AccordionTrigger>補足情報</AccordionTrigger>
        <AccordionContent>必要なときに確認する内容です。</AccordionContent>
      </AccordionItem>
    </Accordion>
  );
}

describe("Accordion", () => {
  describe("正常系", () => {
    it("native details と summary で詳細を開閉する", () => {
      render(<ExampleAccordion />);

      const item = screen.getByText("補足情報").closest("details");
      if (item === null) throw new Error("accordion item が見つかりません。");

      expect(item).not.toHaveAttribute("open");
      fireEvent.click(screen.getByText("補足情報"));
      expect(item).toHaveAttribute("open");
    });

    it("複数の項目を初期状態で開ける", () => {
      render(
        <Accordion>
          <AccordionItem open>
            <AccordionTrigger>一つ目</AccordionTrigger>
            <AccordionContent>内容一</AccordionContent>
          </AccordionItem>
          <AccordionItem open>
            <AccordionTrigger>二つ目</AccordionTrigger>
            <AccordionContent>内容二</AccordionContent>
          </AccordionItem>
        </Accordion>,
      );

      expect(document.querySelectorAll("details[open]")).toHaveLength(2);
    });

    it("hover 時も背景色と文字色のコントラストを保つ", () => {
      render(<ExampleAccordion />);

      expect(screen.getByText("補足情報")).toHaveClass(
        "hover:bg-foreground",
        "hover:text-background",
      );
    });

    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(<ExampleAccordion />);
      expect(
        (await axe(container, { rules: { "color-contrast": { enabled: false } } })).violations,
      ).toEqual([]);
    });
  });
});

describe("AccordionItem", () => {
  describe("正常系", () => {
    it("JavaScript なしで開閉できる native details として描く", () => {
      const { container } = render(<AccordionItem>項目</AccordionItem>);

      expect(container.querySelector("details")).toHaveAttribute("data-slot", "accordion-item");
    });

    it("開閉状態を子から参照できるよう group 名を付ける", () => {
      const { container } = render(<AccordionItem>項目</AccordionItem>);

      expect(container.querySelector("details")).toHaveClass("group/accordion-item");
    });

    it("open を渡すと初期状態で開く", () => {
      const { container } = render(<AccordionItem open>項目</AccordionItem>);

      expect(container.querySelector("details")).toHaveAttribute("open");
    });
  });
});

describe("AccordionTrigger", () => {
  describe("正常系", () => {
    it("details の直接の子となる summary として描く", () => {
      const { container } = render(
        <AccordionItem>
          <AccordionTrigger>補足情報</AccordionTrigger>
        </AccordionItem>,
      );

      expect(container.querySelector("details > summary")).toHaveAttribute(
        "data-slot",
        "accordion-trigger",
      );
    });

    it("ブラウザ既定の三角マークを隠して独自の指標に置き換える", () => {
      const { container } = render(<AccordionTrigger>補足情報</AccordionTrigger>);

      expect(container.querySelector("summary")).toHaveClass(
        "list-none",
        "[&::-webkit-details-marker]:hidden",
      );
    });

    it("開閉の指標は装飾なので読み上げから外す", () => {
      const { container } = render(<AccordionTrigger>補足情報</AccordionTrigger>);

      expect(container.querySelector("svg")).toHaveAttribute("aria-hidden", "true");
    });

    it("開いている間は指標を反転させる", () => {
      const { container } = render(<AccordionTrigger>補足情報</AccordionTrigger>);

      expect(container.querySelector("svg")).toHaveClass(
        "group-open/accordion-item:rotate-180",
      );
    });
  });
});

describe("AccordionContent", () => {
  describe("正常系", () => {
    it("詳細内容として accordion-content の slot を名乗る", () => {
      render(<AccordionContent>詳細</AccordionContent>);

      expect(screen.getByText("詳細")).toHaveAttribute("data-slot", "accordion-content");
    });

    it("見出しとの間に区切りを引き、本文より弱い前景色で描く", () => {
      render(<AccordionContent>詳細</AccordionContent>);

      expect(screen.getByText("詳細")).toHaveClass("border-t", "text-muted-foreground");
    });
  });
});
