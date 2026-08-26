// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "./card";

describe("Card", () => {
  describe("正常系", () => {
    it("各領域を合成して表示する", () => {
      render(
        <Card>
          <CardHeader>
            <CardTitle>設定の概要</CardTitle>
            <CardDescription>確認してください</CardDescription>
            <CardAction>変更</CardAction>
          </CardHeader>
          <CardContent>設定は有効です</CardContent>
          <CardFooter>詳細を見る</CardFooter>
        </Card>,
      );

      expect(screen.getByText("設定の概要")).toHaveAttribute("data-slot", "card-title");
      expect(screen.getByText("確認してください")).toHaveAttribute("data-slot", "card-description");
      expect(screen.getByText("変更")).toHaveAttribute("data-slot", "card-action");
      expect(screen.getByText("設定は有効です")).toHaveAttribute("data-slot", "card-content");
      expect(screen.getByText("詳細を見る")).toHaveAttribute("data-slot", "card-footer");
    });

    it("呼び出し元の className を各領域へ追加する", () => {
      const { container } = render(
        <Card className="w-80">
          <CardHeader className="border-b border-border">
            <CardTitle className="text-lg">見出し</CardTitle>
          </CardHeader>
          <CardContent className="text-sm">本文</CardContent>
          <CardFooter className="border-t border-border">補助操作</CardFooter>
        </Card>,
      );

      expect(container.querySelector('[data-slot="card"]')).toHaveClass("w-80");
      expect(container.querySelector('[data-slot="card-header"]')).toHaveClass("border-b");
      expect(container.querySelector('[data-slot="card-title"]')).toHaveClass("text-lg");
      expect(container.querySelector('[data-slot="card-content"]')).toHaveClass("text-sm");
      expect(container.querySelector('[data-slot="card-footer"]')).toHaveClass("border-t");
    });

    it("a11y 自動検査に違反しない", async () => {
      const { container } = render(
        <Card>
          <CardHeader>
            <CardTitle>設定の概要</CardTitle>
            <CardDescription>現在の状態を確認できます。</CardDescription>
          </CardHeader>
          <CardContent>設定は有効です。</CardContent>
          <CardFooter>補助情報</CardFooter>
        </Card>,
      );

      expect(
        (await axe(container, { rules: { "color-contrast": { enabled: false } } })).violations,
      ).toEqual([]);
    });
  });
});

describe("CardHeader", () => {
  describe("正常系", () => {
    it("見出し領域として card-header の slot を名乗る", () => {
      render(<CardHeader>見出し領域</CardHeader>);

      expect(screen.getByText("見出し領域")).toHaveAttribute("data-slot", "card-header");
    });

    it("CardAction を含むときだけ二列に切り替わる指定を持つ", () => {
      render(<CardHeader>見出し領域</CardHeader>);

      expect(screen.getByText("見出し領域")).toHaveClass(
        "has-data-[slot=card-action]:grid-cols-[1fr_auto]",
      );
    });

    it("native div 属性をそのまま渡す", () => {
      render(<CardHeader id="header">見出し領域</CardHeader>);

      expect(screen.getByText("見出し領域")).toHaveAttribute("id", "header");
    });
  });
});

describe("CardTitle", () => {
  describe("正常系", () => {
    it("見出しとして card-title の slot を名乗る", () => {
      render(<CardTitle>設定</CardTitle>);

      expect(screen.getByText("設定")).toHaveAttribute("data-slot", "card-title");
    });

    it("呼び出し元の className を足しても既定の強調を落とさない", () => {
      render(<CardTitle className="text-lg">設定</CardTitle>);

      expect(screen.getByText("設定")).toHaveClass("font-semibold", "text-lg");
    });

    it("文書構造上の見出しは持たず div のままにする", () => {
      render(<CardTitle>設定</CardTitle>);

      expect(screen.queryByRole("heading")).toBeNull();
    });
  });
});

describe("CardDescription", () => {
  describe("正常系", () => {
    it("補足文として card-description の slot を名乗る", () => {
      render(<CardDescription>確認してください</CardDescription>);

      expect(screen.getByText("確認してください")).toHaveAttribute(
        "data-slot",
        "card-description",
      );
    });

    it("見出しより弱い前景色で描く", () => {
      render(<CardDescription>確認してください</CardDescription>);

      expect(screen.getByText("確認してください")).toHaveClass("text-muted-foreground");
    });
  });
});

describe("CardAction", () => {
  describe("正常系", () => {
    it("補助操作として card-action の slot を名乗る", () => {
      render(<CardAction>変更</CardAction>);

      expect(screen.getByText("変更")).toHaveAttribute("data-slot", "card-action");
    });

    it("見出し群の右側へ寄せる配置を持つ", () => {
      render(<CardAction>変更</CardAction>);

      expect(screen.getByText("変更")).toHaveClass("col-start-2", "justify-self-end");
    });
  });
});

describe("CardContent", () => {
  describe("正常系", () => {
    it("主内容として card-content の slot を名乗る", () => {
      render(<CardContent>本文</CardContent>);

      expect(screen.getByText("本文")).toHaveAttribute("data-slot", "card-content");
    });

    it("呼び出し元の className を足しても既定の余白を落とさない", () => {
      render(<CardContent className="text-sm">本文</CardContent>);

      expect(screen.getByText("本文")).toHaveClass("px-6", "text-sm");
    });
  });
});

describe("CardFooter", () => {
  describe("正常系", () => {
    it("下部領域として card-footer の slot を名乗る", () => {
      render(<CardFooter>補助操作</CardFooter>);

      expect(screen.getByText("補助操作")).toHaveAttribute("data-slot", "card-footer");
    });

    it("上端に区切りを付けたときだけ余白を足す指定を持つ", () => {
      render(<CardFooter>補助操作</CardFooter>);

      expect(screen.getByText("補助操作")).toHaveClass("[.border-t]:pt-6");
    });
  });
});
