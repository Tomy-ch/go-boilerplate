// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";

import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
} from "./dialog";

function DialogFixture({
  defaultOpen = false,
  showCloseButton = true,
}: {
  defaultOpen?: boolean;
  showCloseButton?: boolean;
}) {
  return (
    <Dialog defaultOpen={defaultOpen}>
      <DialogTrigger>詳細を見る</DialogTrigger>
      <DialogContent showCloseButton={showCloseButton}>
        <DialogHeader>
          <DialogTitle>表示条件</DialogTitle>
          <DialogDescription>条件を満たす項目だけを一覧に表示します。</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose>戻る</DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

describe("Dialog", () => {
  describe("正常系", () => {
    it("trigger の操作で modal を開き、title と説明を関連付ける", () => {
      render(<DialogFixture />);

      fireEvent.click(screen.getByRole("button", { name: "詳細を見る" }));

      const content = screen.getByRole("dialog", { name: "表示条件" });

      expect(content).toHaveAttribute("data-slot", "dialog-content");
      expect(content).toHaveAccessibleDescription("条件を満たす項目だけを一覧に表示します。");
    });

    it("不可逆操作用ではないため alertdialog ではなく dialog の意味論を持つ", () => {
      render(<DialogFixture defaultOpen />);

      expect(screen.getByRole("dialog", { name: "表示条件" })).toBeInTheDocument();
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    });

    it("ページ内容へ重ねても背後が透けない不透明な面として描画する", () => {
      render(<DialogFixture defaultOpen />);

      expect(screen.getByRole("dialog", { name: "表示条件" })).toHaveClass(
        "bg-background",
        "text-foreground",
        "border-border",
      );
    });

    it("右上の閉じる操作は読み上げ可能な名前を持ち、押すと閉じる", () => {
      render(<DialogFixture defaultOpen />);

      fireEvent.click(screen.getByRole("button", { name: "閉じる" }));

      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("DialogClose で内容側からも閉じられる", () => {
      render(<DialogFixture defaultOpen />);

      fireEvent.click(screen.getByRole("button", { name: "戻る" }));

      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("Escape で閉じる", () => {
      render(<DialogFixture defaultOpen />);

      fireEvent.keyDown(document, { key: "Escape" });

      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("DialogPortal と DialogOverlay を直接指定して描画先と背面を差し替えられる", () => {
      render(
        <Dialog defaultOpen>
          <DialogTrigger>開く</DialogTrigger>
          <DialogPortal>
            <DialogOverlay data-testid="overlay" />
            <DialogContent showCloseButton={false}>
              <DialogTitle>差し替え</DialogTitle>
              <DialogDescription>背面と描画先を明示した場合の構成です。</DialogDescription>
            </DialogContent>
          </DialogPortal>
        </Dialog>,
      );

      expect(screen.getByTestId("overlay")).toHaveAttribute("data-slot", "dialog-overlay");
      expect(screen.getByRole("dialog", { name: "差し替え" })).toBeInTheDocument();
    });

    it("開いた状態で a11y 自動検査に違反しない", async () => {
      const { baseElement } = render(<DialogFixture defaultOpen />);

      const result = await axe(baseElement, { rules: { "color-contrast": { enabled: false } } });

      expect(result.violations).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("開くまで内容を表示しない", () => {
      render(<DialogFixture />);

      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("showCloseButton を false にすると右上の閉じる操作を描画しない", () => {
      render(<DialogFixture defaultOpen showCloseButton={false} />);

      expect(screen.queryByRole("button", { name: "閉じる" })).not.toBeInTheDocument();
      expect(screen.getByRole("dialog", { name: "表示条件" })).toBeInTheDocument();
    });
  });
});

/** 開いた状態の dialog を描画し、内容側の任意の子を差し込む。 */
function renderOpen(children: ReactNode) {
  return render(
    <Dialog defaultOpen>
      <DialogContent>
        <DialogTitle>表示条件</DialogTitle>
        <DialogDescription>条件を満たす項目だけを表示します。</DialogDescription>
        {children}
      </DialogContent>
    </Dialog>,
  );
}

describe("DialogPortal", () => {
  describe("正常系", () => {
    it("指定した描画先へ内容を移す", () => {
      const host = document.createElement("div");
      document.body.append(host);

      render(
        <Dialog defaultOpen>
          <DialogPortal container={host}>
            <div>移送された内容</div>
          </DialogPortal>
        </Dialog>,
      );

      expect(host.textContent).toContain("移送された内容");
    });
  });
});

describe("DialogOverlay", () => {
  describe("正常系", () => {
    it("背後が透けない不透明な面として dialog-overlay の slot を名乗る", () => {
      render(
        <Dialog defaultOpen>
          <DialogPortal>
            <DialogOverlay />
            <DialogContent>
              <DialogTitle>表示条件</DialogTitle>
              <DialogDescription>説明</DialogDescription>
            </DialogContent>
          </DialogPortal>
        </Dialog>,
      );

      const overlay = document.querySelector('[data-slot="dialog-overlay"]');

      expect(overlay).toHaveClass("fixed", "inset-0", "bg-black/50");
    });
  });
});

describe("DialogContent", () => {
  describe("正常系", () => {
    it("内容の器として dialog-content の slot を名乗る", () => {
      renderOpen(null);

      expect(screen.getByRole("dialog")).toHaveAttribute("data-slot", "dialog-content");
    });

    it("呼び出し元の className を足しても画面中央への配置を落とさない", () => {
      render(
        <Dialog defaultOpen>
          <DialogContent className="max-w-xl">
            <DialogTitle>表示条件</DialogTitle>
            <DialogDescription>説明</DialogDescription>
          </DialogContent>
        </Dialog>,
      );

      expect(screen.getByRole("dialog")).toHaveClass("fixed", "top-1/2", "left-1/2", "max-w-xl");
    });
  });
});

describe("DialogHeader", () => {
  describe("正常系", () => {
    it("見出し群の領域として dialog-header の slot を名乗る", () => {
      renderOpen(<DialogHeader>見出し領域</DialogHeader>);

      expect(screen.getByText("見出し領域")).toHaveAttribute("data-slot", "dialog-header");
    });

    it("狭い画面では中央、広い画面では左へ寄せる", () => {
      renderOpen(<DialogHeader>見出し領域</DialogHeader>);

      expect(screen.getByText("見出し領域")).toHaveClass("text-center", "sm:text-left");
    });
  });
});

describe("DialogFooter", () => {
  describe("正常系", () => {
    it("操作群の領域として dialog-footer の slot を名乗る", () => {
      renderOpen(<DialogFooter>操作領域</DialogFooter>);

      expect(screen.getByText("操作領域")).toHaveAttribute("data-slot", "dialog-footer");
    });

    it("狭い画面では主操作を最後に積み、広い画面では右へ寄せる", () => {
      renderOpen(<DialogFooter>操作領域</DialogFooter>);

      expect(screen.getByText("操作領域")).toHaveClass("flex-col-reverse", "sm:justify-end");
    });
  });
});

describe("DialogTitle", () => {
  describe("正常系", () => {
    it("dialog のアクセシブルな名前になる", () => {
      renderOpen(null);

      expect(screen.getByRole("dialog", { name: "表示条件" })).toBeInTheDocument();
    });

    it("見出しとして dialog-title の slot を名乗り強調して描く", () => {
      renderOpen(null);

      const title = screen.getByText("表示条件");

      expect(title).toHaveAttribute("data-slot", "dialog-title");
      expect(title).toHaveClass("font-semibold");
    });
  });
});

describe("DialogDescription", () => {
  describe("正常系", () => {
    it("補足文として dialog-description の slot を名乗る", () => {
      renderOpen(null);

      expect(screen.getByText("条件を満たす項目だけを表示します。")).toHaveAttribute(
        "data-slot",
        "dialog-description",
      );
    });

    it("見出しより弱い前景色で描く", () => {
      renderOpen(null);

      expect(screen.getByText("条件を満たす項目だけを表示します。")).toHaveClass(
        "text-muted-foreground",
      );
    });
  });
});

describe("DialogClose", () => {
  describe("正常系", () => {
    it("押すと dialog を閉じる", () => {
      renderOpen(<DialogClose>戻る</DialogClose>);

      fireEvent.click(screen.getByRole("button", { name: "戻る" }));

      expect(screen.queryByRole("dialog")).toBeNull();
    });

    it("閉じる操作として dialog-close の slot を名乗る", () => {
      renderOpen(<DialogClose>戻る</DialogClose>);

      expect(screen.getByRole("button", { name: "戻る" })).toHaveAttribute(
        "data-slot",
        "dialog-close",
      );
    });
  });
});

describe("DialogTrigger", () => {
  describe("正常系", () => {
    it("押すと dialog を開く", () => {
      render(
        <Dialog>
          <DialogTrigger>開く</DialogTrigger>
          <DialogContent>
            <DialogTitle>表示条件</DialogTitle>
            <DialogDescription>説明</DialogDescription>
          </DialogContent>
        </Dialog>,
      );

      fireEvent.click(screen.getByRole("button", { name: "開く" }));

      expect(screen.getByRole("dialog")).toBeVisible();
    });

    it("開く操作として dialog-trigger の slot を名乗る", () => {
      render(
        <Dialog>
          <DialogTrigger>開く</DialogTrigger>
        </Dialog>,
      );

      expect(screen.getByRole("button", { name: "開く" })).toHaveAttribute(
        "data-slot",
        "dialog-trigger",
      );
    });
  });
});
