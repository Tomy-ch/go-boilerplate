import { describe, expect, it } from "vitest";

import type { Candidate } from "./candidates";
import { BODY_SECTIONS, type Observation } from "./issue";
import { FINDING_KINDS } from "./score";
import {
  buildPrompt,
  issueLabels,
  kindLabels,
  needsCandidateComment,
  parseSummary,
  LOCAL_CANDIDATE_LIMIT,
  LOCAL_EXCERPT_CHARS,
  NEEDS_SUMMARY_LABEL,
  type Summary,
} from "./summarize";

const observation: Observation = {
  windowId: "w1",
  client: "claude",
  branch: "feature/1204-closed-loop",
  pr: 1250,
  openedAt: 100,
  closedAt: 900,
  phases: [{ from: "openedAt", to: "implStartedAt", sec: 300 }],
  prompts: 12,
  toolCalls: 40,
  toolFailures: 3,
  interrupts: 1,
};

const candidate = (reason: Candidate["reason"], text: string): Candidate => ({ at: 1, reason, text });

const summary = (kinds: Summary["kinds"]): Summary => ({ sections: { Outcome: "完了" }, kinds });

describe("LOCAL_CANDIDATE_LIMIT", () => {
  describe("正常系", () => {
    it("公開時の既定より多く材料を取る", () => {
      expect(LOCAL_CANDIDATE_LIMIT).toBeGreaterThan(20);
    });
  });
});

describe("LOCAL_EXCERPT_CHARS", () => {
  describe("正常系", () => {
    it("公開時の既定より長く材料を取る", () => {
      expect(LOCAL_EXCERPT_CHARS).toBeGreaterThan(300);
    });
  });
});

describe("NEEDS_SUMMARY_LABEL", () => {
  describe("正常系", () => {
    it("分類ラベルと同じ接頭辞を持つ", () => {
      expect(NEEDS_SUMMARY_LABEL.startsWith("feedback/")).toBe(true);
    });

    it("分類として集計されない名前である", () => {
      const asKind = NEEDS_SUMMARY_LABEL.slice("feedback/".length);
      expect(FINDING_KINDS).not.toContain(asKind);
    });
  });
});

describe("buildPrompt", () => {
  describe("正常系", () => {
    it("観測の値を問いに含める", () => {
      const p = buildPrompt(observation, []);
      expect(p).toContain("w1");
      expect(p).toContain("feature/1204-closed-loop");
      expect(p).toContain("#1250");
    });

    it("フェーズ区間を秒付きで並べる", () => {
      expect(buildPrompt(observation, [])).toContain("openedAt → implStartedAt: 300 秒");
    });

    it("候補を選定理由つきの逐語として並べる", () => {
      const p = buildPrompt(observation, [candidate("corrective", "そうじゃなくて")]);
      expect(p).toContain("corrective");
      expect(p).toContain("> そうじゃなくて");
    });

    it("埋めるべき節を本文と同じ順で列挙する", () => {
      const p = buildPrompt(observation, []);
      const positions = BODY_SECTIONS.map((s) => p.indexOf(`## ${s}`));
      expect(positions.every((n) => n >= 0)).toBe(true);
      expect([...positions].sort((a, b) => a - b)).toEqual(positions);
    });

    it("推測を禁じる指示を含める", () => {
      expect(buildPrompt(observation, [])).toContain("読み取れないことは書かない");
    });
  });

  describe("異常系", () => {
    it("候補が無ければ該当なしと伝える", () => {
      expect(buildPrompt(observation, [])).toContain("該当したターンはありません");
    });

    it("観測できなかった項目は問いに書かない", () => {
      const bare: Observation = { windowId: "w2", client: "codex", phases: [] };
      const p = buildPrompt(bare, []);
      expect(p).not.toContain("ブランチ:");
      expect(p).not.toContain("ツール失敗:");
    });
  });
});

describe("parseSummary", () => {
  describe("正常系", () => {
    it("見出しごとに本文を切り分ける", () => {
      const parsed = parseSummary("## Outcome\n実装した\n\n## Friction\n遅かった\n");
      expect(parsed?.sections.Outcome).toBe("実装した");
      expect(parsed?.sections.Friction).toBe("遅かった");
    });

    it("kinds 行から既知の分類を取り出す", () => {
      const parsed = parseSummary("## Outcome\nok\nkinds: skill, ci\n");
      expect(parsed?.kinds).toEqual(["skill", "ci"]);
    });

    it("kinds 行は節の本文に混ぜない", () => {
      const parsed = parseSummary("## Outcome\nok\nkinds: skill\n");
      expect(parsed?.sections.Outcome).toBe("ok");
    });

    it("重複した分類は 1 度だけ数える", () => {
      const parsed = parseSummary("## Outcome\nok\nkinds: skill, skill\n");
      expect(parsed?.kinds).toEqual(["skill"]);
    });

    it("複数行の節を改行ごと保つ", () => {
      const parsed = parseSummary("## Outcome\n1 行目\n2 行目\n");
      expect(parsed?.sections.Outcome).toBe("1 行目\n2 行目");
    });
  });

  describe("異常系", () => {
    it("知らない見出しの本文は捨てる", () => {
      const parsed = parseSummary("## Outcome\nok\n## Nonsense\n拾わない\n");
      expect(Object.keys(parsed?.sections ?? {})).toEqual(["Outcome"]);
    });

    it("知らない分類は捨てる", () => {
      const parsed = parseSummary("## Outcome\nok\nkinds: skill, 未知\n");
      expect(parsed?.kinds).toEqual(["skill"]);
    });

    it("既知の節が 1 つも無ければ undefined を返す", () => {
      expect(parseSummary("読解に失敗しました")).toBeUndefined();
    });

    it("空の出力は undefined を返す", () => {
      expect(parseSummary("")).toBeUndefined();
    });

    it("kinds 行が空でも節が取れれば読解として扱う", () => {
      const parsed = parseSummary("## Outcome\nok\nkinds:\n");
      expect(parsed?.kinds).toEqual([]);
      expect(parsed?.sections.Outcome).toBe("ok");
    });
  });
});

describe("kindLabels", () => {
  describe("正常系", () => {
    it("分類をラベル名に変換する", () => {
      expect(kindLabels(["skill", "ci"])).toEqual(["feedback/skill", "feedback/ci"]);
    });
  });

  describe("異常系", () => {
    it("分類が無ければ空を返す", () => {
      expect(kindLabels([])).toEqual([]);
    });
  });
});

describe("issueLabels", () => {
  describe("正常系", () => {
    it("読解できていれば分類ラベルを重ねる", () => {
      expect(issueLabels(summary(["skill"]))).toEqual(["feedback", "feedback/skill"]);
    });

    it("読解できていれば needs-summary を付けない", () => {
      expect(issueLabels(summary([]))).not.toContain(NEEDS_SUMMARY_LABEL);
    });
  });

  describe("異常系", () => {
    it("読解できなければ needs-summary を付ける", () => {
      expect(issueLabels(undefined)).toEqual(["feedback", NEEDS_SUMMARY_LABEL]);
    });
  });
});

describe("needsCandidateComment", () => {
  describe("正常系", () => {
    it("読解できていれば逐語を公開しない", () => {
      expect(needsCandidateComment(summary(["skill"]))).toBe(false);
    });
  });

  describe("異常系", () => {
    it("読解できなければ CI へ渡すために公開する", () => {
      expect(needsCandidateComment(undefined)).toBe(true);
    });
  });
});
