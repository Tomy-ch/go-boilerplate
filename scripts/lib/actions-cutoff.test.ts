import { describe, expect, it } from "vitest";

import {
  callsCommentAction,
  callsReusableWorkflow,
  conditionOf,
  hasCutOffHeading,
  hasJobTimeout,
  reachesCancelled,
  scanWorkflow,
  titleOf,
} from "./actions-cutoff";
import { splitJobs, splitSteps } from "./workflow";

function job(...lines: string[]) {
  return splitJobs(["jobs:", "  a:", ...lines].join("\n")).jobs[0];
}

function step(...lines: string[]) {
  return splitSteps(job("    steps:", ...lines))[0];
}

describe("callsReusableWorkflow", () => {
  describe("正常系", () => {
    it("ジョブ直下の uses: を呼び出しジョブと判定する", () => {
      expect(callsReusableWorkflow(job("    uses: ./.github/workflows/x.yaml"))).toBe(true);
    });

    it("ステップの uses: を呼び出しジョブと誤認しない", () => {
      expect(callsReusableWorkflow(job("    steps:", "      - uses: actions/checkout@v7"))).toBe(false);
    });
  });
});

describe("hasJobTimeout", () => {
  describe("正常系", () => {
    it("ジョブ直下の timeout-minutes を見つける", () => {
      expect(hasJobTimeout(job("    timeout-minutes: 10"))).toBe(true);
    });

    it("ステップの桁に書かれた timeout-minutes をジョブのものと誤認しない", () => {
      expect(hasJobTimeout(job("    steps:", "      - timeout-minutes: 10"))).toBe(false);
    });
  });
});

describe("callsCommentAction", () => {
  describe("正常系", () => {
    it("ステップ先頭キーの uses: を見つける", () => {
      expect(callsCommentAction(step("      - uses: ./.github/actions/upsert-pr-comment"))).toBe(true);
    });

    it("先頭キーでない位置の uses: も見つける", () => {
      expect(
        callsCommentAction(step("      - name: comment", "        uses: ./.github/actions/upsert-pr-comment")),
      ).toBe(true);
    });
  });
});

describe("conditionOf", () => {
  describe("正常系", () => {
    it("同じ行に書かれた if: を行番号付きで読む", () => {
      expect(conditionOf(step("      - if: always()", "        uses: x"))).toEqual({
        line: 4,
        value: "always()",
      });
    });

    it("折り畳みスカラーの続き行を 1 つの式へ繋ぐ", () => {
      const found = conditionOf(
        step("      - if: >-", "          always() &&", "          github.event_name == 'pull_request'", "        uses: x"),
      );

      expect(found?.value).toBe("always() && github.event_name == 'pull_request'");
    });

    it("リテラルスカラーの続き行も繋ぐ", () => {
      expect(conditionOf(step("      - if: |-", "          cancelled()", "        uses: x"))?.value).toBe(
        "cancelled()",
      );
    });
  });

  describe("異常系", () => {
    it("if: の無いステップは null を返す", () => {
      expect(conditionOf(step("      - uses: x"))).toBeNull();
    });

    it("次のキーで続き行の取り込みを止める", () => {
      const found = conditionOf(step("      - if: >-", "          always()", "        uses: x", "        with:"));

      expect(found?.value).toBe("always()");
    });
  });
});

describe("titleOf", () => {
  describe("正常系", () => {
    it("with: 配下の title: を行番号付きで読む", () => {
      const found = titleOf(step("      - uses: x", "        with:", "          title: '## OK'"));

      expect(found).toEqual({ line: 6, value: "'## OK'" });
    });
  });

  describe("異常系", () => {
    it("title: の無いステップは null を返す", () => {
      expect(titleOf(step("      - uses: x", "        with:", "          marker: '<!-- x -->'"))).toBeNull();
    });
  });
});

