// log.test.ts は認可フローの構造化イベントログの形を検証する。1 行 1 JSON であることと、
// event キーが fields に潰されないことが、ログを機械で読む側の唯一の前提になる。
import { afterEach, describe, expect, it, vi } from "vitest";
import { logEvent } from "./log.ts";

afterEach(() => {
  vi.restoreAllMocks();
});

// captureLog は console.log の呼び出し引数を記録する。
function captureLog(): string[] {
  const written: string[] = [];
  vi.spyOn(console, "log").mockImplementation((line: string) => {
    written.push(line);
  });
  return written;
}

describe("logEvent", () => {
  describe("正常系", () => {
    it("fields 省略時は event だけの 1 行を書く", () => {
      const written = captureLog();

      logEvent("token_issued");

      expect(written).toEqual([JSON.stringify({ event: "token_issued" })]);
    });

    it("fields を event と同じオブジェクトへ平坦に並べる", () => {
      const written = captureLog();

      logEvent("token_issued", { subject: "user-john-doe", client_id: "go-boilerplate-client" });

      expect(JSON.parse(written[0])).toEqual({
        event: "token_issued",
        subject: "user-john-doe",
        client_id: "go-boilerplate-client",
      });
    });

    it("1 回の呼び出しで 1 行だけ書く", () => {
      const written = captureLog();

      logEvent("a");
      logEvent("b");

      expect(written).toHaveLength(2);
    });
  });

  describe("異常系", () => {
    it("fields の event キーは後から与えた側が勝つ", () => {
      const written = captureLog();

      logEvent("declared", { event: "overridden" });

      expect(JSON.parse(written[0]).event).toBe("overridden");
    });
  });
});
