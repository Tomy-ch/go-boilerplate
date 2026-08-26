"use client";

import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps, MouseEvent } from "react";

import { cn } from "@/components/cn";

import { Button } from "../../action/button/button";
import { BUTTON_VARIANT } from "../../action/button/button.definition";
import { Input } from "../input/input";
import { Textarea } from "../textarea/textarea";
import {
  INPUT_GROUP_ADDON_ALIGN,
  INPUT_GROUP_BUTTON_SIZE,
  type InputGroupAddonAlign,
  type InputGroupButtonSize,
} from "./input-group.definition";

/** {@link InputGroup} の props。 */
export type InputGroupProps = ComponentProps<"div">;

/**
 * 一つの入力欄と、その前後に置く addon を一続きの枠として見せる外枠。
 *
 * @remarks
 * 直接の子には `InputGroupInput` または `InputGroupTextarea` を一つだけ置き、その前後へ
 * `InputGroupAddon` を並べる。枠線・角丸・focus 表示・invalid 表示・disabled 表示は外枠が
 * 描画し、内側の control は枠を持たない。addon の `align` に応じて、外枠は横並びと縦積みを
 * 切り替える。
 *
 * 枠線は control の `disabled` から自動で控えめな色へ落ちる。addon も同時に減光する場合は、
 * 外枠へ `data-disabled="true"` を渡す。
 *
 * addon を押したときに control へ focus を移すため hydration が必要で、Server Component から
 * 直接 render できない。単位や補助操作を枠内へ収める必要がない場合は、`Input` と `Label` /
 * `Field` を組み合わせた Server Component 側の構成を選ぶ。
 *
 * @example
 * ```tsx
 * <InputGroup>
 *   <InputGroupInput aria-label="数量" inputMode="numeric" name="quantity" />
 *   <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.INLINE_END}>
 *     <InputGroupText>kg</InputGroupText>
 *   </InputGroupAddon>
 * </InputGroup>
 * ```
 *
 * @param props - native `div` 属性。`role="group"` は既定で付与する。
 */
export function InputGroup({ className, ...props }: InputGroupProps) {
  return (
    // biome-ignore lint/a11y/useSemanticElements: fieldset は legend を伴う複数 control の集合を表す。ここは一つの control と装飾の外枠であり、意味論は role="group" が正しい
    <div
      className={cn(
        "group/input-group relative flex w-full items-center rounded-md border border-input shadow-xs transition-[color,box-shadow] dark:bg-input/30",
        "h-10 min-w-0 has-[>textarea]:h-auto",

        // Variants based on alignment.
        "has-[>[data-align=inline-start]]:[&>input]:pl-2",
        "has-[>[data-align=inline-end]]:[&>input]:pr-2",
        "has-[>[data-align=block-start]]:h-auto has-[>[data-align=block-start]]:flex-col has-[>[data-align=block-start]]:[&>input]:pb-3",
        "has-[>[data-align=block-end]]:h-auto has-[>[data-align=block-end]]:flex-col has-[>[data-align=block-end]]:[&>input]:pt-3",

        // Focus state.
        "has-[[data-slot=input-group-control]:focus-visible]:outline-2 has-[[data-slot=input-group-control]:focus-visible]:outline-offset-2 has-[[data-slot=input-group-control]:focus-visible]:outline-foreground",

        // Error state.
        "has-[[data-slot][aria-invalid=true]]:border-destructive has-[[data-slot][aria-invalid=true]]:ring-destructive/20 dark:has-[[data-slot][aria-invalid=true]]:ring-destructive/40",

        // Disabled state.
        "has-[[data-slot=input-group-control]:disabled]:border-border data-[disabled=true]:border-border",

        className,
      )}
      data-slot="input-group"
      role="group"
      {...props}
    />
  );
}

const inputGroupAddonVariants = cva(
  "flex h-auto cursor-text items-center justify-center gap-2 py-1.5 text-sm font-medium text-muted-foreground select-none group-data-[disabled=true]/input-group:opacity-50 [&>kbd]:rounded-sm [&>svg:not([class*='size-'])]:size-4",
  {
    variants: {
      align: {
        [INPUT_GROUP_ADDON_ALIGN.INLINE_START]:
          "order-first pl-3 has-[>button]:ml-[-0.45rem] has-[>kbd]:ml-[-0.35rem]",
        [INPUT_GROUP_ADDON_ALIGN.INLINE_END]:
          "order-last pr-3 has-[>button]:mr-[-0.45rem] has-[>kbd]:mr-[-0.35rem]",
        [INPUT_GROUP_ADDON_ALIGN.BLOCK_START]:
          "order-first w-full justify-start border-border border-b px-3 pt-3 pb-3 group-has-[>input]/input-group:pt-2.5",
        [INPUT_GROUP_ADDON_ALIGN.BLOCK_END]:
          "order-last w-full justify-start border-border border-t px-3 pt-3 pb-3 group-has-[>input]/input-group:pb-2.5",
      },
    },
    defaultVariants: {
      align: INPUT_GROUP_ADDON_ALIGN.INLINE_START,
    },
  },
);

