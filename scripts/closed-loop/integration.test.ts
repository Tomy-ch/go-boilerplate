import { describe, expect, it } from "vitest";

import { IMPROVEMENT_SECTION, type Observation } from "./issue";
import {
  buildConcernPrompt,
  parseConcerns,
  renderIntegrationBody,
  renderRollupComment,
  rollupDestinations,
  rollupTargets,
  INTEGRATION_LABEL,
  ROLLED_UP_REASON,
  type Concern,
  type RollupSource,
} from "./integration";

const obs = (o: Partial<Observation> = {}): Observation => ({
  windowId: "w1",
  client: "claude",
  phases: [],
  ...o,
});

const src = (number: number, sections: Record<string, string>, o: Partial<Observation> = {}): RollupSource => ({
  number,
  observation: obs(o),
  sections,
});

const withImprovement = (number: number, text: string) => src(number, { [IMPROVEMENT_SECTION]: text });

describe("INTEGRATION_LABEL", () => {
  describe("正常系", () => {
    it("観測側の feedback ラベルと混ざらない名前である", () => {
      expect(INTEGRATION_LABEL).not.toBe("feedback");
      expect(INTEGRATION_LABEL.startsWith("feedback/")).toBe(false);
    });
  });
});

describe("ROLLED_UP_REASON", () => {
  describe("正常系", () => {
    it("着地を表す completed とは別の理由である", () => {
      expect(ROLLED_UP_REASON).not.toBe("completed");
    });
  });
});

describe("rollupTargets", () => {
  describe("正常系", () => {
    it("改善提案を持つものを対象にする", () => {
      expect(rollupTargets([withImprovement(1, "skill を 1 つ減らす")]).map((s) => s.number)).toEqual([1]);
    });
  });

  describe("異常系", () => {
    it("改善提案が無いものは対象にしない", () => {
      expect(rollupTargets([src(2, { Outcome: "実装した" })])).toEqual([]);
    });

    it("該当なしと書かれたものは対象にしない", () => {
      expect(rollupTargets([withImprovement(3, "該当なし。根拠となる候補が無い")])).toEqual([]);
    });

    it("空文字の提案は対象にしない", () => {
      expect(rollupTargets([withImprovement(4, "")])).toEqual([]);
    });

    it("対象が無ければ空を返す", () => {
      expect(rollupTargets([])).toEqual([]);
    });
  });
});

describe("buildConcernPrompt", () => {
  describe("正常系", () => {
    it("まとめる軸がブランチではなく関心だと明示する", () => {
      const p = buildConcernPrompt([withImprovement(1, "x")]);
      expect(p).toContain("どのブランチで起きたか");
      expect(p).toContain("何が問題か");
    });

    it("各 Issue の番号とブランチを渡す", () => {
      const p = buildConcernPrompt([src(7, { [IMPROVEMENT_SECTION]: "x" }, { branch: "feature/y" })]);
      expect(p).toContain("## Feedback #7");
      expect(p).toContain("feature/y");
    });

    it("節の中身を見出しつきで渡す", () => {
      const p = buildConcernPrompt([src(1, { Friction: "遅かった", [IMPROVEMENT_SECTION]: "x" })]);
      expect(p).toContain("### Friction");
      expect(p).toContain("遅かった");
    });

    it("無理にまとめないよう指示する", () => {
      expect(buildConcernPrompt([withImprovement(1, "x")])).toContain("無理にまとめない");
    });
  });

  describe("異常系", () => {
    it("該当なしの節は渡さない", () => {
      const p = buildConcernPrompt([src(1, { Friction: "該当なし", [IMPROVEMENT_SECTION]: "x" })]);
      expect(p).not.toContain("### Friction");
    });

    it("ブランチが無ければ書かない", () => {
      expect(buildConcernPrompt([withImprovement(1, "x")])).not.toContain("ブランチ:");
    });
  });
});

describe("parseConcerns", () => {
  describe("正常系", () => {
    it("見出しと根拠と本文を読む", () => {
      const c = parseConcerns("## skill が足りない\nsources: 1, 2\n本文\n", [1, 2]);
      expect(c).toHaveLength(1);
      expect(c[0]?.title).toBe("skill が足りない");
      expect(c[0]?.sources).toEqual([1, 2]);
      expect(c[0]?.body).toBe("本文");
    });

    it("複数の関心を読む", () => {
      const c = parseConcerns("## A\nsources: 1\naaa\n## B\nsources: 2\nbbb\n", [1, 2]);
      expect(c.map((x) => x.title)).toEqual(["A", "B"]);
    });

    it("番号に # が付いていても読む", () => {
      expect(parseConcerns("## A\nsources: #1, #2\n本文\n", [1, 2])[0]?.sources).toEqual([1, 2]);
    });
  });

  describe("異常系", () => {
    it("根拠を持たない関心は捨てる", () => {
      expect(parseConcerns("## A\n本文だけ\n", [1])).toEqual([]);
    });

    it("知らない番号は根拠に数えない", () => {
      expect(parseConcerns("## A\nsources: 1, 999\n本文\n", [1])[0]?.sources).toEqual([1]);
    });

    it("既知の根拠が 1 つも無ければ関心ごと捨てる", () => {
      expect(parseConcerns("## A\nsources: 999\n本文\n", [1])).toEqual([]);
    });

    it("見出しが無ければ何も読まない", () => {
      expect(parseConcerns("sources: 1\n本文\n", [1])).toEqual([]);
    });

    it("空の出力は空を返す", () => {
      expect(parseConcerns("", [1])).toEqual([]);
    });

    it("サブ見出しは関心の区切りにしない", () => {
      const c = parseConcerns("## A\nsources: 1\n### 補足\n続き\n", [1]);
      expect(c).toHaveLength(1);
      expect(c[0]?.body).toBe("### 補足\n続き");
    });
  });
});

describe("renderIntegrationBody", () => {
  const concern: Concern = { title: "A", body: "説明", sources: [11, 22] };

  describe("正常系", () => {
    it("根拠の Issue を辿れる形で並べる", () => {
      const b = renderIntegrationBody(concern);
      expect(b).toContain("- #11");
      expect(b).toContain("- #22");
    });

    it("機械が作ったことを本文に明記する", () => {
      expect(renderIntegrationBody(concern)).toContain("週次の統合機構が作成");
    });

    it("関心の説明を含める", () => {
      expect(renderIntegrationBody(concern)).toContain("説明");
    });
  });
});

describe("renderRollupComment", () => {
  describe("正常系", () => {
    it("畳んだ先を閉じられた側から辿れるようにする", () => {
      expect(renderRollupComment([42])).toContain("#42");
    });

    it("畳み先が複数あればすべて挙げる", () => {
      const c = renderRollupComment([42, 43]);
      expect(c).toContain("#42");
      expect(c).toContain("#43");
    });
  });
});

describe("rollupDestinations", () => {
  describe("正常系", () => {
    it("大元を鍵にして畳み先を引けるようにする", () => {
      const d = rollupDestinations([{ issue: 10, sources: [1, 2] }]);
      expect(d.get(1)).toEqual([10]);
      expect(d.get(2)).toEqual([10]);
    });

    it("同じ大元が複数の関心の根拠なら畳み先をまとめる", () => {
      const d = rollupDestinations([
        { issue: 10, sources: [1] },
        { issue: 11, sources: [1] },
      ]);
      expect(d.get(1)).toEqual([10, 11]);
      expect(d.size).toBe(1);
    });
  });

  describe("異常系", () => {
    it("関心が無ければ空を返す", () => {
      expect(rollupDestinations([]).size).toBe(0);
    });
  });
});
