import { describe, expect, it } from "vitest";

import { splitJobs, splitSteps, usesActionPattern } from "./workflow";

function workflow(...lines: string[]): string {
  return lines.join("\n");
}

describe("splitJobs", () => {
  describe("正常系", () => {
    it("ジョブ id と見出しの行番号を取り出す", () => {
      const { found, jobs } = splitJobs(workflow("name: X", "jobs:", "  build:", "    runs-on: x"));

      expect(found).toBe(true);
      expect(jobs).toHaveLength(1);
      expect(jobs[0]).toMatchObject({ id: "build", number: 3 });
    });

    it("ジョブ本文を行番号付きで持つ", () => {
      const { jobs } = splitJobs(workflow("jobs:", "  build:", "    runs-on: x"));

      expect(jobs[0].lines).toEqual([{ number: 3, text: "    runs-on: x" }]);
    });

    it("引用符付きのジョブ id を読む", () => {
      const single = splitJobs(workflow("jobs:", "  'build':", "    runs-on: x"));
      const double = splitJobs(workflow("jobs:", '  "build":', "    runs-on: x"));

      expect([single.jobs[0].id, double.jobs[0].id]).toEqual(["build", "build"]);
    });

    it("行末コメント付きのジョブ id を読む", () => {
      const { jobs } = splitJobs(workflow("jobs:", "  build: # メイン", "    runs-on: x"));

      expect(jobs[0].id).toBe("build");
    });

    it("複数ジョブを順に切り出す", () => {
      const { jobs } = splitJobs(workflow("jobs:", "  a:", "    runs-on: x", "  b:", "    runs-on: y"));

      expect(jobs.map((job) => job.id)).toEqual(["a", "b"]);
    });

    it("jobs: より前の行を preamble に入れる", () => {
      const { preamble } = splitJobs(workflow("env:", "  TOKEN: x", "jobs:", "  a:", "    runs-on: x"));

      expect(preamble).toEqual([
        { number: 1, text: "env:" },
        { number: 2, text: "  TOKEN: x" },
      ]);
    });

    it("jobs: の後ろに戻ったトップレベルキーも preamble に入れる", () => {
      const { jobs, preamble } = splitJobs(
        workflow("jobs:", "  a:", "    runs-on: x", "env:", "  TOKEN: y"),
      );

      expect(jobs).toHaveLength(1);
      expect(preamble).toEqual([
        { number: 4, text: "env:" },
        { number: 5, text: "  TOKEN: y" },
      ]);
    });

    it("桁 0 のコメント行では jobs: を打ち切らない", () => {
      const { jobs } = splitJobs(
        workflow("jobs:", "  a:", "    runs-on: x", "# 区切り", "  b:", "    runs-on: y"),
      );

      expect(jobs.map((job) => job.id)).toEqual(["a", "b"]);
    });
  });

  describe("異常系", () => {
    it("jobs: が無いことを空のジョブ列と区別する", () => {
      const { found, jobs, preamble } = splitJobs(workflow("name: X", "on: push"));

      expect(found).toBe(false);
      expect(jobs).toEqual([]);
      expect(preamble).toHaveLength(2);
    });

    it("ジョブ見出しより前の行は捨てる（どのジョブにも属さない）", () => {
      const { jobs } = splitJobs(workflow("jobs:", "  # 説明だけ", "  a:", "    runs-on: x"));

      expect(jobs[0].lines).toEqual([{ number: 4, text: "    runs-on: x" }]);
    });

    it("ジョブ id に使えない文字を含む見出しをジョブにしない", () => {
      const { jobs } = splitJobs(workflow("jobs:", "  a.b:", "    runs-on: x"));

      expect(jobs).toEqual([]);
    });
  });
});

describe("splitSteps", () => {
  describe("正常系", () => {
    it("`- ` を境にステップを切り出す", () => {
      const { jobs } = splitJobs(
        workflow("jobs:", "  a:", "    steps:", "      - uses: x", "      - run: y"),
      );

      expect(splitSteps(jobs[0]).map((step) => step.number)).toEqual([4, 5]);
    });

    it("ステップの続き行を同じステップに含める", () => {
      const { jobs } = splitJobs(
        workflow("jobs:", "  a:", "    steps:", "      - uses: x", "        with:", "          k: v"),
      );

      expect(splitSteps(jobs[0])[0].lines).toHaveLength(3);
    });

    it("steps: と同じかそれより浅いキーでステップ列を終える", () => {
      const { jobs } = splitJobs(
        workflow("jobs:", "  a:", "    steps:", "      - uses: x", "    timeout-minutes: 10"),
      );

      expect(splitSteps(jobs[0])[0].lines).toHaveLength(1);
    });
  });

  describe("異常系", () => {
    it("最初のステップより前の行はどのステップにも入れない", () => {
      const { jobs } = splitJobs(
        workflow("jobs:", "  a:", "    steps:", "      # 先頭の説明", "      - uses: x"),
      );

      expect(splitSteps(jobs[0])).toEqual([
        { number: 5, lines: [{ number: 5, text: "      - uses: x" }] },
      ]);
    });

    it("steps: を持たないジョブは空を返す", () => {
      const { jobs } = splitJobs(workflow("jobs:", "  a:", "    uses: ./x.yaml"));

      expect(splitSteps(jobs[0])).toEqual([]);
    });
  });
});

describe("usesActionPattern", () => {
  describe("正常系", () => {
    it("引用符の有無に関わらず当たる", () => {
      const pattern = usesActionPattern("./.github/actions/upsert-pr-comment", false);

      expect(pattern.test("        uses: ./.github/actions/upsert-pr-comment")).toBe(true);
      expect(pattern.test("        uses: './.github/actions/upsert-pr-comment'")).toBe(true);
      expect(pattern.test('        uses: "./.github/actions/upsert-pr-comment"')).toBe(true);
    });

    it("行末コメント付きでも当たる", () => {
      const pattern = usesActionPattern("./.github/actions/upsert-pr-comment", false);

      expect(pattern.test("        uses: ./.github/actions/upsert-pr-comment # 投稿")).toBe(true);
    });

    it("桁を固定した形はステップ先頭とキー位置の両方に当たる", () => {
      const pattern = usesActionPattern("./.github/actions/upsert-pr-comment", true);

      expect(pattern.test("      - uses: ./.github/actions/upsert-pr-comment")).toBe(true);
      expect(pattern.test("        uses: ./.github/actions/upsert-pr-comment")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("パス中のドットを任意文字として扱わない", () => {
      const pattern = usesActionPattern("./.github/actions/upsert-pr-comment", false);

      expect(pattern.test("        uses: ./Xgithub/actions/upsert-pr-comment")).toBe(false);
    });

    it("桁を固定した形はステップ以外の桁に当たらない", () => {
      const pattern = usesActionPattern("./.github/actions/upsert-pr-comment", true);

      expect(pattern.test("    uses: ./.github/actions/upsert-pr-comment")).toBe(false);
    });
  });
});
