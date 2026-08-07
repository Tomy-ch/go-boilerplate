import type { ComponentProps } from "react";

import { cn } from "@/components/cn";

/** {@link Input} の props。 */
export type InputProps = ComponentProps<"input">;

/**
 * 単一行の値を受け取る、native `input` の見た目を統一する基礎部品。
 *
 * @remarks
 * `type`、`name`、`value`、`defaultValue`、`required` などの native 属性はそのまま渡せる。
 * `<form action>` と組み合わせる場合は、送信する値を識別するため `name` を指定する。
 *
 * 入力項目の意味を利用者へ伝える `label`、説明文、検証エラーはこの部品の責務ではない。
 * form を構成するときは、後で追加する `Label` / `Field` か、同等の native 要素で補う。
 * `aria-invalid` を `true` にすると、入力内容が不正であることを視覚的にも表す。
 *
 * この部品は state や browser API を持たないため、Server Component と Client Component の
 * どちらからも利用できる。入力値を React state で制御するか、native form に委ねるかは
 * 呼び出し側が選ぶ。
 *
 * @example
 * ```tsx
 * <Input name="email" type="email" autoComplete="email" required />
 *
 * <Input
 *   aria-invalid={hasError}
 *   aria-describedby="email-error"
 *   defaultValue="user@example.com"
 *   name="email"
 *   type="email"
 * />
 * ```
 *
 * @param props - native `input` 属性。`className` は既定の見た目へ追加・上書きできる。
 */
export function Input({ className, type, ...props }: InputProps) {
  return (
    <input
      className={cn(
        "h-10 w-full min-w-0 rounded-md border border-border bg-background px-3 py-2 text-base text-foreground shadow-xs transition-colors placeholder:text-muted-foreground selection:bg-foreground selection:text-background file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground aria-invalid:border-destructive aria-invalid:outline-destructive",
        className,
      )}
      data-slot="input"
      type={type}
      {...props}
    />
  );
}
