import type { ComponentProps } from "react";

import { cn } from "@/components/cn";

/** {@link Textarea} の props。 */
export type TextareaProps = ComponentProps<"textarea">;

/**
 * 複数行の値を受け取る、native `textarea` の見た目を統一する基礎部品。
 *
 * @remarks
 * `name`、`value`、`defaultValue`、`rows`、`required` などの native 属性はそのまま渡せる。
 * `<form action>` と組み合わせる場合は、送信する値を識別するため `name` を指定する。
 *
 * 入力項目の意味を利用者へ伝える `label`、説明文、検証エラーはこの部品の責務ではない。
 * form を構成するときは `Label` / `Field` か、同等の native 要素で補う。`aria-invalid` を
 * `true` にすると、入力内容が不正であることを視覚的にも表す。
 *
 * この部品は state や browser API を持たないため、Server Component と Client Component の
 * どちらからも利用できる。入力値を React state で制御するか、native form に委ねるかは
 * 呼び出し側が選ぶ。
 *
 * @example
 * ```tsx
 * <Textarea name="address-note" rows={4} />
 *
 * <Textarea aria-invalid={hasError} aria-describedby="note-error" name="note" />
 * ```
 *
 * @param props - native `textarea` 属性。`className` は既定の見た目へ追加・上書きできる。
 */
export function Textarea({ className, ...props }: TextareaProps) {
  return (
    <textarea
      className={cn(
        "field-sizing-content min-h-24 w-full resize-y rounded-md border border-border bg-background px-3 py-2 text-base text-foreground shadow-xs transition-colors placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 md:text-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground aria-invalid:border-destructive aria-invalid:outline-destructive",
        className,
      )}
      data-slot="textarea"
      {...props}
    />
  );
}
