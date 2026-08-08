import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * 条件付き class 名を連結し、Tailwind の競合する utility を最後の指定へ正規化する。
 *
 * @remarks
 * `components` 内で class 名を条件分岐するときの唯一の入口。`clsx` や
 * `tailwind-merge` を利用側から直接 import しない。
 *
 * @example
 * ```ts
 * cn("px-2", isCompact && "px-1") // "px-1"
 * ```
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
