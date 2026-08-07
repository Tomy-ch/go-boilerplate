import { describe, expect, it } from "vitest";

import { ensureCodeOwners, ensureFourDigitYear, ensureRepositoryReference } from "./validators";

describe("ensureRepositoryReference", () => {
  describe("正常系", () => {
    it("<owner>/<repo> を通す", () => {
      expect(() => ensureRepositoryReference("example-org/example-api")).not.toThrow();
    });

    it("リポジトリ名のドット・アンダースコアを通す", () => {
      expect(() => ensureRepositoryReference("org/example_api.v2")).not.toThrow();
    });
  });

  describe("異常系", () => {
    // owner/repo を要求しないと、README のバッジ URL に owner だけ・URL 全体などが
    // そのまま埋め込まれ、リンク切れのバッジが第一印象の画面に残る。
    it("owner を欠いた指定を拒否する", () => {
      expect(() => ensureRepositoryReference("example-api")).toThrow("<owner>/<repo>");
    });

    it("スラッシュを 2 つ含む指定を拒否する", () => {
      expect(() => ensureRepositoryReference("github.com/org/repo")).toThrow("<owner>/<repo>");
    });

    // 末尾を固定しないと `org/repo/sub` や余分な語の付いた値が通り、
    // バッジ URL や clone URL に余りがそのまま連結される。
    it("repo の後ろに余分が続く指定を拒否する", () => {
      expect(() => ensureRepositoryReference("org/repo/sub")).toThrow("<owner>/<repo>");
      expect(() => ensureRepositoryReference("org/repo (fork)")).toThrow("<owner>/<repo>");
    });

    it("URL 全体を拒否する", () => {
      expect(() => ensureRepositoryReference("https://github.com/org/repo")).toThrow(
        "<owner>/<repo>",
      );
    });

    it("owner の先頭がハイフンの指定を拒否する", () => {
      expect(() => ensureRepositoryReference("-org/repo")).toThrow("<owner>/<repo>");
    });

    it("空文字を拒否する", () => {
      expect(() => ensureRepositoryReference("")).toThrow("<owner>/<repo>");
    });
  });
});

describe("ensureFourDigitYear", () => {
  describe("正常系", () => {
    it("4 桁の西暦を通す", () => {
      expect(() => ensureFourDigitYear("2026")).not.toThrow();
    });
  });

  describe("異常系", () => {
    it("桁数が異なる年を拒否する", () => {
      expect(() => ensureFourDigitYear("226")).toThrow("4 桁");
      expect(() => ensureFourDigitYear("20261")).toThrow("4 桁");
    });

    it("数字以外を含む年を拒否する", () => {
      expect(() => ensureFourDigitYear("20x6")).toThrow("4 桁");
    });

    // `^` `$` を欠くと "2026年" のような前後の付いた値が通り、LICENSE 行に混ざる。
    it("前後に文字が付いた年を拒否する", () => {
      expect(() => ensureFourDigitYear(" 2026")).toThrow("4 桁");
      expect(() => ensureFourDigitYear("2026年")).toThrow("4 桁");
    });
  });
});

describe("ensureCodeOwners", () => {
  describe("正常系", () => {
    it("@user 形式を通す", () => {
      expect(() => ensureCodeOwners(["@octocat"])).not.toThrow();
    });

    it("@org/team 形式を通す", () => {
      expect(() => ensureCodeOwners(["@example-org/tech-leads"])).not.toThrow();
    });

    it("メールアドレス形式を通す", () => {
      expect(() => ensureCodeOwners(["owner@example.com"])).not.toThrow();
    });

    it("複数指定をすべて検査する", () => {
      expect(() =>
        ensureCodeOwners(["@octocat", "@example-org/security", "owner@example.com"]),
      ).not.toThrow();
    });
  });

  describe("異常系", () => {
    // 所有者ゼロで通すと CODEOWNERS の全ルールが所有者無しに書き換わり、
    // レビュー必須が黙って外れる。
    it("空の指定を拒否する", () => {
      expect(() => ensureCodeOwners([])).toThrow("1 つ以上");
    });

    it("@ の無いハンドルを拒否する", () => {
      expect(() => ensureCodeOwners(["octocat"])).toThrow("不正です");
    });

    it("空白を含むハンドルを拒否する", () => {
      expect(() => ensureCodeOwners(["@example org"])).toThrow("不正です");
    });

    it("ドメイン部にドットの無いメールを拒否する", () => {
      expect(() => ensureCodeOwners(["owner@localhost"])).toThrow("不正です");
    });

    it("末尾がハイフンのハンドルを拒否する", () => {
      expect(() => ensureCodeOwners(["@octocat-"])).toThrow("不正です");
    });

    it("1 件でも不正なら全体を拒否する", () => {
      expect(() => ensureCodeOwners(["@octocat", "bad owner"])).toThrow("不正です");
    });
  });
});
