import { describe, expect, it } from "vitest";

import {
  BODY_SECTIONS,
  IMPROVEMENT_SECTION,
  issueTitle,
  parseObservation,
  parseSections,
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

    it("読解を渡せば節の下に本文を置く", () => {
      expect(renderBody(obs(), { Outcome: "実装して merge した" })).toContain("## Outcome\n実装して merge した");
    });

    it("読解を渡しても観測ブロックは読み返せる", () => {
      const original = obs({ prompts: 7 });
      expect(parseObservation(renderBody(original, { Friction: "遅かった" }))).toEqual(original);
    });
  });

  describe("異常系", () => {
    it("読解に無い節は空のまま置く", () => {
      const body = renderBody(obs(), { Outcome: "ok" });
      expect(body).toContain("## Friction\n");
      expect(body).not.toContain("## Friction\nok");
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

describe("IMPROVEMENT_SECTION", () => {
  describe("正常系", () => {
    it("本文に置く節のひとつである", () => {
      expect(BODY_SECTIONS).toContain(IMPROVEMENT_SECTION);
    });
  });
});

describe("parseSections", () => {
  describe("正常系", () => {
    it("見出しごとに本文を切り分ける", () => {
      const s = parseSections("## Outcome\n実装した\n\n## Friction\n遅かった\n");
      expect(s).toEqual({ Outcome: "実装した", Friction: "遅かった" });
    });

    it("複数行の節を改行ごと保つ", () => {
      expect(parseSections("## Outcome\n1 行目\n2 行目\n").Outcome).toBe("1 行目\n2 行目");
    });

    it("renderBody が書いたものをそのまま読み返せる", () => {
      const sections = { Outcome: "実装して merge した", [IMPROVEMENT_SECTION]: "skill を 1 つ減らす" };
      expect(parseSections(renderBody(obs(), sections))).toEqual(sections);
    });

    it("観測ブロックの行は節として拾わない", () => {
      expect(parseSections(renderBody(obs({ prompts: 3 })))).toEqual({});
    });
  });

  describe("異常系", () => {
    it("知らない見出しの本文は捨てる", () => {
      expect(parseSections("## Nonsense\n拾わない\n")).toEqual({});
    });

    it("空の節は返さない", () => {
      expect(parseSections("## Outcome\n\n## Friction\nある\n")).toEqual({ Friction: "ある" });
    });

    it("見出しの空白が無くても節として読む", () => {
      expect(parseSections("##Outcome\nok\n")).toEqual({ Outcome: "ok" });
    });

    it("見出しの大小文字が崩れても節として読む", () => {
      expect(parseSections("## outcome\nok\n")).toEqual({ Outcome: "ok" });
    });

    it("見出しが崩れても次の節を巻き込まない", () => {
      expect(parseSections("## Outcome\nA\n##Friction\nB\n")).toEqual({ Outcome: "A", Friction: "B" });
    });

    it("節がひとつも無ければ空を返す", () => {
      expect(parseSections("見出しのない本文")).toEqual({});
    });
  });
});
