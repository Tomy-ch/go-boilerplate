import { describe, expect, it } from "vitest";

import {
  EXCLUDED_DIRECTORIES,
  ensureModuleArguments,
  isReplacementTarget,
  replaceModuleOccurrences,
} from "./module-replace";

describe("ensureModuleArguments", () => {
  describe("正常系", () => {
    it("単純なプロジェクト名の組を通す", () => {
      expect(() => ensureModuleArguments("go-boilerplate", "example-api")).not.toThrow();
    });

    it("ドット・アンダースコアを含む名前を通す", () => {
      expect(() => ensureModuleArguments("go_boilerplate.v1", "example.api_v2")).not.toThrow();
    });
  });

  describe("異常系", () => {
    // モジュールパス全体（`github.com/org/name`）を渡されると、置換後に
    // `github.com/org/github.com/org/name` のような壊れた import が量産される。
    it("スラッシュを含むモジュールパスを拒否する", () => {
      expect(() => ensureModuleArguments("github.com/org/go-boilerplate", "example-api")).toThrow(
        "旧モジュール名",
      );
    });

    it("新モジュール名側の不正も拒否する", () => {
      expect(() => ensureModuleArguments("go-boilerplate", "example api")).toThrow("新モジュール名");
    });

    it("空文字を拒否する", () => {
      expect(() => ensureModuleArguments("", "example-api")).toThrow("旧モジュール名");
    });

    // 同一名を通すと全ファイルが「変更なし」で終わり、置換したつもりの成功報告だけが残る。
    it("新旧が同一の指定を拒否する", () => {
      expect(() => ensureModuleArguments("go-boilerplate", "go-boilerplate")).toThrow("同一です");
    });
  });
});

describe("isReplacementTarget", () => {
  describe("正常系", () => {
    it("モジュール名が現れるファイル種別を対象にする", () => {
      for (const target of [
        "internal/domain/user/user.go",
        "go.mod",
        "docker-compose.yaml",
        ".github/workflows/ci.yml",
        "README.md",
        "scripts/tool.js",
        "scripts/tool.cjs",
        "scripts/tool.mjs",
        "package.json",
        "docs-viewer/index.html",
      ]) {
        expect(isReplacementTarget(target)).toBe(true);
      }
    });

    it("拡張子を持たない Dockerfile を basename で拾う", () => {
      expect(isReplacementTarget("docker/api/Dockerfile")).toBe(true);
    });
  });

  describe("異常系", () => {
    // 生成物を書き換えると、再生成で戻る変更をコミットに載せることになる。
    it("生成物を対象から外す", () => {
      for (const generated of [
        "internal/controller/handler/gen/server.gen.go",
        "internal/infrastructure/rdb/sqlc/gen/user_repository.gen.sql.go",
        "openapi/openapi.gen.yaml",
        "internal/usecase/user/mock_user.go",
        "internal/domain/user/user_repository_mock.go",
      ]) {
        expect(isReplacementTarget(generated)).toBe(false);
      }
    });

    // `openapi.gen.yaml` はセパレータ込みで判定する。接尾辞だけで見ると
    // `my-openapi.gen.yaml` のような別ファイルまで巻き込む。
    it("ルート直下でない openapi.gen.yaml と同名接尾辞を区別する", () => {
      expect(isReplacementTarget("openapi/openapi.gen.yaml")).toBe(false);
      expect(isReplacementTarget("openapi/my-openapi.gen.yaml")).toBe(true);
    });

    it("docs 配下と scripts/setup 配下を対象から外す", () => {
      expect(isReplacementTarget("docs/architecture.md")).toBe(false);
      expect(isReplacementTarget("scripts/setup/replace-module.ts")).toBe(false);
    });

    // 接頭辞はセパレータまで含めて判定する。`docs` だけで見ると `docs-viewer/` や
    // `documentation.md` まで除外され、置換の取りこぼしになる。
    it("接頭辞の部分一致で無関係なパスを除外しない", () => {
      expect(isReplacementTarget("docs-viewer/src/main.html")).toBe(true);
      expect(isReplacementTarget("scripts/setup-notes.md")).toBe(true);
    });

    it("対象外の拡張子を弾く", () => {
      for (const other of [
        "database/migrations/000001_create.up.sql",
        "mise.toml",
        "LICENSE",
        "storage/seed/products/item.png",
      ]) {
        expect(isReplacementTarget(other)).toBe(false);
      }
    });

    it("Dockerfile の派生名は basename 一致でないため対象外", () => {
      expect(isReplacementTarget("docker/api/Dockerfile.dev")).toBe(false);
    });
  });
});

describe("replaceModuleOccurrences", () => {
  describe("正常系", () => {
    it("出現箇所をすべて置き換えて件数を返す", () => {
      const replaced = replaceModuleOccurrences(
        'import "go-boilerplate/internal/domain"\nimport "go-boilerplate/pkg/log"\n',
        "go-boilerplate",
        "example-api",
      );

      expect(replaced).toEqual({
        content: 'import "example-api/internal/domain"\nimport "example-api/pkg/log"\n',
        occurrences: 2,
      });
    });

    // 置換を `String.replace` の文字列指定で書くと `$&` が「マッチ全体」として解釈され、
    // 記号を含むモジュール名がそのまま入らない。分割・結合ならその解釈自体が起きない。
    it("置換パターンに見える文字列をそのまま埋め込む", () => {
      const replaced = replaceModuleOccurrences("mod go-boilerplate", "go-boilerplate", "a$&b$'c");

      expect(replaced?.content).toBe("mod a$&b$'c");
    });

    it("正規表現メタ文字を含む旧モジュール名を literal として扱う", () => {
      const replaced = replaceModuleOccurrences("x a.c y abc z", "a.c", "NEW");

      expect(replaced).toEqual({ content: "x NEW y abc z", occurrences: 1 });
    });
  });

  describe("異常系", () => {
    it("出現しなければ null を返す（書き込みを起こさない）", () => {
      expect(replaceModuleOccurrences("package main\n", "go-boilerplate", "example-api")).toBeNull();
    });
  });
});

describe("EXCLUDED_DIRECTORIES", () => {
  describe("異常系", () => {
    // vendor / node_modules を走査すると、書き換えてはいけない依存の中身まで
    // 置換対象になり、実行時間も桁違いになる。
    it("依存・作業用ディレクトリを走査対象から外す", () => {
      for (const excluded of ["vendor", "node_modules", "tmp", ".git"]) {
        expect(EXCLUDED_DIRECTORIES.has(excluded)).toBe(true);
      }
    });
  });
});
