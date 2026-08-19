import { describe, expect, it } from "vitest";

import type { Observation } from "./issue";
import {
  DEFAULT_WEIGHTS,
  FINDING_KINDS,
  KIND_LABEL_PREFIX,
  labelsToKinds,
  clusterIssues,
  clusterKey,
  failureRate,
  mergeWaitSec,
  reevaluations,
  waitDominated,
  REEVALUATION_DAYS,
  UNCLASSIFIED,
  type FeedbackIssue,
  type FindingKind,
} from "./score";

const obs = (o: Partial<Observation> = {}): Observation => ({
  windowId: "w1",
  client: "claude",
  phases: [],
  ...o,
});

const issue = (number: number, kinds: FindingKind[], o: Partial<Observation> = {}): FeedbackIssue => ({
  number,
  kinds,
  observation: obs(o),
});

describe("DEFAULT_WEIGHTS", () => {
  describe("正常系", () => {
    it("再発を最も重く見る", () => {
      const w = DEFAULT_WEIGHTS;
      expect(w.recurrence).toBeGreaterThan(w.humanIntervention);
      expect(w.humanIntervention).toBeGreaterThan(w.impact);
      expect(w.impact).toBeGreaterThan(w.frequency);
    });
  });
});

describe("KIND_LABEL_PREFIX", () => {
  describe("正常系", () => {
    it("ラベルの名前空間と一致している", () => {
      expect(KIND_LABEL_PREFIX).toBe("feedback/");
    });
  });
});

describe("FINDING_KINDS", () => {
  describe("正常系", () => {
    it("重複した分類を持たない", () => {
      expect(new Set(FINDING_KINDS).size).toBe(FINDING_KINDS.length);
    });
  });
});

describe("labelsToKinds", () => {
  describe("正常系", () => {
    it("接頭辞を外して分類を取り出す", () => {
      expect(labelsToKinds(["feedback/skill", "feedback/ci"])).toEqual(["skill", "ci"]);
    });

    it("宣言されている全分類を取り出せる", () => {
      const labels = FINDING_KINDS.map((k) => `${KIND_LABEL_PREFIX}${k}`);
      expect(labelsToKinds(labels)).toEqual([...FINDING_KINDS]);
    });
  });

  describe("異常系", () => {
    it("接頭辞の無いラベルを捨てる", () => {
      expect(labelsToKinds(["feedback", "bug", "feedback/skill"])).toEqual(["skill"]);
    });

    it("知らない分類を捨てる", () => {
      expect(labelsToKinds(["feedback/unknown-kind"])).toEqual([]);
    });

    it("ラベルが無ければ空になる", () => {
      expect(labelsToKinds([])).toEqual([]);
    });
  });
});

describe("failureRate", () => {
  describe("正常系", () => {
    it("呼出数に対する失敗の割合を返す", () => {
      expect(failureRate(obs({ toolCalls: 200, toolFailures: 20 }))).toBeCloseTo(0.1);
    });

    it("規模の違う窓を比較できる", () => {
      const big = failureRate(obs({ toolCalls: 10_000, toolFailures: 100 })) as number;
      const small = failureRate(obs({ toolCalls: 200, toolFailures: 20 })) as number;
      expect(small).toBeGreaterThan(big);
    });
  });

  describe("異常系", () => {
    it("呼出が観測できなければ undefined を返す", () => {
      expect(failureRate(obs({ toolFailures: 5 }))).toBeUndefined();
    });

    it("失敗が観測できなければ undefined を返す", () => {
      expect(failureRate(obs({ toolCalls: 100 }))).toBeUndefined();
    });

    it("呼出が 0 なら undefined を返す", () => {
      expect(failureRate(obs({ toolCalls: 0, toolFailures: 0 }))).toBeUndefined();
    });
  });
});

describe("mergeWaitSec", () => {
  describe("正常系", () => {
    it("PR からマージまでの区間を取り出す", () => {
      const o = obs({ phases: [{ from: "prOpenedAt", to: "mergedAt", sec: 3600 }] });
      expect(mergeWaitSec(o)).toBe(3600);
    });
  });

  describe("異常系", () => {
    it("該当する区間が無ければ undefined を返す", () => {
      expect(mergeWaitSec(obs({ phases: [{ from: "openedAt", to: "commitAt", sec: 10 }] }))).toBeUndefined();
    });
  });
});

describe("clusterKey", () => {
  describe("正常系", () => {
    it("分類ラベルを鍵にする", () => {
      expect(clusterKey(issue(1, ["skill"]))).toBe("skill");
    });

    it("複数の分類は並べ替えて 1 つの鍵にする", () => {
      expect(clusterKey(issue(1, ["tooling", "ci"]))).toBe("ci+tooling");
    });
  });

  describe("異常系", () => {
    it("分類が無ければ unclassified にまとめる", () => {
      expect(clusterKey(issue(1, []))).toBe("unclassified");
    });
  });
});

