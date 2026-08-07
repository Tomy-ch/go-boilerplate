import type { ComponentProps } from "react";

import { cn } from "@/components/cn";

/**
 * 関連する情報と操作をひとまとまりに見せる外枠。
 *
 * @remarks
 * `Card` は見た目だけを提供する `div` であり、特定の業務型を受け取らない。
 * 必要な内容を `CardHeader`、`CardContent`、`CardFooter` として合成する。カード全体を
 * 遷移先にする場合は、内部に単一の link を置くのではなく、操作対象の範囲と見出しの link を
 * feature 側で明示する。
 *
 * @param props - native `div` 属性。`className` で幅や配置だけを追加できる。
 */
export function Card({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex flex-col gap-6 rounded-lg border border-border bg-card py-6 text-card-foreground",
        className,
      )}
      data-slot="card"
      {...props}
    />
  );
}

/**
 * カードの見出し・説明・補助操作を配置する領域。
 *
 * @remarks
 * 通常は `CardTitle` と `CardDescription` を子に置く。`CardAction` があるときは、見出し群と
 * 操作を二列に配置する。下端に区切りを付ける場合は `className="border-b border-border"` を渡す。
 *
 * @param props - native `div` 属性。
 */
export function CardHeader({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "@container/card-header grid auto-rows-min grid-rows-[auto_auto] items-start gap-2 px-6 has-data-[slot=card-action]:grid-cols-[1fr_auto] [.border-b]:pb-6",
        className,
      )}
      data-slot="card-header"
      {...props}
    />
  );
}

/**
 * カード内の見出しを視覚的に強調する領域。
 *
 * @remarks
 * この部品は見た目のみを持つ `div` である。文書構造上の見出しが必要な場合は、呼び出し元で
 * heading 要素を子として渡すか、feature 側で適切な heading を配置する。
 *
 * @param props - native `div` 属性。
 */
export function CardTitle({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("leading-none font-semibold", className)}
      data-slot="card-title"
      {...props}
    />
  );
}

/**
 * 見出しを補足する短い説明文の領域。
 *
 * @param props - native `div` 属性。
 */
export function CardDescription({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("text-sm text-muted-foreground", className)}
      data-slot="card-description"
      {...props}
    />
  );
}

/**
 * 見出しの右側へ置く補助操作の領域。
 *
 * @remarks
 * `Button` や補助 link など、カード全体ではなく明確に独立した操作だけを置く。主操作を
 * カード全体への click に暗黙化しない。
 *
 * @param props - native `div` 属性。
 */
export function CardAction({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("col-start-2 row-span-2 row-start-1 self-start justify-self-end", className)}
      data-slot="card-action"
      {...props}
    />
  );
}

/**
 * カードの主内容を配置する領域。
 *
 * @param props - native `div` 属性。
 */
export function CardContent({ className, ...props }: ComponentProps<"div">) {
  return <div className={cn("px-6", className)} data-slot="card-content" {...props} />;
}

/**
 * カード下部の操作または補足情報を配置する領域。
 *
 * @remarks
 * 上端に区切りを付ける場合は `className="border-t border-border"` を渡す。操作が複数あるときの
 * 並びや間隔は feature 側で指定し、ここへ業務固有の variant を増やさない。
 *
 * @param props - native `div` 属性。
 */
export function CardFooter({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("flex items-center px-6 [.border-t]:pt-6", className)}
      data-slot="card-footer"
      {...props}
    />
  );
}
