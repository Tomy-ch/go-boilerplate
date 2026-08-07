import { afterEach, describe, expect, it, vi } from "vitest";

import { renderMermaid, resolveMermaidTheme } from "./render-mermaid";

// mermaid 本体は描画に browser の SVG 計測 API を必要とし、jsdom では動かない。ここで検査するのは
// 遅延読み込みと設定の与え方なので、mermaid そのものは呼び出しへ置き換える。
const initialize = vi.fn();
const render = vi.fn().mockResolvedValue({ svg: "<svg></svg>" });

vi.mock("mermaid", () => ({ default: { initialize, render } }));

function prefersDark(matches: boolean) {
  vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches }));
}

afterEach(() => {
  vi.unstubAllGlobals();
  delete document.documentElement.dataset.theme;
});

describe("resolveMermaidTheme", () => {
  describe("正常系", () => {
    it("data-theme の宣言を OS の設定より優先する", () => {
      expect(resolveMermaidTheme(true, "light")).toBe("default");
      expect(resolveMermaidTheme(false, "dark")).toBe("dark");
    });
    it("data-theme の宣言が無ければ OS の設定に従う", () => {
      expect(resolveMermaidTheme(true, null)).toBe("dark");
      expect(resolveMermaidTheme(false, null)).toBe("default");
    });
  });
});

describe("renderMermaid", () => {
  describe("正常系", () => {
    it("図の定義を SVG へ変換する", async () => {
      prefersDark(false);

      expect(await renderMermaid("mermaid1", "graph TD; A-->B;")).toBe("<svg></svg>");
      expect(render).toHaveBeenCalledWith("mermaid1", "graph TD; A-->B;");
    });
    it("図の定義に書かれた HTML と script を実行しない設定で初期化する", async () => {
      prefersDark(false);

      await renderMermaid("mermaid2", "graph TD;");

      expect(initialize).toHaveBeenLastCalledWith(
        expect.objectContaining({ startOnLoad: false, securityLevel: "strict" }),
      );
    });
    it("暗い配色の面では図も暗い配色で描く", async () => {
      prefersDark(true);

      await renderMermaid("mermaid3", "graph TD;");

      expect(initialize).toHaveBeenLastCalledWith(expect.objectContaining({ theme: "dark" }));
    });
    it("図ごとに設定を与え直す", async () => {
      prefersDark(false);
      await renderMermaid("mermaid4", "graph TD;");

      prefersDark(true);
      await renderMermaid("mermaid5", "graph TD;");

      expect(initialize).toHaveBeenLastCalledWith(expect.objectContaining({ theme: "dark" }));
    });
  });
});
