import { ChevronDownIcon } from "lucide-react";
import type { ComponentProps } from "react";

import { cn } from "@/components/cn";

/**
 * 複数の {@link AccordionItem} を縦に並べる SSR first の外枠。
 */
export function Accordion({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("w-full divide-y divide-border rounded-md border border-border", className)}
      data-slot="accordion"
      {...props}
    />
  );
}

/**
 * 開閉できる一つの accordion 項目。
 *
 * @remarks
 * native の `details` を使うため、JavaScript なしで開閉できる。複数項目を同時に閉じる・開く
 * 表現を基本とし、常に一つだけを開く制御は client island が必要になった時点で追加する。
 */
export function AccordionItem({ className, ...props }: ComponentProps<"details">) {
  return (
    <details
      className={cn("group/accordion-item", className)}
      data-slot="accordion-item"
      {...props}
    />
  );
}

/**
 * {@link AccordionItem} を開閉する見出し。
 *
 * @remarks
 * `summary` は `details` の直接の子として置く。見出しレベルは呼び出し側で必要に応じて外側の
 * `h2` などと合成する。
 */
export function AccordionTrigger({ className, children, ...props }: ComponentProps<"summary">) {
  return (
    <summary
      className={cn(
        "flex cursor-pointer list-none items-center justify-between gap-4 px-4 py-3 text-left text-sm font-medium transition-colors hover:bg-foreground hover:text-background focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground [&::-webkit-details-marker]:hidden",
        className,
      )}
      data-slot="accordion-trigger"
      {...props}
    >
      {children}
      <ChevronDownIcon
        aria-hidden="true"
        className="size-4 shrink-0 text-muted-foreground transition-transform group-open/accordion-item:rotate-180"
      />
    </summary>
  );
}

/**
 * {@link AccordionItem} の開閉対象となる詳細内容。
 */
export function AccordionContent({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("border-t border-border px-4 py-3 text-sm text-muted-foreground", className)}
      data-slot="accordion-content"
      {...props}
    />
  );
}
