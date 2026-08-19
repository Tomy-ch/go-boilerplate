import { describe, expect, it } from "vitest";

import {
  MARK_ORDER,
  anomaliesOf,
  isSubstantive,
  substantiveMarks,
  parseMarkFile,
  phasesOf,
  representativeAt,
  stampCount,
  toWindow,
  totalDurationSec,
  type Window,
} from "./windows";

const windowOf = (marks: Record<string, number[]>): Window => ({ id: "w1-test", marks });

describe("MARK_ORDER", () => {
  describe("正常系", () => {
    it("窓の開始と終了で挟まれている", () => {
      expect(MARK_ORDER[0]).toBe("openedAt");
      expect(MARK_ORDER[MARK_ORDER.length - 1]).toBe("closedAt");
    });

    it("重複した名前を持たない", () => {
      expect(new Set(MARK_ORDER).size).toBe(MARK_ORDER.length);
    });
  });
});

describe("parseMarkFile", () => {
  describe("正常系", () => {
    it("1 行 1 epoch を打刻順に読む", () => {
      expect(parseMarkFile("100\n200\n300\n")).toEqual([100, 200, 300]);
    });

    it("末尾に改行が無くても読む", () => {
      expect(parseMarkFile("100")).toEqual([100]);
    });

    it("前後の空白を無視する", () => {
      expect(parseMarkFile("  100  \n\t200\n")).toEqual([100, 200]);
    });
  });

  describe("異常系", () => {
    it("空ファイルは空配列になる", () => {
      expect(parseMarkFile("")).toEqual([]);
    });

    it("数値でない行を落とす", () => {
      expect(parseMarkFile("100\nnot-a-number\n200\n")).toEqual([100, 200]);
    });

    it("0 と負数を落とす", () => {
      expect(parseMarkFile("0\n-5\n100\n")).toEqual([100]);
    });

    it("小数を落とす", () => {
      expect(parseMarkFile("100.5\n200\n")).toEqual([200]);
    });
  });
});

describe("toWindow", () => {
  describe("正常系", () => {
    it("正準順序にあるマーカーを取り込む", () => {
      const w = toWindow("w1", { openedAt: "100\n", closedAt: "300\n" });
      expect(w.id).toBe("w1");
      expect(w.marks).toEqual({ openedAt: [100], closedAt: [300] });
    });
  });

  describe("異常系", () => {
    it("正準順序に無い名前を捨てる", () => {
      const w = toWindow("w1", { openedAt: "100\n", bogusAt: "200\n" });
      expect(w.marks).toEqual({ openedAt: [100] });
    });

    it("中身が空のマーカーは持たない", () => {
      const w = toWindow("w1", { openedAt: "100\n", commitAt: "\n\n" });
      expect(w.marks).toEqual({ openedAt: [100] });
    });
  });
});

describe("representativeAt", () => {
  describe("正常系", () => {
    it("最初の打刻を代表時刻にする", () => {
      expect(representativeAt(windowOf({ commitAt: [100, 200, 300] }), "commitAt")).toBe(100);
    });
  });

  describe("異常系", () => {
    it("打刻の無いマーカーは undefined になる", () => {
      expect(representativeAt(windowOf({}), "commitAt")).toBeUndefined();
    });
  });
});

describe("stampCount", () => {
  describe("正常系", () => {
    it("繰り返した打刻を数える", () => {
      expect(stampCount(windowOf({ reviewStartedAt: [100, 200] }), "reviewStartedAt")).toBe(2);
    });
  });

  describe("異常系", () => {
    it("打刻の無いマーカーは 0 になる", () => {
      expect(stampCount(windowOf({}), "reviewStartedAt")).toBe(0);
    });
  });
});

describe("phasesOf", () => {
  describe("正常系", () => {
    it("隣り合うマーカーの間に区間を張る", () => {
      const phases = phasesOf(windowOf({ openedAt: [100], implStartedAt: [150], closedAt: [400] }));
      expect(phases).toEqual([
        { from: "openedAt", to: "implStartedAt", startedAt: 100, endedAt: 150, durationSec: 50 },
        { from: "implStartedAt", to: "closedAt", startedAt: 150, endedAt: 400, durationSec: 250 },
      ]);
    });

    it("欠けたマーカーを飛ばして次の実在マーカーへ繋ぐ", () => {
      const phases = phasesOf(windowOf({ openedAt: [100], mergedAt: [500] }));
      expect(phases).toHaveLength(1);
      expect(phases[0]).toMatchObject({ from: "openedAt", to: "mergedAt", durationSec: 400 });
    });

    it("繰り返した打刻があっても最初の時刻で区間を張る", () => {
      const phases = phasesOf(windowOf({ openedAt: [100], commitAt: [200, 900] }));
      expect(phases[0]).toMatchObject({ durationSec: 100 });
    });
  });

  describe("異常系", () => {
    it("マーカーが 1 つだけなら区間は張られない", () => {
      expect(phasesOf(windowOf({ openedAt: [100] }))).toEqual([]);
    });

    it("マーカーが無ければ区間は張られない", () => {
      expect(phasesOf(windowOf({}))).toEqual([]);
    });

    it("時刻が前後していても宣言順のまま負の区間として残す", () => {
      const phases = phasesOf(windowOf({ openedAt: [500], closedAt: [100] }));
      expect(phases[0]).toMatchObject({ from: "openedAt", to: "closedAt", durationSec: -400 });
    });
  });
});

