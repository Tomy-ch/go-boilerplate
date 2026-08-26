import { describe, expect, it } from "vitest";
import { cn } from "./cn";

describe("cn", () => {
  describe("正常系", () => {
    it("複数の class 名を空白で連結する", () => {
      expect(cn("rounded", "border")).toBe("rounded border");
    });

    it("競合する Tailwind utility を最後の指定へ正規化する", () => {
      expect(cn("px-2", "px-1")).toBe("px-1");
    });

    it("競合しない utility は両方とも残す", () => {
      expect(cn("px-2", "py-1")).toBe("px-2 py-1");
    });

    it("条件が真のときだけ class 名を採る", () => {
      expect(cn("px-2", true && "px-1")).toBe("px-1");
    });

    it("配列で渡した class 名を平坦にする", () => {
      expect(cn(["rounded", ["border"]])).toBe("rounded border");
    });

    it("オブジェクト記法は値が真のキーだけを採る", () => {
      expect(cn({ rounded: true, border: false })).toBe("rounded");
    });
  });

  describe("異常系", () => {
    it("引数が無ければ空文字を返す", () => {
      expect(cn()).toBe("");
    });

    it("偽値だけを渡されたら空文字を返す", () => {
      expect(cn(false, null, undefined, "")).toBe("");
    });
  });
});
