import fs from "node:fs";
import path from "node:path";

import { ROOT_DIR } from "../lib/runtime";
import { describe, expect, it } from "vitest";

import {
  isEnvFile,
  replaceCopilotTitle,
  replaceEnvAppName,
  replaceOpenapiTitle, APP_METADATA_TARGETS } from "./app-metadata";

describe("isEnvFile", () => {
  describe("正常系", () => {
    it("env ディレクトリの .env 系ファイルを対象にする", () => {
      for (const name of [".env", ".env.ci", ".env.dev", ".env.prd", ".env.stg"]) {
        expect(isEnvFile(name)).toBe(true);
      }
    });
  });

  describe("異常系", () => {
    // env/ には .env 系以外のファイル（README 等）も置かれうる。前方一致を外すと
    // それらの先頭行を APP_NAME 置換の対象として開いてしまう。
    it(".env で始まらない名前を対象外にする", () => {
      for (const name of ["README.md", "env.sample", "sample.env"]) {
        expect(isEnvFile(name)).toBe(false);
      }
    });
  });
});

describe("replaceEnvAppName", () => {
  describe("正常系", () => {
    it("APP_NAME 行だけを置き換える", () => {
      const content = "DB_HOST=db\nAPP_NAME=go-boilerplate\nDB_PORT=5432\n";

      expect(replaceEnvAppName(content, "Example API")).toBe(
        "DB_HOST=db\nAPP_NAME=Example API\nDB_PORT=5432\n",
      );
    });

    it("先頭行の APP_NAME も対象にする", () => {
      expect(replaceEnvAppName("APP_NAME=old\n", "new")).toBe("APP_NAME=new\n");
    });
  });

  describe("異常系", () => {
    // 行頭固定を外すと `MOCK_APP_NAME=` のような別キーの一部に当たり、
    // 無関係な設定値が壊れる。
    it("別キーの一部に当てない", () => {
      const content = "MOCK_APP_NAME=mock\nOTHER_APP_NAME=other\n";

      expect(replaceEnvAppName(content, "Example API")).toBeNull();
    });

    it("最初の 1 行だけを置き換える", () => {
      const content = "APP_NAME=a\nAPP_NAME=b\n";

      expect(replaceEnvAppName(content, "c")).toBe("APP_NAME=c\nAPP_NAME=b\n");
    });

    // 文字列置換だと `$&` が「マッチした行全体」に展開され、`APP_NAME=APP_NAME=old` になる。
    it("置換パターンに見えるアプリ名をそのまま書き込む", () => {
      expect(replaceEnvAppName("APP_NAME=old\n", "a$&b$`c")).toBe("APP_NAME=a$&b$`c\n");
    });

    it("APP_NAME 行が無ければ null を返す（書き込みを起こさない）", () => {
      expect(replaceEnvAppName("DB_HOST=db\n", "Example API")).toBeNull();
    });
  });
});

describe("replaceOpenapiTitle", () => {
  describe("正常系", () => {
    it("info 直下の title 行を置き換える", () => {
      const content = "openapi: 3.1.0\ninfo:\n  title: go-boilerplate\n  version: 1.0.0\n";

      expect(replaceOpenapiTitle(content, "Example API")).toBe(
        "openapi: 3.1.0\ninfo:\n  title: Example API\n  version: 1.0.0\n",
      );
    });
  });

  describe("異常系", () => {
    // インデント 2 に固定しないと、paths 配下の深い階層にある title:
    // （スキーマやレスポンスの説明）まで書き換える。
    it("より深いインデントの title を対象にしない", () => {
      const content = "paths:\n  /v1/users:\n    get:\n      title: 一覧\n";

      expect(replaceOpenapiTitle(content, "Example API")).toBeNull();
    });

    it("インデントの無い title を対象にしない", () => {
      expect(replaceOpenapiTitle("title: root\n", "Example API")).toBeNull();
    });

    it("最初の 1 行だけを置き換える", () => {
      const content = "info:\n  title: a\ncomponents:\n  title: b\n";

      expect(replaceOpenapiTitle(content, "c")).toBe("info:\n  title: c\ncomponents:\n  title: b\n");
    });

    it("置換パターンに見えるタイトルをそのまま書き込む", () => {
      expect(replaceOpenapiTitle("  title: old\n", "a$&b")).toBe("  title: a$&b\n");
    });

    it("title 行が無ければ null を返す（書き込みを起こさない）", () => {
      expect(replaceOpenapiTitle("openapi: 3.1.0\n", "Example API")).toBeNull();
    });
  });
});

describe("replaceCopilotTitle", () => {
  describe("正常系", () => {
    it("先頭見出しを置き換える", () => {
      expect(replaceCopilotTitle("# go-boilerplate\n\n本文\n", "example-api Copilot")).toBe(
        "# example-api Copilot\n\n本文\n",
      );
    });
  });

  describe("異常系", () => {
    // `# ` の空白を落とすと `## 見出し` にも当たり、下位見出しがタイトルに化ける。
    it("下位見出しを対象にしない", () => {
      expect(replaceCopilotTitle("## 見出し\n本文\n", "Example")).toBeNull();
    });

    it("最初の見出しだけを置き換える", () => {
      expect(replaceCopilotTitle("# a\n本文\n# b\n", "c")).toBe("# c\n本文\n# b\n");
    });

    it("置換パターンに見えるタイトルをそのまま書き込む", () => {
      expect(replaceCopilotTitle("# old\n", "a$1b")).toBe("# a$1b\n");
    });

    it("見出しが無ければ null を返す（書き込みを起こさない）", () => {
      expect(replaceCopilotTitle("本文だけ\n", "Example")).toBeNull();
    });
  });
});

describe("APP_METADATA_TARGETS", () => {
  describe("正常系", () => {
    it("env ディレクトリと OpenAPI と Copilot 指示書を対象にする", () => {
      expect(APP_METADATA_TARGETS).toEqual({
        envDir: "env",
        openapiFile: "openapi/openapi.yaml",
        copilotInstructionsFile: ".github/copilot-instructions.md",
      });
    });

    it("挙げた対象がすべて実在する", () => {
      for (const target of Object.values(APP_METADATA_TARGETS)) {
        expect(fs.existsSync(path.join(ROOT_DIR, target)), target).toBe(true);
      }
    });
  });
});
