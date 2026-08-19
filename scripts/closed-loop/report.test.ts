import { describe, expect, it } from "vitest";

import type { SessionFacts } from "./events";
import {
  DAY_BOUNDARY_OFFSET_SEC,
  overlapsPeriod,
  resolvePeriod,
  summarizePeriod,
  uncalledSkills,
  type Period,
} from "./report";

const DAY = 86_400;
const day = (s: string) => Math.floor(Date.parse(`${s}T00:00:00Z`) / 1000) - DAY_BOUNDARY_OFFSET_SEC;

const facts = (o: Partial<SessionFacts>): SessionFacts => ({
  client: "claude",
  branches: [],
  prompts: 0,
  toolCalls: 0,
  interrupts: 0,
  compactions: 0,
  ...o,
});

const period: Period = { from: day("2026-08-10"), to: day("2026-08-17") + DAY - 1 };

describe("DAY_BOUNDARY_OFFSET_SEC", () => {
  describe("正常系", () => {
    it("運用タイムゾーン（JST）の境界になっている", () => {
      expect(DAY_BOUNDARY_OFFSET_SEC).toBe(9 * 3600);
    });
  });
});

describe("resolvePeriod", () => {
  describe("正常系", () => {
    it("指定した日が、その日の 00:00 から 23:59:59 になる", () => {
      const p = resolvePeriod("2026-08-19", "2026-08-19", day("2026-08-25"));
      const jst = (e: number) => new Date(e * 1000).toLocaleString("sv-SE", { timeZone: "Asia/Tokyo" });
      expect(jst(p.from)).toBe("2026-08-19 00:00:00");
      expect(jst(p.to)).toBe("2026-08-19 23:59:59");
    });

    it("両端を指定すればその期間になる", () => {
      const p = resolvePeriod("2026-08-01", "2026-08-07", day("2026-08-19"));
      expect(p.from).toBe(day("2026-08-01"));
      expect(p.to).toBe(day("2026-08-07") + DAY - 1);
    });

    it("to を省略すると now の日になる", () => {
      const p = resolvePeriod("2026-08-01", undefined, day("2026-08-19") + 3600);
      expect(p.to).toBe(day("2026-08-19") + DAY - 1);
    });

    it("from を省略すると to の 7 日前になる", () => {
      const p = resolvePeriod(undefined, "2026-08-19", day("2026-08-19"));
      expect(p.from).toBe(day("2026-08-12"));
    });

    it("同じ入力なら実行時刻に依らず同じ期間になる", () => {
      const a = resolvePeriod("2026-08-01", "2026-08-07", day("2026-08-19"));
      const b = resolvePeriod("2026-08-01", "2026-08-07", day("2027-01-01"));
      expect(a).toEqual(b);
    });

    it("同じ日を指定するとその 1 日になる", () => {
      const p = resolvePeriod("2026-08-05", "2026-08-05", day("2026-08-19"));
      expect(p.to - p.from).toBe(DAY - 1);
    });
  });

  describe("異常系", () => {
    it("日付の形式が違えば拒否する", () => {
      expect(() => resolvePeriod("2026/08/01", undefined, day("2026-08-19"))).toThrow("YYYY-MM-DD");
    });

    it("存在しない日付を拒否する", () => {
      expect(() => resolvePeriod("2026-13-99", undefined, day("2026-08-19"))).toThrow("解釈できない");
    });

    it("from が to より後なら拒否する", () => {
      expect(() => resolvePeriod("2026-08-20", "2026-08-01", day("2026-08-19"))).toThrow("逆転");
    });

    it("to を省略していても from が未来なら拒否する", () => {
      expect(() => resolvePeriod("2027-01-01", undefined, day("2026-08-19"))).toThrow("to=(既定)");
    });
  });
});

describe("overlapsPeriod", () => {
  describe("正常系", () => {
    it("期間に完全に収まるセッションを含む", () => {
      expect(overlapsPeriod(facts({ startedAt: day("2026-08-12"), endedAt: day("2026-08-13") }), period)).toBe(true);
    });

    it("期間の前から始まり中で終わるセッションを含む", () => {
      expect(overlapsPeriod(facts({ startedAt: day("2026-08-01"), endedAt: day("2026-08-11") }), period)).toBe(true);
    });

    it("期間の中で始まり後で終わるセッションを含む", () => {
      expect(overlapsPeriod(facts({ startedAt: day("2026-08-16"), endedAt: day("2026-08-25") }), period)).toBe(true);
    });

    it("期間全体を覆うセッションを含む", () => {
      expect(overlapsPeriod(facts({ startedAt: day("2026-08-01"), endedAt: day("2026-08-25") }), period)).toBe(true);
    });
  });

  describe("異常系", () => {
    it("期間より前に終わったセッションを外す", () => {
      expect(overlapsPeriod(facts({ startedAt: day("2026-08-01"), endedAt: day("2026-08-02") }), period)).toBe(false);
    });

    it("期間より後に始まるセッションを外す", () => {
      expect(overlapsPeriod(facts({ startedAt: day("2026-08-20"), endedAt: day("2026-08-21") }), period)).toBe(false);
    });

    it("開始時刻の無いセッションを外す", () => {
      expect(overlapsPeriod(facts({ endedAt: day("2026-08-12") }), period)).toBe(false);
    });

    it("終了時刻の無いセッションを外す", () => {
      expect(overlapsPeriod(facts({ startedAt: day("2026-08-12") }), period)).toBe(false);
    });
  });
});