describe("anomaliesOf", () => {
  describe("正常系", () => {
    it("開始と終了が揃い時刻も順当なら何も報告しない", () => {
      expect(anomaliesOf(windowOf({ openedAt: [100], closedAt: [200] }))).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("openedAt が無ければ missing-open を報告する", () => {
      const kinds = anomaliesOf(windowOf({ closedAt: [200] })).map((a) => a.kind);
      expect(kinds).toContain("missing-open");
    });

    it("closedAt が無ければ unclosed を報告する", () => {
      const kinds = anomaliesOf(windowOf({ openedAt: [100] })).map((a) => a.kind);
      expect(kinds).toContain("unclosed");
    });

    it("時刻が前後していれば out-of-order を報告する", () => {
      const anomalies = anomaliesOf(windowOf({ openedAt: [500], closedAt: [100] }));
      expect(anomalies.map((a) => a.kind)).toContain("out-of-order");
      expect(anomalies.find((a) => a.kind === "out-of-order")?.detail).toContain("openedAt");
    });

    it("マーカーが 1 つも無ければ開始と終了の両方を報告する", () => {
      const kinds = anomaliesOf(windowOf({})).map((a) => a.kind);
      expect(kinds).toEqual(["missing-open", "unclosed"]);
    });
  });
});

describe("substantiveMarks", () => {
  describe("正常系", () => {
    it("開始と終了以外の打刻を挙げる", () => {
      const w = windowOf({ openedAt: [100], commitAt: [200], closedAt: [300] });
      expect(substantiveMarks(w)).toEqual(["commitAt"]);
    });

    it("正準順序で並べる", () => {
      const w = windowOf({ reviewStartedAt: [300], implStartedAt: [200] });
      expect(substantiveMarks(w)).toEqual(["implStartedAt", "reviewStartedAt"]);
    });
  });

  describe("異常系", () => {
    it("開始と終了しか無ければ空になる", () => {
      expect(substantiveMarks(windowOf({ openedAt: [100], closedAt: [200] }))).toEqual([]);
    });

    it("打刻が無ければ空になる", () => {
      expect(substantiveMarks(windowOf({}))).toEqual([]);
    });
  });
});

describe("isSubstantive", () => {
  describe("正常系", () => {
    it("実質的な打刻があれば送るに値する", () => {
      expect(isSubstantive(windowOf({ openedAt: [100], commitAt: [150], closedAt: [200] }))).toBe(true);
    });

    it("打刻が無くても発話があれば送るに値する", () => {
      const w = windowOf({ openedAt: [100], closedAt: [200] });
      expect(isSubstantive(w, { prompts: 3, toolCalls: 0 })).toBe(true);
    });

    it("打刻が無くてもツール呼出があれば送るに値する", () => {
      const w = windowOf({ openedAt: [100], closedAt: [200] });
      expect(isSubstantive(w, { prompts: 0, toolCalls: 5 })).toBe(true);
    });

    it("打刻があれば活動が 0 でも送るに値する", () => {
      const w = windowOf({ openedAt: [100], reviewStartedAt: [150], closedAt: [200] });
      expect(isSubstantive(w, { prompts: 0, toolCalls: 0 })).toBe(true);
    });
  });

  describe("異常系", () => {
    it("開いて閉じただけの窓は送らない", () => {
      const w = windowOf({ openedAt: [100], closedAt: [200] });
      expect(isSubstantive(w, { prompts: 0, toolCalls: 0 })).toBe(false);
    });

    it("活動を観測できなければ、打刻が無い窓は送らない", () => {
      expect(isSubstantive(windowOf({ openedAt: [100], closedAt: [200] }))).toBe(false);
    });

    it("打刻が 1 つも無い窓は送らない", () => {
      expect(isSubstantive(windowOf({}), { prompts: 0, toolCalls: 0 })).toBe(false);
    });
  });
});

describe("totalDurationSec", () => {
  describe("正常系", () => {
    it("開始から終了までの秒を返す", () => {
      expect(totalDurationSec(windowOf({ openedAt: [100], closedAt: [460] }))).toBe(360);
    });
  });

  describe("異常系", () => {
    it("終了が無ければ undefined を返す", () => {
      expect(totalDurationSec(windowOf({ openedAt: [100] }))).toBeUndefined();
    });

    it("開始が無ければ undefined を返す", () => {
      expect(totalDurationSec(windowOf({ closedAt: [100] }))).toBeUndefined();
    });
  });
});
