import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/components/cn";

/**
 * {@link Toggle} の見た目の variant。同じ見た目を要する部品から再利用する。
 *
 * @remarks
 * 選択中の面は `aria-pressed` と `data-[state=on]` の両方で指定している。単独の `Toggle` は
 * `aria-pressed` を持つが、`ToggleGroupClient` の項目は `type="single"` のとき `aria-checked` に
 * なり `aria-pressed` を持たないため、両モードで共通する `data-state` も見る必要がある。
 */
export const toggleVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium text-sm transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-2 focus-visible:outline-foreground focus-visible:outline-offset-2 disabled:pointer-events-none disabled:opacity-50 aria-pressed:bg-accent aria-pressed:text-accent-foreground data-[state=on]:bg-accent data-[state=on]:text-accent-foreground [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-transparent",
        outline: "border border-input bg-transparent shadow-xs",
      },
      size: {
        default: "h-9 min-w-9 px-2",
        sm: "h-8 min-w-8 px-1.5",
        lg: "h-10 min-w-10 px-2.5",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

/** {@link Toggle} の props。 */
export type ToggleProps = ComponentProps<"button"> &
  VariantProps<typeof toggleVariants> & {
    /** 押下状態。 */
    pressed: boolean;
  };

/**
 * 押下状態を持つ単独の切り替えボタン。
 *
 * @remarks
 * 状態は `pressed` として受け取るだけで、この component は state を持たない。そのため
 * `"use client"` を必要とせず、**Server Component からも Client Component からも同じように
 * 使える**。URL や form で切り替える場合は Server 側で `pressed` を決め、browser 側の一時的な
 * state で切り替える場合は呼び出し元の client island が `useState` の値を渡す。
 *
 * `aria-pressed` を持つ button として公開される。同じ二値でも、form の値として送るものは
 * `CheckboxNative`、設定の入り切りは `SwitchNative` / `SwitchClient` を使う。この component は
 * 「今その表示が適用されているか」を示す**操作ボタン**であり、送信値を持たない。
 *
 * 既定の `type` は `"button"` である。URL に載せて切り替える場合は、呼び出し元が `type="submit"`
 * と `name` / `value` を与えて native form へ載せるか、`Link` で置き換える。
 *
 * icon だけを置く場合は、`aria-label` でアクセシブルな名前を与える。押下状態は `aria-pressed` が
 * 伝えるため、名前を「〜を有効にする」「〜を無効にする」と状態で切り替えない。名前は変えずに
 * 状態だけを変える。
 */
export function Toggle({
  className,
  pressed,
  size,
  type = "button",
  variant,
  ...props
}: ToggleProps) {
  return (
    <button
      aria-pressed={pressed}
      className={cn(toggleVariants({ className, size, variant }))}
      data-slot="toggle"
      type={type}
      {...props}
    />
  );
}
