import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/components/cn";
import {
  BUTTON_SIZE,
  BUTTON_VARIANT,
  type ButtonSize,
  type ButtonVariant,
} from "./button.definition";

/**
 * ボタンの見た目を `variant` と `size` から組み立てる class 名生成器。
 *
 * @remarks
 * {@link Button} を render できない場所で、ボタンと同じ見た目だけを借りるために公開している。
 * 通常は {@link Button} を使い、リンクをボタンの見た目にする場合も `asChild` を使う。
 *
 * @example
 * ```tsx
 * <a className={buttonVariants({ variant: BUTTON_VARIANT.OUTLINE })} href="/help">ヘルプ</a>
 * ```
 */
export const buttonVariants = cva(
  "inline-flex cursor-pointer items-center justify-center gap-2 rounded-md font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground disabled:pointer-events-none disabled:cursor-default disabled:opacity-50",
  {
    variants: {
      variant: {
        [BUTTON_VARIANT.DEFAULT]:
          "bg-foreground text-background hover:bg-foreground/85 active:bg-foreground/70",
        [BUTTON_VARIANT.OUTLINE]:
          "border border-border bg-background text-foreground hover:bg-foreground hover:text-background active:bg-foreground/80",
        [BUTTON_VARIANT.GHOST]:
          "bg-transparent text-foreground hover:bg-foreground hover:text-background active:bg-foreground/80",
      },
      size: {
        [BUTTON_SIZE.DEFAULT]: "h-10 px-4 py-2",
        [BUTTON_SIZE.SMALL]: "h-8 px-3 text-sm",
        [BUTTON_SIZE.LARGE]: "h-11 px-6",
      },
    },
    defaultVariants: {
      variant: BUTTON_VARIANT.DEFAULT,
      size: BUTTON_SIZE.DEFAULT,
    },
  },
);

/**
 * {@link Button} の props。
 *
 * @remarks
 * 通常時は native の {@link HTMLButtonElement} として振る舞う。フォーム内で送信しない
 * 操作には、意図しない submit を避けるため `type="button"` を明示する。
 *
 * `asChild` を指定したときは、単一の React 要素を子に渡す。子要素へ class 名・
 * event handler・アクセシビリティ属性を合成するため、文字列、複数要素、Fragment は渡せない。
 * リンクをボタンと同じ見た目にする用途には `asChild` と単一の link 要素を組み合わせる。
 * 遷移には native の `a` を使う。
 */
export type ButtonProps = ComponentProps<"button"> &
  Omit<VariantProps<typeof buttonVariants>, "size" | "variant"> & {
    /**
     * 操作の優先度を表す見た目。
     *
     * - `default`: 画面の主要操作
     * - `outline`: 主要操作に並ぶ副次操作
     * - `ghost`: 表示の主張を抑える補助操作
     */
    variant?: ButtonVariant;
    /**
     * ボタンの大きさ。
     *
     * - `sm`: 表や密度の高い領域
     * - `default`: 通常の画面操作
     * - `lg`: 単独で目立たせる主要操作
     */
    size?: ButtonSize;
    /** 子要素へボタンの見た目と props を合成するか。 */
    asChild?: boolean;
  };

/**
 * 一貫した操作状態と focus-visible 表現を提供する基礎ボタン。
 *
 * @remarks
 * 画面固有の配色を `variant` へ直接指定せず、variant の追加または design token の見直しで扱う
 * （各 variant の意味は {@link ButtonProps.variant}）。
 *
 * disabled 状態は native `disabled` 属性で指定する。`asChild` でリンクを使う場合、
 * HTML の link には `disabled` が存在しないため、遷移禁止の状態を表す別の UI を選ぶ。
 *
 * @example
 * ```tsx
 * <Button type="submit">保存する</Button>
 *
 * <Button asChild variant={BUTTON_VARIANT.OUTLINE}>
 *   <a href="/settings">設定へ進む</a>
 * </Button>
 * ```
 *
 * @param props - native `button` 属性と、以下の表示用 props。
 * @param props.variant - 操作の優先度に対応する見た目。
 * @param props.size - 操作を置く領域に対応する大きさ。
 * @param props.asChild - 子要素へボタンの見た目と props を合成するか。
 */
export function Button({ className, variant, size, asChild = false, ...props }: ButtonProps) {
  const Component = asChild ? Slot : "button";

  return <Component className={cn(buttonVariants({ variant, size, className }))} {...props} />;
}
