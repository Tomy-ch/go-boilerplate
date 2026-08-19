import { describe, expect, it } from "vitest";

import {
  BODY_SECTIONS,
  issueTitle,
  parseObservation,
  renderBody,
  renderObservation,
  type Observation,
} from "./issue";

const obs = (o: Partial<Observation> = {}): Observation => ({
  windowId: "w1787-abc",
  client: "claude",
  phases: [],
  ...o,
});

describe("BODY_SECTIONS", () => {
  describe("正常系", () => {
    it("Outcome から始まり Evidence で終わる", () => {
      expect(BODY_SECTIONS[0]).toBe("Outcome");
      expect(BODY_SECTIONS[BODY_SECTIONS.length - 1]).toBe("Evidence");
    });

    it("重複した節を持たない", () => {
      expect(new Set(BODY_SECTIONS).size).toBe(BODY_SECTIONS.length);
    });
  });
});

describe("issueTitle", () => {
  describe("正常系", () => {
    it("ブランチがあればブランチを見出しにする", () => {
      expect(issueTitle(obs({ branch: "feature/x" }))).toBe("[feedback] feature/x (w1787-abc)");
    });

    it("窓 ID を含め、本文を読まずに突き合わせられるようにする", () => {
      expect(issueTitle(obs())).toContain("w1787-abc");
    });
  });

  describe("異常系", () => {
    it("ブランチが無ければクライアント名で代替する", () => {
      expect(issueTitle(obs({ client: "codex" }))).toBe("[feedback] codex (w1787-abc)");
    });
  });
});

describe("renderObservation", () => {
  describe("正常系", () => {
    it("観測ブロックとして囲む", () => {
      const out = renderObservation(obs());
      expect(out.startsWith("```yaml closed-loop")).toBe(true);
      expect(out.endsWith("```")).toBe(true);
    });

    it("フェーズを 1 行 1 件で書く", () => {
      const out = renderObservation(obs({ phases: [{ from: "openedAt", to: "commitAt", sec: 120 }] }));
      expect(out).toContain("  - {from: openedAt, to: commitAt, sec: 120}");
    });

    it("スキル呼出を名前ごとに書く", () => {
      const out = renderObservation(obs({ skills: { commit: 2 } }));
      expect(out).toContain("skills:");
      expect(out).toContain("  commit: 2");
    });
  });

  describe("異常系", () => {
    it("観測できなかった項目は行ごと書かない", () => {
      const out = renderObservation(obs());
      expect(out).not.toContain("tool_failures");
      expect(out).not.toContain("skills:");
    });

    it("0 は観測された値なので書く", () => {
      expect(renderObservation(obs({ toolFailures: 0 }))).toContain("tool_failures: 0");
    });

    it("スキルが空でも観測できていれば節を書く", () => {
      expect(renderObservation(obs({ skills: {} }))).toContain("skills:");
    });
  });
});

describe("renderBody", () => {
  describe("正常系", () => {
    it("要件の H2 節をすべて置く", () => {
      const body = renderBody(obs());
      for (const s of BODY_SECTIONS) expect(body).toContain(`## ${s}`);
    });

    it("手で編集しないことを本文に明記する", () => {
      expect(renderBody(obs())).toContain("手で編集する");
    });
  });
});

describe("parseObservation", () => {
  describe("正常系", () => {
    it("書いたものをそのまま読み返せる", () => {
      const original = obs({
        branch: "feature/x",
        pr: 1234,
        parentIssue: 1204,
        kind: "implementation",
        openedAt: 100,
        closedAt: 500,
        sessions: 3,
        prompts: 45,
        toolCalls: 320,
        toolFailures: 7,
        interrupts: 2,
        phases: [
          { from: "openedAt", to: "implStartedAt", sec: 120 },
          { from: "implStartedAt", to: "closedAt", sec: 280 },
        ],
        skills: { commit: 1, "impl-review": 2 },
      });
      expect(parseObservation(renderBody(original))).toEqual(original);
    });

    it("H2 節に文章が書き足されても読み返せる", () => {
      const body = `${renderBody(obs({ toolCalls: 5 }))}\nここに AI が Friction を書く\n`;
      expect(parseObservation(body)?.toolCalls).toBe(5);
    });

    it("負の秒を持つフェーズも読む", () => {
      const body = renderBody(obs({ phases: [{ from: "commitAt", to: "closedAt", sec: -66 }] }));
      expect(parseObservation(body)?.phases[0]?.sec).toBe(-66);
    });
  });

  describe("異常系", () => {
    it("観測ブロックが無ければ undefined を返す", () => {
      expect(parseObservation("## Outcome\n\nふつうの issue")).toBeUndefined();
    });

    it("ブロックが閉じていなければ undefined を返す", () => {
      expect(parseObservation("```yaml closed-loop\nwindow_id: w1\nclient: claude")).toBeUndefined();
    });

    it("window_id が無ければ undefined を返す", () => {
      expect(parseObservation("```yaml closed-loop\nclient: claude\n```")).toBeUndefined();
    });

    it("client が無ければ undefined を返す", () => {
      expect(parseObservation("```yaml closed-loop\nwindow_id: w1\n```")).toBeUndefined();
    });

    it("数値でない値を持つ項目は落として読み進める", () => {
      const body = "```yaml closed-loop\nwindow_id: w1\nclient: claude\npr: 未定\n```";
      const parsed = parseObservation(body);
      expect(parsed?.windowId).toBe("w1");
      expect(parsed?.pr).toBeUndefined();
    });

    it("壊れたフェーズ行を飛ばす", () => {
      const body = "```yaml closed-loop\nwindow_id: w1\nclient: claude\nphases:\n  - 壊れた行\n```";
      expect(parseObservation(body)?.phases).toEqual([]);
    });

    it("壊れたスキル行を飛ばす", () => {
      const body = "```yaml closed-loop\nwindow_id: w1\nclient: claude\nskills:\n  壊れた\n```";
      expect(parseObservation(body)?.skills).toEqual({});
    });

    it("スキル節が無ければ観測できなかったものとして undefined になる", () => {
      const body = "```yaml closed-loop\nwindow_id: w1\nclient: claude\n```";
      expect(parseObservation(body)?.skills).toBeUndefined();
    });

    it("どの形にも当てはまらない行を飛ばす", () => {
      const body = "```yaml closed-loop\nwindow_id: w1\nこれは何でもない行\nclient: claude\n```";
      expect(parseObservation(body)?.client).toBe("claude");
    });

    it("空行を挟んでも読み進める", () => {
      const body = "```yaml closed-loop\nwindow_id: w1\n\nclient: claude\n```";
      expect(parseObservation(body)?.client).toBe("claude");
    });
  });
});