describe("clusterIssues", () => {
  describe("正常系", () => {
    it("同じ鍵の Issue をまとめる", () => {
      const clusters = clusterIssues([issue(1, ["skill"]), issue(2, ["skill"]), issue(3, ["ci"])]);
      expect(clusters).toHaveLength(2);
      expect(clusters.find((c) => c.key === "skill")?.issues).toEqual([1, 2]);
    });

    it("複数の Issue に跨れば反復事象になる", () => {
      const [c] = clusterIssues([issue(1, ["skill"]), issue(2, ["skill"])]);
      expect(c?.isRecurring).toBe(true);
      expect(c?.recurrence).toBe(1);
    });

    it("影響と人間の介入を合算する", () => {
      const [c] = clusterIssues([
        issue(1, ["skill"], { toolFailures: 10, interrupts: 2 }),
        issue(2, ["skill"], { toolFailures: 5, interrupts: 3 }),
      ]);
      expect(c?.impact).toBe(15);
      expect(c?.humanIntervention).toBe(5);
    });

    it("スコアの降順に並べる", () => {
      const clusters = clusterIssues([
        issue(1, ["skill"], { interrupts: 100 }),
        issue(2, ["ci"], { interrupts: 1 }),
      ]);
      expect(clusters[0]?.key).toBe("skill");
    });

    it("重みを差し替えられる", () => {
      const only = { frequency: 1, impact: 0, humanIntervention: 0, recurrence: 0 };
      const [c] = clusterIssues([issue(1, ["skill"], { toolFailures: 999 })], only);
      expect(c?.score).toBe(1);
    });

    it("スコアが同じなら鍵の順に並べる", () => {
      const clusters = clusterIssues([issue(2, ["skill"]), issue(1, ["ci"])]);
      expect(clusters.map((c) => c.key)).toEqual(["ci", "skill"]);
    });
  });

  describe("異常系", () => {
    it("単発は反復事象にしない", () => {
      const [c] = clusterIssues([issue(1, ["skill"])]);
      expect(c?.isRecurring).toBe(false);
      expect(c?.recurrence).toBe(0);
    });

    it("観測できていない項目を 0 として扱う", () => {
      const [c] = clusterIssues([issue(1, ["skill"])]);
      expect(c?.impact).toBe(0);
      expect(c?.humanIntervention).toBe(0);
    });

    it("Issue が無ければ空になる", () => {
      expect(clusterIssues([])).toEqual([]);
    });
  });
});

describe("waitDominated", () => {
  describe("正常系", () => {
    it("待ち時間が実装時間を上回る Issue を挙げる", () => {
      const slow = issue(1, [], {
        phases: [
          { from: "openedAt", to: "prOpenedAt", sec: 100 },
          { from: "prOpenedAt", to: "mergedAt", sec: 5000 },
        ],
      });
      expect(waitDominated([slow])).toEqual([1]);
    });

    it("番号の順に並べる", () => {
      const mk = (n: number) =>
        issue(n, [], {
          phases: [
            { from: "openedAt", to: "prOpenedAt", sec: 1 },
            { from: "prOpenedAt", to: "mergedAt", sec: 100 },
          ],
        });
      expect(waitDominated([mk(9), mk(2)])).toEqual([2, 9]);
    });
  });

  describe("異常系", () => {
    it("実装時間の方が長ければ挙げない", () => {
      const fast = issue(1, [], {
        phases: [
          { from: "openedAt", to: "prOpenedAt", sec: 5000 },
          { from: "prOpenedAt", to: "mergedAt", sec: 100 },
        ],
      });
      expect(waitDominated([fast])).toEqual([]);
    });

    it("待ち時間と実装時間が等しければ挙げない（上回るときだけ）", () => {
      const even = issue(1, [], {
        phases: [
          { from: "openedAt", to: "prOpenedAt", sec: 100 },
          { from: "prOpenedAt", to: "mergedAt", sec: 100 },
        ],
      });
      expect(waitDominated([even])).toEqual([]);
    });

    it("待ち時間が観測できなければ挙げない", () => {
      expect(waitDominated([issue(1, [], { phases: [{ from: "openedAt", to: "commitAt", sec: 10 }] })])).toEqual([]);
    });

    it("負の区間を実装時間に数えない", () => {
      const odd = issue(1, [], {
        phases: [
          { from: "openedAt", to: "prOpenedAt", sec: -500 },
          { from: "prOpenedAt", to: "mergedAt", sec: 100 },
        ],
      });
      expect(waitDominated([odd])).toEqual([1]);
    });
  });
});

const DAY = 86_400;

const tracked = (
  number: number,
  kinds: FindingKind[],
  t: { createdAt?: number; resolvedAt?: number },
): FeedbackIssue => ({
  number,
  kinds,
  observation: obs(),
  ...t,
});

describe("UNCLASSIFIED", () => {
  describe("正常系", () => {
    it("clusterKey が分類なしに返す鍵と一致する", () => {
      expect(clusterKey(issue(1, []))).toBe(UNCLASSIFIED);
    });
  });
});