/** addon の空白部分を押したとき、同じ枠内の control へ focus を移す。 */
function forwardFocusToControl(event: MouseEvent<HTMLDivElement>): void {
  if (event.target instanceof HTMLElement && event.target.closest("button")) {
    return;
  }

  const group = event.currentTarget.closest("[data-slot='input-group']");
  group?.querySelector<HTMLElement>("[data-slot='input-group-control']")?.focus();
}

/** {@link InputGroupAddon} の props。 */
export type InputGroupAddonProps = ComponentProps<"div"> &
  Omit<VariantProps<typeof inputGroupAddonVariants>, "align"> & {
    /**
     * 入力欄に対して addon を置く位置。
     *
     * - `inline-start`: 単位記号や検索アイコンなど、入力欄の前に置く
     * - `inline-end`: 送信・消去など、入力欄の後ろに置く
     * - `block-start`: 見出しや操作列として、入力欄の上に積む
     * - `block-end`: 補足や操作列として、入力欄の下に積む
     */
    align?: InputGroupAddonAlign;
  };

/**
 * 入力欄の枠内に、記号・アイコン・補助操作を並べる領域。
 *
 * @remarks
 * `align` が `block-start` / `block-end` のとき、外枠は縦積みになり addon は横幅いっぱいに
 * 広がる。この向きの addon には区切り線を既定で引く。入力欄の上下へ積まれた別の行であり、線が
 * 無いと入力領域と地続きに見え、置いた文字列が placeholder と区別できなくなる。inline 方向は
 * 入力欄と同じ行に収まるため引かない。addon の空白部分を押すと、同じ枠内の control へ focus が移る。addon 内の button を
 * 押した場合は button の操作を優先し、focus は移さない。
 *
 * @remarks
 * 表示のための領域であり、それ自体は focus を受け取らない。keyboard 利用者は control へ直接
 * tab で到達する。addon へ操作を置く場合は `InputGroupButton` を使い、その button 自身に
 * アクセシブルな名前を与える。
 *
 * @param props - native `div` 属性と、以下の表示用 props。
 * @param props.align - 入力欄に対して addon を置く位置。
 */
export function InputGroupAddon({
  className,
  align = INPUT_GROUP_ADDON_ALIGN.INLINE_START,
  ...props
}: InputGroupAddonProps) {
  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: control へ focus を委譲する pointer 専用の補助であり、keyboard では control へ直接 tab で到達する
    // biome-ignore lint/a11y/useSemanticElements: fieldset は legend を伴う複数 control の集合を表す。ここは control に付随する装飾領域であり、意味論は role="group" が正しい
    <div
      className={cn(inputGroupAddonVariants({ align }), className)}
      data-align={align}
      data-slot="input-group-addon"
      onClick={forwardFocusToControl}
      role="group"
      {...props}
    />
  );
}

const inputGroupButtonVariants = cva("flex items-center gap-2 text-sm shadow-none", {
  variants: {
    size: {
      [INPUT_GROUP_BUTTON_SIZE.EXTRA_SMALL]:
        "h-6 gap-1 rounded-sm px-2 has-[>svg]:px-2 [&>svg:not([class*='size-'])]:size-3.5",
      [INPUT_GROUP_BUTTON_SIZE.SMALL]: "h-8 gap-1.5 rounded-md px-2.5 has-[>svg]:px-2.5",
      [INPUT_GROUP_BUTTON_SIZE.ICON_EXTRA_SMALL]: "size-6 rounded-sm p-0 has-[>svg]:p-0",
      [INPUT_GROUP_BUTTON_SIZE.ICON_SMALL]: "size-8 p-0 has-[>svg]:p-0",
    },
  },
  defaultVariants: {
    size: INPUT_GROUP_BUTTON_SIZE.EXTRA_SMALL,
  },
});

/** {@link InputGroupButton} の props。 */
export type InputGroupButtonProps = Omit<ComponentProps<typeof Button>, "size"> &
  Omit<VariantProps<typeof inputGroupButtonVariants>, "size"> & {
    /**
     * 入力欄の枠内に収める button の大きさ。
     *
     * - `xs` / `sm`: 文言を伴う操作
     * - `icon-xs` / `icon-sm`: アイコンだけの正方形の操作
     */
    size?: InputGroupButtonSize;
  };

