import { describe, expect, it } from "vitest";

import { replaceOpenapiTermsOfService, replaceReadmeReferences } from "./repository-reference";

const README = [
  "# go-boilerplate",
  "",
  "![Go Version](https://img.shields.io/github/go-mod/go-version/Tomy-ch/go-boilerplate)",
  "![License](https://img.shields.io/github/license/Tomy-ch/go-boilerplate)",
  "",
  "```bash",
  "git clone https://github.com/Tomy-ch/go-boilerplate.git",
  "cd go-boilerplate",
  "```",
  "",
].join("\n");

describe("replaceReadmeReferences", () => {
  describe("正常系", () => {
    it("見出し・バッジ・clone URL・cd をまとめて置き換える", () => {
      expect(replaceReadmeReferences(README, "example-org/example-api")).toBe(
        [
          "# example-api",
          "",
          "![Go Version](https://img.shields.io/github/go-mod/go-version/example-org/example-api)",
          "![License](https://img.shields.io/github/license/example-org/example-api)",
          "",
          "```bash",
          "git clone https://github.com/example-org/example-api.git",
          "cd example-api",
          "```",
          "",
        ].join("\n"),
      );
    });

    // バッジは README 内に何度も現れる（冒頭の一覧と本文中の再掲）。1 件しか置き換えないと
    // 他人のリポジトリを指すバッジが残り、状態が混ざったまま公開される。
    it("バッジ URL は出現するすべてを置き換える", () => {
      const content =
        "![a](https://img.shields.io/github/license/Tomy-ch/go-boilerplate)\n" +
        "![b](https://img.shields.io/github/license/Tomy-ch/go-boilerplate)\n";

      expect(replaceReadmeReferences(content, "org/api")).toBe(
        "![a](https://img.shields.io/github/license/org/api)\n" +
          "![b](https://img.shields.io/github/license/org/api)\n",
      );
    });

    it("clone URL は出現するすべてを置き換える", () => {
      const content =
        "https://github.com/a/b.git\nhttps://github.com/c/d.git\n";

      expect(replaceReadmeReferences(content, "org/api")).toBe(
        "https://github.com/org/api.git\nhttps://github.com/org/api.git\n",
      );
    });
  });

  describe("異常系", () => {
    // 見出しと cd は最初の 1 件だけ。全置換にすると、手順中の別ディレクトリへの
    // `cd` や本文の下位見出しまでリポジトリ名に化ける。
    it("2 つ目以降の見出しと cd を書き換えない", () => {
      const content = "# go-boilerplate\n# 別の見出し\ncd go-boilerplate\ncd internal\n";

      expect(replaceReadmeReferences(content, "org/api")).toBe(
        "# api\n# 別の見出し\ncd api\ncd internal\n",
      );
    });

    it("下位見出しを書き換えない", () => {
      expect(replaceReadmeReferences("## セットアップ\n", "org/api")).toBe("## セットアップ\n");
    });

    // `.git` で終わる URL だけを clone URL と見なす。これを外すと、本文中の
    // issue / PR / ドキュメントへのリンクまで clone URL に書き換わる。
    it(".git で終わらない github.com のリンクを書き換えない", () => {
      const content = "See https://github.com/Tomy-ch/go-boilerplate/issues/1 for details.\n";

      expect(replaceReadmeReferences(content, "org/api")).toBe(content);
    });

    // URL の終端を `)` と空白で切らないと、Markdown リンクの閉じ括弧まで飲み込んで
    // リンク記法が壊れる。
    it("Markdown リンクの閉じ括弧を飲み込まない", () => {
      const content = "[badge](https://img.shields.io/github/license/Tomy-ch/go-boilerplate) 続き\n";

      expect(replaceReadmeReferences(content, "org/api")).toBe(
        "[badge](https://img.shields.io/github/license/org/api) 続き\n",
      );
    });

    it("置換パターンに見えるリポジトリ名をそのまま書き込む", () => {
      expect(replaceReadmeReferences("# old\n", "org/a$&b")).toBe("# a$&b\n");
    });
  });
});

describe("replaceOpenapiTermsOfService", () => {
  describe("正常系", () => {
    it("termsOfService の URL を置き換える", () => {
      const content = "info:\n  termsOfService: https://github.com/Tomy-ch/go-boilerplate\n";

      expect(replaceOpenapiTermsOfService(content, "org/api")).toBe(
        "info:\n  termsOfService: https://github.com/org/api\n",
      );
    });
  });

  describe("異常系", () => {
    // インデント 2 の termsOfService 行に固定する。緩めると、説明文中の
    // github.com への言及や別階層のキーまで巻き込む。
    it("インデントの異なる termsOfService を書き換えない", () => {
      const content = "termsOfService: https://github.com/a/b\n";

      expect(replaceOpenapiTermsOfService(content, "org/api")).toBe(content);
    });

    it("github.com 以外の termsOfService を書き換えない", () => {
      const content = "  termsOfService: https://example.com/terms\n";

      expect(replaceOpenapiTermsOfService(content, "org/api")).toBe(content);
    });
  });
});
