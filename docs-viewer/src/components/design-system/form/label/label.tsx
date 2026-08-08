import type { ComponentProps } from "react";

import { cn } from "@/components/cn";

/** {@link Label} の props。 */
export type LabelProps = ComponentProps<"label">;

/**
 * form control の項目名を利用者へ伝えるためのラベル部品。
 *
 * @remarks
 * `htmlFor` には対象 control の `id` を指定する。対応する control をクリック可能にし、
 * スクリーンリーダーが項目名として認識できるようにする。`id` は `useId` などで一意に生成する。
 *
 * 説明文や検証エラー、必須表示などの field 全体のレイアウトは持たない。複数の要素をまとめる
 * 必要がある場合は `Field`、または feature 側の form 構成で組み合わせる。
 *
 * @example
 * ```tsx
 * const inputId = useId();
 *
 * <Label htmlFor={inputId}>表示名</Label>
 * <Input id={inputId} name="displayName" />
 * ```
 *
 * この部品は native 要素だけで構成するため、Server Component からそのまま利用できる。
 *
 * @param props - native `label` 属性。`className` は既定の見た目へ追加・上書きできる。
 */
export function Label({ children, className, htmlFor, ...props }: LabelProps) {
  return (
    <label
      data-slot="label"
      className={cn(
        "flex items-center gap-2 text-sm leading-none font-medium select-none group-data-[disabled=true]:pointer-events-none group-data-[disabled=true]:opacity-50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50",
        className,
      )}
      htmlFor={htmlFor}
      {...props}
    >
      {children}
    </label>
  );
}
