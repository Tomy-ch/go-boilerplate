"use client";

import * as DialogPrimitive from "@radix-ui/react-dialog";
import { XIcon } from "lucide-react";
import type { ComponentProps } from "react";

import { cn } from "@/components/cn";

/**
 * 内容の補助表示や通常の編集操作を、画面を覆う modal として開く client island root。
 *
 * @remarks
 * 開閉状態・focus trap・Escape・背面の inert 化を browser 側で行うため hydration が必要で、
 * Server Component からは直接 render できない。内容自体に client runtime が要らない場合は、
 * Server Component で組み立てた要素を `children` として渡す。
 *
 * 取り消せない操作の確認には使わない。その用途は `role="alertdialog"` の意味論を持つ別部品が要る。
 *
 * @param props - Radix `Dialog.Root` の props。`open` / `defaultOpen` / `onOpenChange` で
 *   開閉を制御でき、省略時は trigger と close の操作だけで開閉する。
 */
function Dialog({ ...props }: ComponentProps<typeof DialogPrimitive.Root>) {
  return <DialogPrimitive.Root data-slot="dialog" {...props} />;
}

/**
 * Dialog を開く trigger。
 *
 * @remarks
 * 既定では `button` を render する。`Button` や link を trigger にする場合は `asChild` を
 * 指定して単一の子要素へ合成する。
 *
 * @param props - Radix `Dialog.Trigger` の props。
 */
function DialogTrigger({ ...props }: ComponentProps<typeof DialogPrimitive.Trigger>) {
  return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />;
}

/**
 * dialog を document body 側へ描画する Portal。
 *
 * @remarks
 * `DialogContent` が内部で使うため、通常は直接指定しない。描画先の container を差し替える
 * 場合だけ使う。
 *
 * @param props - Radix `Dialog.Portal` の props。
 */
function DialogPortal({ ...props }: ComponentProps<typeof DialogPrimitive.Portal>) {
  return <DialogPrimitive.Portal data-slot="dialog-portal" {...props} />;
}

/**
 * dialog を閉じる操作。
 *
 * @remarks
 * footer の「キャンセル」や「閉じる」など、内容側に置く閉じる操作へ使う。`Button` を使う
 * 場合は `asChild` で合成する。右上の閉じる操作は `DialogContent` が既定で描画する。
 *
 * @param props - Radix `Dialog.Close` の props。
 */
function DialogClose({ ...props }: ComponentProps<typeof DialogPrimitive.Close>) {
  return <DialogPrimitive.Close data-slot="dialog-close" {...props} />;
}

/**
 * dialog の背面を覆う overlay。
 *
 * @remarks
 * `DialogContent` が内部で描画するため、通常は直接指定しない。
 *
 * @param props - Radix `Dialog.Overlay` の props。
 */
function DialogOverlay({ className, ...props }: ComponentProps<typeof DialogPrimitive.Overlay>) {
  return (
    <DialogPrimitive.Overlay
      className={cn(
        "fixed inset-0 z-50 bg-black/50 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:animate-in data-[state=open]:fade-in-0",
        className,
      )}
      data-slot="dialog-overlay"
      {...props}
    />
  );
}

/**
 * overlay・Portal とともに dialog 本体を描画する。
 *
 * @remarks
 * modal のため、開いている間は背面が操作不能になる。アクセシブルな名前として `DialogTitle` を
 * 必ず子に置く。説明が要る場合は `DialogDescription` を添え、不要な場合は
 * `aria-describedby={undefined}` を明示する。どちらも無いと Radix が警告する。
 *
 * 背景は `bg-background`、境界は `border-border` を使う。面が透明だと背後の内容と重なって
 * 可読性と contrast を失う。
 *
 * @param props - Radix `Dialog.Content` の props。`showCloseButton` を `false` にすると右上の
 *   閉じる操作を描画しない。閉じる手段を内容側で用意する場合だけ使う。
 */
function DialogContent({
  className,
  children,
  showCloseButton = true,
  ...props
}: ComponentProps<typeof DialogPrimitive.Content> & {
  showCloseButton?: boolean;
}) {
  return (
    <DialogPortal>
      <DialogOverlay />
      <DialogPrimitive.Content
        className={cn(
          "fixed top-1/2 left-1/2 z-50 grid w-full max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 gap-4 rounded-lg border border-border bg-background p-6 text-foreground shadow-lg duration-200 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95 sm:max-w-lg",
          className,
        )}
        data-slot="dialog-content"
        {...props}
      >
        {children}
        {showCloseButton ? (
          <DialogPrimitive.Close
            className="absolute top-4 right-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"
            data-slot="dialog-close"
          >
            <XIcon />
            <span className="sr-only">閉じる</span>
          </DialogPrimitive.Close>
        ) : null}
      </DialogPrimitive.Content>
    </DialogPortal>
  );
}

/**
 * title と説明をまとめる領域。
 *
 * @param props - native `div` 属性。
 */
function DialogHeader({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("flex flex-col gap-2 text-center sm:text-left", className)}
      data-slot="dialog-header"
      {...props}
    />
  );
}

/**
 * 操作を並べる footer 領域。
 *
 * @remarks
 * 配置だけを担い、操作そのものは持たない。閉じる操作は `DialogClose` を `Button` へ合成して
 * 呼び出し元が置く。
 *
 * @param props - native `div` 属性。
 */
function DialogFooter({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("flex flex-col-reverse gap-2 sm:flex-row sm:justify-end", className)}
      data-slot="dialog-footer"
      {...props}
    />
  );
}

/**
 * dialog のアクセシブルな名前を提供する title。
 *
 * @remarks
 * Radix が `aria-labelledby` を自動で結び付けるため、`id` を手で振る必要はない。
 *
 * @param props - Radix `Dialog.Title` の props。
 */
function DialogTitle({ className, ...props }: ComponentProps<typeof DialogPrimitive.Title>) {
  return (
    <DialogPrimitive.Title
      className={cn("text-lg leading-none font-semibold", className)}
      data-slot="dialog-title"
      {...props}
    />
  );
}

/**
 * dialog の目的や影響を説明する本文。
 *
 * @remarks
 * Radix が `aria-describedby` を自動で結び付ける。
 *
 * @param props - Radix `Dialog.Description` の props。
 */
function DialogDescription({
  className,
  ...props
}: ComponentProps<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      className={cn("text-sm text-muted-foreground", className)}
      data-slot="dialog-description"
      {...props}
    />
  );
}

export {
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
};