describe("summarizePeriod", () => {
  const inside = { startedAt: day("2026-08-12"), endedAt: day("2026-08-13") };

  describe("正常系", () => {
    it("期間内のセッションだけを畳む", () => {
      const s = summarizePeriod(
        [facts({ ...inside, prompts: 3 }), facts({ startedAt: day("2026-08-01"), endedAt: day("2026-08-02"), prompts: 99 })],
        period,
      );
      expect(s.sessions).toBe(1);
      expect(s.prompts).toBe(3);
    });

    it("クライアント別に数える", () => {
      const s = summarizePeriod([facts(inside), facts({ ...inside, client: "codex" })], period);
      expect(s.byClient).toEqual({ claude: 1, codex: 1 });
    });

    it("スキル呼出を合算する", () => {
      const s = summarizePeriod(
        [facts({ ...inside, skillCalls: { commit: 2 } }), facts({ ...inside, skillCalls: { commit: 1, "submit-pr": 4 } })],
        period,
      );
      expect(s.skillCalls).toEqual({ commit: 3, "submit-pr": 4 });
    });

    it("失敗率をツール呼出数に対して出す", () => {
      const s = summarizePeriod([facts({ ...inside, toolCalls: 10, toolFailures: 2 })], period);
      expect(s.toolFailureRate).toBeCloseTo(0.2);
    });

    it("ブランチを重複なく並べる", () => {
      const s = summarizePeriod([facts({ ...inside, branches: ["b", "a"] }), facts({ ...inside, branches: ["a"] })], period);
      expect(s.branches).toEqual(["a", "b"]);
    });
  });

  describe("異常系", () => {
    it("観測できたセッションが無ければ失敗数は undefined になる", () => {
      const s = summarizePeriod([facts({ ...inside, client: "codex", toolCalls: 5 })], period);
      expect(s.toolFailures).toBeUndefined();
      expect(s.toolFailureRate).toBeUndefined();
    });

    it("観測できたセッションが無ければスキル呼出は undefined になる", () => {
      const s = summarizePeriod([facts({ ...inside, client: "codex" })], period);
      expect(s.skillCalls).toBeUndefined();
    });

    it("観測はできたが 0 件なら 0 として残る", () => {
      const s = summarizePeriod([facts({ ...inside, skillCalls: {}, toolCalls: 3, toolFailures: 0 })], period);
      expect(s.skillCalls).toEqual({});
      expect(s.toolFailures).toBe(0);
    });

    it("ツール呼出が 0 なら失敗率は出さない", () => {
      const s = summarizePeriod([facts({ ...inside, toolCalls: 0, toolFailures: 0 })], period);
      expect(s.toolFailureRate).toBeUndefined();
    });

    it("該当するセッションが無くても壊れない", () => {
      const s = summarizePeriod([], period);
      expect(s).toMatchObject({ sessions: 0, prompts: 0, branches: [] });
    });
  });
});

describe("uncalledSkills", () => {
  const inside = { startedAt: day("2026-08-12"), endedAt: day("2026-08-13") };

  describe("正常系", () => {
    it("呼ばれなかった宣言済みスキルを挙げる", () => {
      const s = summarizePeriod([facts({ ...inside, skillCalls: { commit: 1 } })], period);
      expect(uncalledSkills(["commit", "tool-map", "arch-check"], s)).toEqual(["arch-check", "tool-map"]);
    });

    it("すべて呼ばれていれば空になる", () => {
      const s = summarizePeriod([facts({ ...inside, skillCalls: { commit: 1 } })], period);
      expect(uncalledSkills(["commit"], s)).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("スキル呼出を観測できない期間では候補を出さない", () => {
      const s = summarizePeriod([facts({ ...inside, client: "codex" })], period);
      expect(uncalledSkills(["commit", "tool-map"], s)).toEqual([]);
    });
  });
});
