import type { VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/components/cn";

import { toggleVariants } from "../../action/toggle/toggle";

/** {@link ToggleGroupNative} の props。 */
export type ToggleGroupNativeProps = ComponentProps<"fieldset">;

/**
 * 関連する切り替えを 1 つの集合として扱う、SSR first の toggle group。
 *
 * @remarks
 * 項目は native の radio / checkbox なので、選択は form の値としてそのまま送信される。初期表示も
 * Server 側で確定し、client runtime を必要としない。URL にも form にも載せない即時の表示切替が
 * 必要な場合だけ、対になる `ToggleGroupClient` を使う。
 *
 * `fieldset` として公開されるため、`aria-label` か `aria-labelledby` で**何の切り替えかを必ず
 * 示す**。`legend` を置く場合はそちらが名前になる。
 *
 * 排他選択は項目を `type="radio"`、複数選択は `type="checkbox"` にし、同じ `name` を与える。
 * 選択肢どうしの移動は browser の標準動作に従う（radio は矢印キー、checkbox は Tab）。
 *
 * @param props - native `fieldset` 属性。
 */
export function ToggleGroupNative({ className, ...props }: ToggleGroupNativeProps) {
  return (
    <fieldset
      className={cn("flex w-fit items-center rounded-md", className)}
      data-slot="toggle-group-native"
      {...props}
    />
  );
}

/** {@link ToggleGroupNativeItem} の props。 */
export type ToggleGroupNativeItemProps = Omit<ComponentProps<"input">, "size" | "type"> &
  VariantProps<typeof toggleVariants> & {
    /** 排他選択なら `radio`、複数選択なら `checkbox`。 */
    type?: "radio" | "checkbox";
  };

/**
 * 集合の中の 1 項目。
 *
 * @remarks
 * 実体は `label` と、その中に視覚的に隠した native input である。input が focus と選択を担い、
 * 見た目は `label` が `has-[:checked]` / `has-[:focus-visible]` で追従する。隠しても支援技術からは
 * radio / checkbox として読み上げられ、keyboard でも到達できる。
 *
 * 隣接する項目は境界を重ねて 1 つの segmented control に見せるため、角丸は両端だけに付く。
 *
 * 選択中の面は `Toggle` と同じ `accent` で示す。`toggleVariants` を共有しているため、大きさと
 * variant の見た目も `Toggle` と揃う。
 *
 * `children` が項目のアクセシブルな名前になる。icon だけを置く場合は `aria-label` を添える。
 *
 * @param props - native `input` 属性と `variant` / `size`。`name` は集合内で共通にし、`value` は
 *   送信する値を表す。
 */
export function ToggleGroupNativeItem({
  children,
  className,
  size,
  type = "radio",
  variant,
  ...props
}: ToggleGroupNativeItemProps) {
  return (
    <label
      className={cn(
        toggleVariants({ size, variant }),
        "-ml-px cursor-pointer rounded-none px-3 first:ml-0 first:rounded-l-md last:rounded-r-md has-[:checked]:bg-accent has-[:checked]:text-accent-foreground has-[:disabled]:pointer-events-none has-[:disabled]:opacity-50 has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-foreground has-[:focus-visible]:outline-offset-2",
        className,
      )}
      data-slot="toggle-group-native-item"
    >
      <input className="sr-only" type={type} {...props} />
      {children}
    </label>
  );
}