describe("REEVALUATION_DAYS", () => {
  describe("正常系", () => {
    it("日数として扱える整数である", () => {
      expect(Number.isInteger(REEVALUATION_DAYS)).toBe(true);
      expect(REEVALUATION_DAYS).toBeGreaterThan(0);
    });

    it("この日数がそのまま due の境界になる", () => {
      const at = 100 + REEVALUATION_DAYS * DAY;
      expect(reevaluations([tracked(1, ["skill"], { resolvedAt: 100 })], at)[0]?.due).toBe(true);
      expect(reevaluations([tracked(1, ["skill"], { resolvedAt: 100 })], at - 1)[0]?.due).toBe(false);
    });
  });
});

describe("reevaluations", () => {
  describe("正常系", () => {
    it("閉じた Issue を着地として扱う", () => {
      const r = reevaluations([tracked(1, ["skill"], { createdAt: 0, resolvedAt: 100 })], 100 + REEVALUATION_DAYS * DAY);
      expect(r).toHaveLength(1);
      expect(r[0]?.landedIssue).toBe(1);
      expect(r[0]?.landedAt).toBe(100);
    });

    it("着地後に作られた同じ鍵の Issue を再発として挙げる", () => {
      const r = reevaluations(
        [tracked(1, ["skill"], { createdAt: 0, resolvedAt: 100 }), tracked(2, ["skill"], { createdAt: 200 })],
        1_000_000,
      );
      expect(r[0]?.recurred).toEqual([2]);
    });

    it("同じ鍵で複数閉じていれば最後の 1 件を着地とする", () => {
      const r = reevaluations(
        [
          tracked(1, ["skill"], { createdAt: 0, resolvedAt: 100 }),
          tracked(2, ["skill"], { createdAt: 50, resolvedAt: 300 }),
        ],
        1_000_000,
      );
      expect(r[0]?.landedIssue).toBe(2);
    });

    it("再発が多い鍵を先に並べる", () => {
      const r = reevaluations(
        [
          tracked(1, ["skill"], { createdAt: 0, resolvedAt: 100 }),
          tracked(2, ["ci"], { createdAt: 0, resolvedAt: 100 }),
          tracked(3, ["ci"], { createdAt: 200 }),
          tracked(4, ["ci"], { createdAt: 300 }),
        ],
        1_000_000,
      );
      expect(r[0]?.key).toBe("ci");
    });

    it("再発の数が同じなら鍵の順に並べ、実行ごとに順序が変わらないようにする", () => {
      const r = reevaluations([tracked(1, ["skill"], { resolvedAt: 100 }), tracked(2, ["ci"], { resolvedAt: 100 })], 1_000_000);
      expect(r.map((x) => x.key)).toEqual(["ci", "skill"]);
    });

    it("着地と同じ秒に作られた Issue は再発に数えない", () => {
      const r = reevaluations(
        [tracked(1, ["skill"], { createdAt: 0, resolvedAt: 100 }), tracked(2, ["skill"], { createdAt: 100 })],
        1_000_000,
      );
      expect(r[0]?.recurred).toEqual([]);
    });

    it("再発の境界は含まず、判定期の境界は含む（非対称を同じ入力で固定する）", () => {
      const r = reevaluations(
        [tracked(1, ["skill"], { createdAt: 0, resolvedAt: 100 }), tracked(2, ["skill"], { createdAt: 100 })],
        100 + REEVALUATION_DAYS * DAY,
      );
      expect(r[0]?.recurred).toEqual([]);
      expect(r[0]?.due).toBe(true);
    });

    it("判定の時期が来ていれば due を立てる", () => {
      const at = 100 + REEVALUATION_DAYS * DAY;
      expect(reevaluations([tracked(1, ["skill"], { resolvedAt: 100 })], at)[0]?.due).toBe(true);
    });
  });

  describe("異常系", () => {
    it("閉じた Issue が無い鍵は挙げない", () => {
      expect(reevaluations([tracked(1, ["skill"], { createdAt: 0 })], 1_000_000)).toEqual([]);
    });

    it("分類なしは測り直しの対象にしない", () => {
      expect(reevaluations([tracked(1, [], { createdAt: 0, resolvedAt: 100 })], 1_000_000)).toEqual([]);
    });

    it("着地より前に作られた Issue は再発に数えない", () => {
      const r = reevaluations(
        [tracked(1, ["skill"], { createdAt: 500, resolvedAt: 600 }), tracked(2, ["skill"], { createdAt: 100 })],
        1_000_000,
      );
      expect(r[0]?.recurred).toEqual([]);
    });

    it("作成時刻が観測できない Issue は再発に数えない", () => {
      const r = reevaluations(
        [tracked(1, ["skill"], { createdAt: 0, resolvedAt: 100 }), tracked(2, ["skill"], {})],
        1_000_000,
      );
      expect(r[0]?.recurred).toEqual([]);
    });

    it("14 日経つ前は due を立てない", () => {
      const at = 100 + REEVALUATION_DAYS * DAY - 1;
      expect(reevaluations([tracked(1, ["skill"], { resolvedAt: 100 })], at)[0]?.due).toBe(false);
    });
  });
});
