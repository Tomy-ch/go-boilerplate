import { describe, expect, it } from "vitest";

import type { Event } from "./events";
import {
  CORRECTIVE_MARKERS,
  INJECTED_MARKERS,
  DEFAULT_LIMIT,
  excerpt,
  isCorrective,
  isInjected,
  renderCandidateComment,
  selectCandidates,
  type Candidate,
} from "./candidates";

const prompt = (at: number, text: string): Event => ({ client: "claude", kind: "prompt", at, text });
const interrupt = (at: number): Event => ({ client: "claude", kind: "interrupt", at });
const failure = (at: number): Event => ({ client: "claude", kind: "tool_result", at, ok: false });
const success = (at: number): Event => ({ client: "claude", kind: "tool_result", at, ok: true });

describe("CORRECTIVE_MARKERS", () => {
  describe("正常系", () => {
    it("重複した語を持たない", () => {
      expect(new Set(CORRECTIVE_MARKERS).size).toBe(CORRECTIVE_MARKERS.length);
    });
  });
});

describe("DEFAULT_LIMIT", () => {
  describe("正常系", () => {
    it("1 窓ぶんとして読ませられる程度に抑えてある", () => {
      expect(DEFAULT_LIMIT).toBeGreaterThan(0);
      expect(DEFAULT_LIMIT).toBeLessThanOrEqual(20);
    });
  });
});

describe("INJECTED_MARKERS", () => {
  describe("正常系", () => {
    it("重複した目印を持たない", () => {
      expect(new Set(INJECTED_MARKERS).size).toBe(INJECTED_MARKERS.length);
    });
  });
});

describe("isInjected", () => {
  describe("正常系", () => {
    it("task 通知を注入と判定する", () => {
      expect(isInjected("<task-notification> 完了しました")).toBe(true);
    });

    it("再開時の要約を注入と判定する", () => {
      expect(isInjected("This session is being continued from a previous conversation")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("人の発話は注入と判定しない", () => {
      expect(isInjected("それは違う")).toBe(false);
    });
  });
});

describe("isCorrective", () => {
  describe("正常系", () => {
    it("是正の合図を含む発話を拾う", () => {
      expect(isCorrective("それは違う")).toBe(true);
      expect(isCorrective("A ではなく B")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("合図を含まない発話は拾わない", () => {
      expect(isCorrective("次へ進めて")).toBe(false);
    });

    it("空文字は拾わない", () => {
      expect(isCorrective("")).toBe(false);
    });

    it("注入された本文は是正の語を含んでいても拾わない", () => {
      expect(isCorrective("<task-notification> それは違う")).toBe(false);
    });
  });
});

describe("excerpt", () => {
  describe("正常系", () => {
    it("空白を潰して 1 行にする", () => {
      expect(excerpt("あ\n\nい  う", 100)).toBe("あ い う");
    });

    it("上限以内ならそのまま返す", () => {
      expect(excerpt("みじかい", 100)).toBe("みじかい");
    });
  });

  describe("異常系", () => {
    it("上限を超えたら切って印を付ける", () => {
      expect(excerpt("あいうえお", 3)).toBe("あいう…");
    });

    it("要約はしない（先頭がそのまま残る）", () => {
      expect(excerpt("結論から言うと駄目です", 4).startsWith("結論から")).toBe(true);
    });
  });
});

describe("selectCandidates", () => {
  describe("正常系", () => {
    it("是正した発話を選ぶ", () => {
      const got = selectCandidates([prompt(100, "それは違う")]);
      expect(got).toEqual([{ at: 100, reason: "corrective", text: "それは違う" }]);
    });

    it("中断の直前の発話を選ぶ", () => {
      const got = selectCandidates([prompt(100, "これをやって"), interrupt(200)]);
      expect(got[0]).toMatchObject({ at: 100, reason: "before-interrupt" });
    });

    it("失敗の直後の発話を選ぶ", () => {
      const got = selectCandidates([failure(100), prompt(200, "なおして")]);
      expect(got[0]).toMatchObject({ at: 200, reason: "after-failure" });
    });

    it("是正を中断や失敗より先に並べる", () => {
      const got = selectCandidates([failure(50), prompt(100, "なおして"), prompt(300, "それは違う")]);
      expect(got[0]?.reason).toBe("corrective");
    });

    it("時刻順でない入力でも並べ直して扱う", () => {
      const got = selectCandidates([interrupt(200), prompt(100, "これをやって")]);
      expect(got[0]).toMatchObject({ at: 100, reason: "before-interrupt" });
    });

    it("上限を超えたら切り捨てる", () => {
      const many = Array.from({ length: 30 }, (_, i) => prompt(i + 1, "それは違う"));
      expect(selectCandidates(many, 5)).toHaveLength(5);
    });
  });

  describe("異常系", () => {
    it("同じ発話を理由違いで二重に選ばない", () => {
      const got = selectCandidates([prompt(100, "それは違う"), interrupt(200)]);
      expect(got).toHaveLength(1);
      expect(got[0]?.reason).toBe("corrective");
    });

    it("成功した tool_result では選ばない", () => {
      expect(selectCandidates([success(100), prompt(200, "つづけて")])).toEqual([]);
    });

    it("中断より後にしか発話が無ければ選ばない", () => {
      expect(selectCandidates([interrupt(100), prompt(200, "つづけて")])).toEqual([]);
    });

    it("失敗より前にしか発話が無ければ選ばない", () => {
      expect(selectCandidates([prompt(100, "やって"), failure(200)])).toEqual([]);
    });

    it("本文を持たない発話は選ばない", () => {
      expect(selectCandidates([{ client: "claude", kind: "prompt", at: 100 }])).toEqual([]);
    });

    it("本文が空の発話は選ばない", () => {
      expect(selectCandidates([prompt(100, "")])).toEqual([]);
    });

    it("注入された本文は中断や失敗の文脈でも選ばない", () => {
      expect(selectCandidates([prompt(100, "<task-notification> done"), interrupt(200)])).toEqual([]);
    });

    it("イベントが無ければ空になる", () => {
      expect(selectCandidates([])).toEqual([]);
    });

    it("上限が 0 以下なら何も返さない", () => {
      expect(selectCandidates([prompt(100, "それは違う")], -1)).toEqual([]);
    });
  });
});

describe("renderCandidateComment", () => {
  const c = (o: Partial<Candidate> = {}): Candidate => ({ at: 100, reason: "corrective", text: "それは違う", ...o });

  describe("正常系", () => {
    it("理由と逐語を並べる", () => {
      const out = renderCandidateComment([c()]);
      expect(out).toContain("`corrective`");
      expect(out).toContain("> それは違う");
    });

    it("時刻を機械が読める形で残す", () => {
      expect(renderCandidateComment([c({ at: 1787 })])).toContain("<!-- at:1787 -->");
    });

    it("要約ではなく逐語であることを明示する", () => {
      expect(renderCandidateComment([c()])).toContain("逐語");
    });
  });

  describe("異常系", () => {
    it("候補が無いことを明示する", () => {
      expect(renderCandidateComment([])).toContain("該当したターンはありません");
    });
  });
});
