import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

import { listFilesRecursive, toRelativePath } from "../lib/file-utils";
import { ROOT_DIR } from "../lib/runtime";
import {
  BOILERPLATE_DELETE_FILES,
  BOILERPLATE_MARKER,
  BOILERPLATE_PROSE_MARKERS,
  EXCLUDED_DIRECTORIES,
  EXCLUDED_PATH_PREFIXES,
  MARKER_LITERAL_FILES,
  containsBoilerplateMarker,
  isScanTarget,
} from "./targets";

/** 走査対象ファイルの絶対パス。除去側と同じ経路で列挙する。 */
function scanTargets(): string[] {
  return listFilesRecursive(ROOT_DIR, {
    excludedDirectories: EXCLUDED_DIRECTORIES,
    shouldIncludeFile: (entryPath) => isScanTarget(toRelativePath(entryPath)),
  });
}

/** 宣言したファイルの実際の中身。 */
function read(relativePath: string): string {
  return fs.readFileSync(path.join(ROOT_DIR, relativePath), "utf8");
}

describe("isScanTarget", () => {
  describe("正常系", () => {
    it("除外に当たらないパスを対象にする", () => {
      expect(isScanTarget("docs/adr/README.md")).toBe(true);
      expect(isScanTarget("README.md")).toBe(true);
    });

    // 拡張子でも名前でも絞らない。絞り込みは「マーカーを書いたのに除去されない」を静かに作る。
    it("拡張子を持たないファイルも対象にする", () => {
      expect(isScanTarget("makefile")).toBe(true);
    });

    it("Windows 形式の区切りでも除外を判定する", () => {
      expect(isScanTarget("docs\\portal\\guides\\index.md")).toBe(false);
    });
  });

  describe("異常系", () => {
    // 生成物はマーカーを持っていても再生成で戻るため、除去の対象にしてはいけない。
    it("生成物の接頭辞を対象から外す", () => {
      for (const prefix of EXCLUDED_PATH_PREFIXES) {
        expect(isScanTarget(`${prefix}anything.md`), prefix).toBe(false);
      }
    });

    // 接頭辞一致なので、名前が途中まで同じ別ディレクトリを巻き込まないことを確かめる。
    it("接頭辞が途中まで一致するだけのパスは外さない", () => {
      expect(isScanTarget("docs/portal/manifest.yaml")).toBe(true);
      expect(isScanTarget("docs/coverage-policy.md")).toBe(true);
    });
  });
});

describe("EXCLUDED_DIRECTORIES", () => {
  describe("正常系", () => {
    // 依存の取得物を走査すると、他人のコードのコメントを誤って落としうる。
    it("依存の取得物と VCS の内部を挙げている", () => {
      expect(EXCLUDED_DIRECTORIES.has("node_modules")).toBe(true);
      expect(EXCLUDED_DIRECTORIES.has("vendor")).toBe(true);
      expect(EXCLUDED_DIRECTORIES.has(".git")).toBe(true);
    });
  });

  describe("異常系", () => {
    // マーカーを持つ本文が実際に居るディレクトリを外すと、除去が静かに素通りする。
    it("本文の居るディレクトリを外していない", () => {
      expect(EXCLUDED_DIRECTORIES.has("docs")).toBe(false);
      expect(EXCLUDED_DIRECTORIES.has("internal")).toBe(false);
      expect(EXCLUDED_DIRECTORIES.has("scripts")).toBe(false);
    });
  });
});

describe("containsBoilerplateMarker", () => {
  describe("正常系", () => {
    it("コメント記号を伴うマーカーを検出する", () => {
      expect(containsBoilerplateMarker("<!-- boilerplate-only:begin -->")).toBe(true);
      expect(containsBoilerplateMarker("x # boilerplate-only:line")).toBe(true);
    });
  });

  describe("異常系", () => {
    // コメント記号を必須にしないと、規約を説明している散文を境界と誤認する。
    it("コメント記号の無い同名トークンを検出しない", () => {
      expect(containsBoilerplateMarker('grep -rn "boilerplate-only:line" docs/')).toBe(false);
    });

    it("接頭辞が一致するだけの別マーカーを検出しない", () => {
      expect(containsBoilerplateMarker("<!-- boilerplate:begin -->")).toBe(false);
    });
  });
});

describe("MARKER_LITERAL_FILES", () => {
  describe("正常系", () => {
    it("挙げた対象がすべて実在する", () => {
      for (const target of MARKER_LITERAL_FILES) {
        expect(fs.existsSync(path.join(ROOT_DIR, target)), target).toBe(true);
      }
    });

    // 除去を止める宣言なので、対象がもうマーカーを持たないなら宣言のほうが古い。
    // 放置すると「なぜ除去されないのか」が誰にも分からないファイルが増える。
    it("挙げた対象が実際にマーカー文字列を持つ", () => {
      for (const target of MARKER_LITERAL_FILES) {
        expect(containsBoilerplateMarker(read(target)), target).toBe(true);
      }
    });
  });
});

describe("BOILERPLATE_DELETE_FILES", () => {
  describe("正常系", () => {
    it("挙げた対象がすべて実在する", () => {
      for (const target of BOILERPLATE_DELETE_FILES) {
        expect(fs.existsSync(path.join(ROOT_DIR, target)), target).toBe(true);
      }
    });

    it("正本と日本語ミラーを対にして挙げている", () => {
      expect(BOILERPLATE_DELETE_FILES).toContain("docs/get-started/boilerplate-only-conventions.md");
      expect(BOILERPLATE_DELETE_FILES).toContain(
        "docs/ja/get-started/boilerplate-only-conventions.ja.md",
      );
    });
  });
});

describe("BOILERPLATE_MARKER", () => {
  describe("正常系", () => {
    it("他の除去マーカーと衝突しない名前である", () => {
      expect(BOILERPLATE_MARKER).not.toBe("sample-api");
      expect(BOILERPLATE_MARKER).not.toBe("setup-localize");
    });

    // 走査は列挙を持たないため、除去が空振りしていても何も出力せず成功する。
    // 「マーカーがリポジトリに 1 つは在る」を固定できるのはここだけ。
    it("マーカーがリポジトリに実在する", () => {
      const found = scanTargets().some((file) => containsBoilerplateMarker(fs.readFileSync(file, "utf8")));

      expect(found).toBe(true);
    });

    // 素の `boilerplate` へ寄せ戻すと、除去されないマーカーが静かに増える。
    it("素の boilerplate へ寄せ戻していない", () => {
      const pattern = /(?:\/\/|#|<!--)\s*boilerplate:(?:begin|end|line|replace-)/;
      const offenders = scanTargets()
        .filter((file) => pattern.test(fs.readFileSync(file, "utf8")))
        .map((file) => toRelativePath(file));

      expect(offenders).toEqual([]);
    });
  });
});

describe("BOILERPLATE_PROSE_MARKERS", () => {
  describe("正常系", () => {
    it("英日どちらの言い回しも挙げている", () => {
      expect(BOILERPLATE_PROSE_MARKERS).toContain("boilerplate");
      expect(BOILERPLATE_PROSE_MARKERS).toContain("ボイラープレート");
    });

    // 語が README に一度も現れないなら、消すものが無いか、語が変わったかのどちらか。
    it("挙げた語が実際に README へ現れる", () => {
      const readme = read("README.md") + read("README.ja.md");

      expect(BOILERPLATE_PROSE_MARKERS.some((word) => readme.includes(word))).toBe(true);
    });
  });
});