/**
 * 入力欄の枠内へ収まる寸法に調整した補助操作ボタン。
 *
 * @remarks
 * `Button` の見た目を引き継ぎつつ、枠内へ収まる高さと角丸へ差し替える。既定は `type="button"`
 * で、form の中に置いても送信しない。検索実行など submit させたい場合だけ `type` を明示する。
 * `variant` は `Button` と同じ値を受け取り、既定は周囲の情報量を増やさない `ghost`。
 *
 * @remarks
 * アイコンだけを子に置く場合、button のアクセシブルな名前が空になる。`aria-label` か視覚的に
 * 隠したテキストを必ず与える。
 *
 * @param props - `Button` の props（`size` を除く）と、以下の表示用 props。
 * @param props.size - 入力欄の枠内に収める button の大きさ。
 */
export function InputGroupButton({
  className,
  type = "button",
  variant = BUTTON_VARIANT.GHOST,
  size = INPUT_GROUP_BUTTON_SIZE.EXTRA_SMALL,
  ...props
}: InputGroupButtonProps) {
  return (
    <Button
      className={cn(inputGroupButtonVariants({ size }), className)}
      data-size={size}
      type={type}
      variant={variant}
      {...props}
    />
  );
}

/** {@link InputGroupText} の props。 */
export type InputGroupTextProps = ComponentProps<"span">;

/**
 * addon 内へ単位・記号・短い説明を置く文字列。
 *
 * @remarks
 * 入力欄そのものの意味を伝えるものではない。control のアクセシブルな名前は `Label` /
 * `Field` か `aria-label` で別に与える。子にアイコンを置いた場合は既定の大きさへ揃える。
 *
 * @param props - native `span` 属性。
 */
export function InputGroupText({ className, ...props }: InputGroupTextProps) {
  return (
    <span
      className={cn(
        "flex items-center gap-2 text-sm text-muted-foreground [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4",
        className,
      )}
      {...props}
    />
  );
}

/** {@link InputGroupInput} の props。 */
export type InputGroupInputProps = ComponentProps<"input">;

/**
 * 枠を外枠へ委ねた、`InputGroup` 内で使う単一行の入力欄。
 *
 * @remarks
 * `Input` から枠線・背景・focus ring を外し、外枠がまとめて描画する。focus 表示は `outline` で
 * 描かれるため、打ち消しも `outline` 側で行う。`ring-0` では消えず、外枠の輪と二重になる。`type`、`name`、
 * `value`、`defaultValue`、`required`、`aria-invalid` などの native 属性はそのまま渡せる。
 * `aria-invalid` を `true` にすると、外枠の枠線も invalid の表示へ変わる。
 *
 * @remarks
 * 項目名は `Label` / `Field` の `htmlFor` か `aria-label` で必ず与える。addon に置いた記号や
 * アイコンはアクセシブルな名前にならない。
 *
 * @param props - native `input` 属性。
 */
export function InputGroupInput({ className, ...props }: InputGroupInputProps) {
  return (
    <Input
      className={cn(
        "flex-1 rounded-none border-0 bg-transparent shadow-none focus-visible:outline-hidden dark:bg-transparent",
        className,
      )}
      data-slot="input-group-control"
      {...props}
    />
  );
}

/** {@link InputGroupTextarea} の props。 */
export type InputGroupTextareaProps = ComponentProps<"textarea">;

/**
 * 枠を外枠へ委ねた、`InputGroup` 内で使う複数行の入力欄。
 *
 * @remarks
 * `Textarea` から枠線・背景・focus ring を外し、外枠がまとめて描画する。高さは入力内容に
 * 応じて伸び、外枠もそれに追随する。`rows` は内容に追随しない環境での初期行数として効く。
 * 利用者が枠の大きさを変えられないよう resize は無効にしてある。
 *
 * 内容が増えると外枠が縦に伸びるため、table の cell のように行の高さが揃っている場所へ置くと
 * 周囲が押し下げられる。高さを固定したい場所では単一行の `InputGroupInput` を選ぶ。
 *
 * @remarks
 * 項目名は `Label` / `Field` の `htmlFor` か `aria-label` で必ず与える。
 *
 * @param props - native `textarea` 属性。
 */
export function InputGroupTextarea({ className, ...props }: InputGroupTextareaProps) {
  return (
    <Textarea
      className={cn(
        "flex-1 resize-none rounded-none border-0 bg-transparent py-3 shadow-none focus-visible:outline-hidden dark:bg-transparent",
        className,
      )}
      data-slot="input-group-control"
      {...props}
    />
  );
}
