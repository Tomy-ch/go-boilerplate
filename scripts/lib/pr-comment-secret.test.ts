import { describe, expect, it } from "vitest";

import { describeSecret, secretReferences, usesCommentAction } from "./pr-comment-secret";
import { splitJobs } from "./workflow";

function line(text: string, number = 1) {
  return { number, text };
}

function job(...lines: string[]) {
  return splitJobs(["jobs:", "  a:", ...lines].join("\n")).jobs[0];
}

describe("usesCommentAction", () => {
  describe("正常系", () => {
    it("PR コメント投稿アクションを使うジョブを見分ける", () => {
      expect(usesCommentAction(job("    steps:", "      - uses: ./.github/actions/upsert-pr-comment"))).toBe(
        true,
      );
    });

    it("使っていないジョブを対象外にする", () => {
      expect(usesCommentAction(job("    steps:", "      - uses: actions/checkout@v7"))).toBe(false);
    });
  });
});

describe("secretReferences", () => {
  describe("正常系", () => {
    it("式の中の secrets 参照を行番号付きで返す", () => {
      expect(secretReferences(line("  token: ${{ secrets.NPM_TOKEN }}", 12))).toEqual([
        { number: 12, name: "NPM_TOKEN" },
      ]);
    });

    it("添字表記の参照も拾う", () => {
      expect(secretReferences(line("  token: ${{ secrets['NPM_TOKEN'] }}"))).toEqual([
        { number: 1, name: "NPM_TOKEN" },
      ]);
    });

    it("コンテキスト全体の参照は名前なしで返す", () => {
      expect(secretReferences(line("  all: ${{ toJSON(secrets) }}"))).toEqual([
        { number: 1, name: undefined },
      ]);
    });

    it("1 行に複数の参照があれば全て返す", () => {
      const found = secretReferences(line("  x: ${{ secrets.A }}${{ secrets.B }}"));

      expect(found.map((reference) => reference.name)).toEqual(["A", "B"]);
    });

    it("GITHUB_TOKEN は投稿に必要なので許可する", () => {
      expect(secretReferences(line("  github-token: ${{ secrets.GITHUB_TOKEN }}"))).toEqual([]);
    });

    it("式の外に現れる secrets という語に反応しない", () => {
      expect(secretReferences(line("  # secrets.NPM_TOKEN を渡してはいけない"))).toEqual([]);
    });

    it("secret を含まない行は空を返す", () => {
      expect(secretReferences(line("  runs-on: ubuntu-latest"))).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("複数行にまたがる式でも中身を見る", () => {
      expect(secretReferences(line("  x: ${{ secrets\n        .NPM_TOKEN }}"))).toEqual([
        { number: 1, name: "NPM_TOKEN" },
      ]);
    });

    it("閉じていない式は検査対象にしない（構文誤りは actionlint の担当）", () => {
      expect(secretReferences(line("  x: ${{ secrets.NPM_TOKEN"))).toEqual([]);
    });

    it("GITHUB_TOKEN と別 secret が同居しても別 secret だけを返す", () => {
      const found = secretReferences(line("  x: ${{ secrets.GITHUB_TOKEN }} ${{ secrets.OTHER }}"));

      expect(found.map((reference) => reference.name)).toEqual(["OTHER"]);
    });

    it("接頭辞が一致するだけの語を secrets 参照にしない", () => {
      expect(secretReferences(line("  x: ${{ mysecrets.TOKEN }}"))).toEqual([]);
    });
  });
});

describe("describeSecret", () => {
  describe("正常系", () => {
    it("名前がある参照はコード表記で示す", () => {
      expect(describeSecret("NPM_TOKEN")).toBe("`secrets.NPM_TOKEN`");
    });

    it("名前が無い参照はコンテキスト全体として示す", () => {
      expect(describeSecret(undefined)).toBe("`secrets` コンテキスト全体");
    });
  });
});
