import fs from "node:fs";
import path from "node:path";

import { ROOT_DIR } from "../lib/runtime";
import { describe, expect, it } from "vitest";

import { replaceOpenapiTermsOfService, replaceReadmeReferences, replaceSonarProject, REPOSITORY_REFERENCE_TARGETS } from "./repository-reference";

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

describe("replaceSonarProject", () => {
  const content = [
    "sonar.projectKey=Tomy-ch_go-boilerplate",
    "sonar.organization=tomy-ch",
    "sonar.projectName=go-boilerplate",
    "sonar.sources=.",
    "",
  ].join("\n");

  describe("正常系", () => {
    it("projectKey を <owner>_<repo>、organization を小文字 owner へ置き換える", () => {
      expect(replaceSonarProject(content, "Example-Org/example-api")).toBe(
        [
          "sonar.projectKey=Example-Org_example-api",
          "sonar.organization=example-org",
          "sonar.projectName=example-api",
          "sonar.sources=.",
          "",
        ].join("\n"),
      );
    });

    it("`$&` を含むリポジトリ名を置換パターンとして解釈しない", () => {
      expect(replaceSonarProject(content, "org/a$&b")).toContain("sonar.projectKey=org_a$&b");
    });
  });

  describe("異常系", () => {
    it("対象キーが無ければ本文を変えない", () => {
      const other = "sonar.sources=.\n";

      expect(replaceSonarProject(other, "org/api")).toBe(other);
    });
  });
});

describe("REPOSITORY_REFERENCE_TARGETS", () => {
  describe("正常系", () => {
    it("README の英日対と OpenAPI と Sonar 設定を対象にする", () => {
      expect(REPOSITORY_REFERENCE_TARGETS.readmeFiles).toEqual(["README.md", "README.ja.md"]);
      expect(REPOSITORY_REFERENCE_TARGETS.openapiFile).toBe("openapi/openapi.yaml");
      expect(REPOSITORY_REFERENCE_TARGETS.sonarFile).toBe("sonar-project.properties");
    });

    it("挙げた対象がすべて実在する", () => {
      const targets = [
        ...REPOSITORY_REFERENCE_TARGETS.readmeFiles,
        REPOSITORY_REFERENCE_TARGETS.openapiFile,
        REPOSITORY_REFERENCE_TARGETS.sonarFile,
      ];

      for (const target of targets) {
        expect(fs.existsSync(path.join(ROOT_DIR, target)), target).toBe(true);
      }
    });
  });
});
