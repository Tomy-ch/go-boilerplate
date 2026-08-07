// @vitest-environment jsdom

import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { SEARCH_FIELD_DEBOUNCE_MS, SearchFieldClient } from "./search-field-client";

const noop = () => undefined;

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

function type(value: string) {
  fireEvent.change(screen.getByRole("searchbox"), { target: { value } });
}

function advance(ms: number) {
  act(() => {
    vi.advanceTimersByTime(ms);
  });
}

describe("SearchFieldClient", () => {
  describe("正常系", () => {
    it("search landmark の中に type=search の検索入力を置く", () => {
      const { container } = render(<SearchFieldClient label="項目を検索" onSearch={noop} />);

      expect(container.querySelector("[data-slot='search-field']")?.tagName).toBe("SEARCH");
      expect(screen.getByRole("searchbox", { name: "項目を検索" })).toHaveAttribute("type", "search");
    });

    it("入力が止まってから検索語を通知する", () => {
      const onSearch = vi.fn();
      render(<SearchFieldClient label="項目を検索" onSearch={onSearch} />);

      type("標準");

      expect(onSearch).not.toHaveBeenCalled();

      advance(SEARCH_FIELD_DEBOUNCE_MS);

      expect(onSearch).toHaveBeenCalledExactlyOnceWith("標準");
    });

    it("入力が続く間は通知をまとめ、最後の値だけを通知する", () => {
      const onSearch = vi.fn();
      render(<SearchFieldClient label="項目を検索" onSearch={onSearch} />);

      type("標");
      advance(SEARCH_FIELD_DEBOUNCE_MS - 1);
      type("標準");
      advance(SEARCH_FIELD_DEBOUNCE_MS);

      expect(onSearch).toHaveBeenCalledExactlyOnceWith("標準");
    });

    it("待ち時間を呼び出し元が変えられる", () => {
      const onSearch = vi.fn();
      render(<SearchFieldClient debounceMs={1000} label="項目を検索" onSearch={onSearch} />);

      type("標準");
      advance(SEARCH_FIELD_DEBOUNCE_MS);

      expect(onSearch).not.toHaveBeenCalled();

      advance(1000 - SEARCH_FIELD_DEBOUNCE_MS);

      expect(onSearch).toHaveBeenCalledExactlyOnceWith("標準");
    });

    it("検索語があるときだけ消去ボタンを描画する", () => {
      render(<SearchFieldClient defaultValue="標準" label="項目を検索" onSearch={noop} />);

      expect(screen.getByRole("button", { name: "検索語を消去" })).toHaveAttribute("type", "button");
    });

    it("消去すると入力を空にし、空文字列を通知して入力へ focus を戻す", () => {
      const onSearch = vi.fn();
      render(<SearchFieldClient defaultValue="標準" label="項目を検索" onSearch={onSearch} />);

      fireEvent.click(screen.getByRole("button", { name: "検索語を消去" }));

      const input = screen.getByRole("searchbox", { name: "項目を検索" });

      expect(input).toHaveValue("");
      expect(input).toHaveFocus();

      advance(SEARCH_FIELD_DEBOUNCE_MS);

      expect(onSearch).toHaveBeenCalledExactlyOnceWith("");
    });

    it("消去ボタンのアクセシブルな名前を変えられる", () => {
      render(
        <SearchFieldClient
          clearLabel="入力を消す"
          defaultValue="標準"
          label="項目を検索"
          onSearch={noop}
        />,
      );

      expect(screen.getByRole("button", { name: "入力を消す" })).toBeInTheDocument();
    });

    it("a11y 自動検査に違反しない", async () => {
      vi.useRealTimers();

      const { container } = render(
        <SearchFieldClient
          defaultValue="標準"
          label="項目を検索"
          onSearch={noop}
          placeholder="キーワードを入力"
        />,
      );

      const result = await axe(container, { rules: { "color-contrast": { enabled: false } } });

      expect(result.violations).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("初期表示だけでは通知しない", () => {
      const onSearch = vi.fn();
      render(<SearchFieldClient defaultValue="標準" label="項目を検索" onSearch={onSearch} />);

      advance(SEARCH_FIELD_DEBOUNCE_MS);

      expect(onSearch).not.toHaveBeenCalled();
    });

    it("検索語が空のときは消去ボタンを描画しない", () => {
      render(<SearchFieldClient label="項目を検索" onSearch={noop} />);

      expect(screen.queryByRole("button", { name: "検索語を消去" })).not.toBeInTheDocument();
    });
  });
});
