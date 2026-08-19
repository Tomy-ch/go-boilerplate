import { describe, expect, it } from "vitest";

import type { Observation } from "./issue";
import {
  DEFAULT_WEIGHTS,
  clusterIssues,
  clusterKey,
  failureRate,
  mergeWaitSec,
  waitDominated,
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
