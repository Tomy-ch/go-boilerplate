import { describe, expect, it } from "vitest";

import {
  callsCommentAction,
  callsReusableWorkflow,
  conditionOf,
  hasCutOffHeading,
  hasJobTimeout,
  reachesCancelled,
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