describe("reachesCancelled", () => {
  describe("正常系", () => {
    it("always() を到達と見なす", () => {
      expect(reachesCancelled("always() && x")).toBe(true);
    });

    it("cancelled() を到達と見なす", () => {
      expect(reachesCancelled("cancelled()")).toBe(true);
    });

    it("空白入りの呼び出しも到達と見なす", () => {
      expect(reachesCancelled("always( )")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("failure() は cancelled で false になるため到達と見なさない", () => {
      expect(reachesCancelled("failure()")).toBe(false);
    });

    it("success() だけの条件を到達と見なさない", () => {
      expect(reachesCancelled("success() && github.event_name == 'pull_request'")).toBe(false);
    });

    it("条件を書かない（暗黙の success()）形を到達と見なさない", () => {
      expect(reachesCancelled("")).toBe(false);
    });

    it("接尾辞が一致するだけの識別子を関数呼び出しと見なさない", () => {
      expect(reachesCancelled("steps.always()")).toBe(true);
      expect(reachesCancelled("notcancelled()")).toBe(false);
    });
  });
});

describe("hasCutOffHeading", () => {
  describe("正常系", () => {
    it("打ち切り時の見出しを持つ title を通す", () => {
      expect(hasCutOffHeading("${{ steps.x.outputs.title || '## ⚠️ X: CUT OFF (no result produced)' }}")).toBe(
        true,
      );
    });
  });

  describe("異常系", () => {
    it("打ち切り時の見出しを持たない title を弾く", () => {
      expect(hasCutOffHeading("${{ steps.x.outputs.title }}")).toBe(false);
    });

    it("大文字小文字の違う表記を通さない（見出しの文言を揺らさない）", () => {
      expect(hasCutOffHeading("## cut off")).toBe(false);
    });
  });
});

const COMMENT_STEP = "      - uses: ./.github/actions/upsert-pr-comment";
const GOOD_IF = "        if: always()";
const GOOD_TITLE = "          title: ${{ steps.x.outputs.title || '## ⚠️ X: CUT OFF (no result produced)' }}";

function workflow(...lines: string[]): string {
  return lines.join("\n");
}

/** 違反の出ない最小のワークフロー。個々のケースは、ここから 1 箇所だけ崩す。 */
function healthy(...extra: string[]): string {
  return workflow(
    "jobs:",
    "  a:",
    "    timeout-minutes: 10",
    "    steps:",
    COMMENT_STEP,
    GOOD_IF,
    "        with:",
    GOOD_TITLE,
    ...extra,
  );
}

describe("scanWorkflow", () => {
  describe("正常系", () => {
    it("timeout と if と title が揃っていれば違反にしない", () => {
      const scan = scanWorkflow("w.yaml", healthy());

      expect(scan.found).toBe(true);
      expect(scan.findings).toEqual([]);
    });

    it("検査したジョブ数とコメントステップ数を数える", () => {
      const scan = scanWorkflow("w.yaml", healthy());

      expect(scan.checkedJobs).toBe(1);
      expect(scan.checkedSteps).toBe(1);
    });

    it("再利用ワークフロー呼び出しのジョブは timeout の検査から外す", () => {
      const scan = scanWorkflow(
        "w.yaml",
        workflow("jobs:", "  a:", "    uses: ./.github/workflows/reusable.yaml"),
      );

      expect(scan.findings).toEqual([]);
      expect(scan.checkedJobs).toBe(0);
    });

    it("コメント投稿を呼ばないステップは if / title を要求しない", () => {
      const scan = scanWorkflow(
        "w.yaml",
        workflow("jobs:", "  a:", "    timeout-minutes: 10", "    steps:", "      - run: echo ok"),
      );

      expect(scan.findings).toEqual([]);
      expect(scan.checkedSteps).toBe(0);
    });
  });

  describe("異常系", () => {
    it("timeout-minutes の無いジョブをジョブ見出しの行で挙げる", () => {
      const scan = scanWorkflow(
        "w.yaml",
        workflow("jobs:", "  a:", "    steps:", COMMENT_STEP, GOOD_IF, "        with:", GOOD_TITLE),
      );

      expect(scan.findings).toHaveLength(1);
      expect(scan.findings[0]).toMatchObject({ file: "w.yaml", line: 2 });
      expect(scan.findings[0].message).toContain("timeout-minutes");
    });

    it("if: の無いコメントステップをステップ先頭の行で挙げる", () => {
      const scan = scanWorkflow(
        "w.yaml",
        workflow("jobs:", "  a:", "    timeout-minutes: 10", "    steps:", COMMENT_STEP, "        with:", GOOD_TITLE),
      );

      expect(scan.findings).toHaveLength(1);
      expect(scan.findings[0].line).toBe(5);
      expect(scan.findings[0].message).toContain("if: がありません");
    });

    it("打ち切りに到達しない if: を if: の行で挙げる", () => {
      const scan = scanWorkflow(
        "w.yaml",
        workflow(
          "jobs:",
          "  a:",
          "    timeout-minutes: 10",
          "    steps:",
          COMMENT_STEP,
          "        if: failure()",
          "        with:",
          GOOD_TITLE,
        ),
      );

      expect(scan.findings).toHaveLength(1);
      expect(scan.findings[0].line).toBe(6);
      expect(scan.findings[0].message).toContain("打ち切りに到達しません");
    });

    it("title: の無いコメントステップをステップ先頭の行で挙げる", () => {
      const scan = scanWorkflow(
        "w.yaml",
        workflow("jobs:", "  a:", "    timeout-minutes: 10", "    steps:", COMMENT_STEP, GOOD_IF),
      );

      expect(scan.findings).toHaveLength(1);
      expect(scan.findings[0].line).toBe(5);
      expect(scan.findings[0].message).toContain("title: がありません");
    });

    it("打ち切り見出しの無い title: を title: の行で挙げる", () => {
      const scan = scanWorkflow(
        "w.yaml",
        workflow(
          "jobs:",
          "  a:",
          "    timeout-minutes: 10",
          "    steps:",
          COMMENT_STEP,
          GOOD_IF,
          "        with:",
          "          title: ## 結果",
        ),
      );

      expect(scan.findings).toHaveLength(1);
      expect(scan.findings[0].line).toBe(8);
      expect(scan.findings[0].message).toContain("打ち切り時の見出しがありません");
    });

    it("同じステップの if: と title: の欠落を両方とも挙げる", () => {
      const scan = scanWorkflow(
        "w.yaml",
        workflow("jobs:", "  a:", "    timeout-minutes: 10", "    steps:", COMMENT_STEP),
      );

      expect(scan.findings).toHaveLength(2);
    });

    it("jobs: を読めなければ違反ゼロではなく読めなかったこととして返す", () => {
      const scan = scanWorkflow("w.yaml", workflow("name: X", "on: push"));

      expect(scan.found).toBe(false);
      expect(scan.findings).toEqual([]);
      expect(scan.checkedJobs).toBe(0);
      expect(scan.checkedSteps).toBe(0);
    });
  });
});
