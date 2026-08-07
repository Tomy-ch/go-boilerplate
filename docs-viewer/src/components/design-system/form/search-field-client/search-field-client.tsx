"use client";

import { SearchIcon, XIcon } from "lucide-react";
import { type ChangeEvent, useCallback, useEffect, useRef, useState } from "react";

import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "../input-group/input-group";
import {
  INPUT_GROUP_ADDON_ALIGN,
  INPUT_GROUP_BUTTON_SIZE,
} from "../input-group/input-group.definition";

/**
 * 入力が止まってから {@link SearchFieldClient} が通知するまでの既定の待ち時間（ミリ秒）。
 */
export const SEARCH_FIELD_DEBOUNCE_MS = 300;

/** {@link SearchFieldClient} の props。 */
export type SearchFieldClientProps = {
  /**
   * 検索入力のアクセシブルな名前。
   *
   * 視覚的なラベルは持たないため、何を検索する欄なのかはこの値だけが伝える。`placeholder` は
   * 入力を始めると消えるため、名前の代わりにはならない。
   */
  label: string;
  /**
   * 入力が止まったときに呼ばれる。
   *
   * 引数は現在の検索語で、消去された場合は空文字列になる。URL の更新、取得、遷移はここで
   * 呼び出し元が行う。
   */
  onSearch: (value: string) => void;
  /** 初期表示する検索語。現在の検索条件を反映する場合に、呼び出し元が `searchParams` から渡す。 */
  defaultValue?: string;
  /** 入力例を示す補助文。 */
  placeholder?: string;
  /** 入力が止まってから `onSearch` を呼ぶまでの待ち時間（ミリ秒）。 */
  debounceMs?: number;
  /** 消去ボタンのアクセシブルな名前。 */
  clearLabel?: string;
  /** 追加の class。 */
  className?: string;
};

/**
 * 打鍵に追従してキーワード検索を通知する client island。
 *
 * @remarks
 * 入力の保持と待ち時間の制御を browser 側で行うため hydration が必要で、Server Component からは
 * 直接 render できない。JavaScript が無くても送信できる形が要る場合や、検索が主導線でない場合は
 * `SearchFieldNative` を選ぶ。
 *
 * 検索の実行、結果の取得、URL の組み立ては持たない。入力が止まると `onSearch` を呼ぶだけで、
 * router の操作も行わない。この分担は `Pagination` と同じで、`components` は URL を解釈しない。
 * 呼び出し元は受け取った検索語を `searchParams` へ載せ、結果は Server Component で描画する。
 * 結果まで client 側で取得すると、共有・履歴・戻る操作が URL と一致しなくなる。
 *
 * `onSearch` は入力が止まってから呼ばれる。参照が変わるたびに待ち時間が測り直されるため、
 * 呼び出し元は `useCallback` などで安定した関数を渡す。
 *
 * HTML の `search` 要素で囲むため、支援技術の landmark 一覧から到達できる。同じ画面に検索欄を
 * 複数置く場合は、`aria-label` で landmark を区別する。
 *
 * @example
 * ```tsx
 * const handleSearch = useCallback(
 *   (value: string) => router.replace(value ? `?q=${encodeURIComponent(value)}` : "?"),
 *   [router],
 * );
 *
 * <SearchFieldClient label="項目を検索" onSearch={handleSearch} />
 * ```
 */
export function SearchFieldClient({
  className,
  clearLabel = "検索語を消去",
  debounceMs = SEARCH_FIELD_DEBOUNCE_MS,
  defaultValue = "",
  label,
  onSearch,
  placeholder,
}: SearchFieldClientProps) {
  const [value, setValue] = useState(defaultValue);
  const inputRef = useRef<HTMLInputElement>(null);
  const isFirstRender = useRef(true);

  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }

    const timer = setTimeout(() => onSearch(value), debounceMs);

    return () => clearTimeout(timer);
  }, [debounceMs, onSearch, value]);

  const clear = useCallback(() => {
    setValue("");
    inputRef.current?.focus();
  }, []);

  const handleChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => setValue(event.target.value),
    [],
  );

  return (
    <search className={className} data-slot="search-field">
      <InputGroup>
        <InputGroupAddon>
          <SearchIcon aria-hidden="true" />
        </InputGroupAddon>
        <InputGroupInput
          aria-label={label}
          data-slot="search-field-input"
          onChange={handleChange}
          placeholder={placeholder}
          ref={inputRef}
          type="search"
          value={value}
        />
        {value ? (
          <InputGroupAddon align={INPUT_GROUP_ADDON_ALIGN.INLINE_END}>
            <InputGroupButton
              aria-label={clearLabel}
              data-slot="search-field-clear"
              onClick={clear}
              size={INPUT_GROUP_BUTTON_SIZE.ICON_EXTRA_SMALL}
              type="button"
            >
              <XIcon aria-hidden="true" />
            </InputGroupButton>
          </InputGroupAddon>
        ) : null}
      </InputGroup>
    </search>
  );
}
