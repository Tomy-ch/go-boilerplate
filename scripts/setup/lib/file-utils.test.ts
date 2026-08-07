import { mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { basename, join } from "node:path";
import { describe, expect, it } from "vitest";

import {
  listChildFiles,
  listFilesRecursive,
  toAbsolutePath,
  toRelativePath,
  updateAbsoluteFile,
  updateFile,
} from "./file-utils";

/** 実ファイルを置いた一時ディレクトリ。対象が fs の振る舞いそのものなので偽装しない。 */
function tempDir(): string {
  return mkdtempSync(join(tmpdir(), "setup-file-utils-"));
}

function writeAt(dir: string, relativePath: string, content: string): string {
  const path = join(dir, relativePath);
  mkdirSync(join(path, ".."), { recursive: true });
  writeFileSync(path, content, "utf8");
  return path;
}

describe("toAbsolutePath", () => {
  describe("正常系", () => {
    it("リポジトリルートからの相対パスを絶対パスにする", () => {
      expect(toAbsolutePath("scripts/setup")).toMatch(/\/scripts\/setup$/);
      expect(toAbsolutePath("a.md").startsWith("/")).toBe(true);
    });
  });
});

describe("toRelativePath", () => {
  describe("正常系", () => {
    it("絶対パスをリポジトリルートからの相対へ戻す", () => {
      expect(toRelativePath(toAbsolutePath("scripts/setup/lib/file-utils.ts"))).toBe(
        "scripts/setup/lib/file-utils.ts",
      );
    });
  });
});

describe("updateAbsoluteFile", () => {
  describe("正常系", () => {
    it("変換後の内容を書き戻し相対パスを返す", () => {
      const dir = tempDir();
      const path = writeAt(dir, "a.txt", "old");

      const result = updateAbsoluteFile(path, () => "new", false);

      expect(result).not.toBeNull();
      expect(readFileSync(path, "utf8")).toBe("new");
    });

    it("ドライランでは書き込まないが実行時と同じ戻り値を返す", () => {
      const dir = tempDir();
      const path = writeAt(dir, "a.txt", "old");

      const dry = updateAbsoluteFile(path, () => "new", true);
      expect(readFileSync(path, "utf8")).toBe("old");

      const real = updateAbsoluteFile(path, () => "new", false);
      expect(dry).toBe(real);
    });

    it("変換が変更なしを返せば書き込まず null を返す", () => {
      const dir = tempDir();
      const path = writeAt(dir, "a.txt", "same");

      expect(updateAbsoluteFile(path, () => null, false)).toBeNull();
      expect(readFileSync(path, "utf8")).toBe("same");
    });

    it("変換後が元と同一なら書き込まず null を返す", () => {
      const dir = tempDir();
      const path = writeAt(dir, "a.txt", "same");

      expect(updateAbsoluteFile(path, (original) => original, false)).toBeNull();
    });
  });

  describe("異常系", () => {
    it("対象が存在しなければ変換を呼ばず null を返す", () => {
      let called = false;

      const result = updateAbsoluteFile(join(tempDir(), "absent.txt"), () => {
        called = true;
        return "x";
      }, false);

      expect(result).toBeNull();
      expect(called).toBe(false);
    });

    it("変換が投げた例外は握り潰さず伝播する", () => {
      const dir = tempDir();
      const path = writeAt(dir, "a.txt", "old");

      expect(() =>
        updateAbsoluteFile(path, () => {
          throw new Error("想定外の本文");
        }, false),
      ).toThrow("想定外の本文");
      expect(readFileSync(path, "utf8")).toBe("old");
    });
  });
});

describe("updateFile", () => {
  describe("正常系", () => {
    it("リポジトリ相対パスを解決して同じ判断を通す", () => {
      expect(updateFile("scripts/setup/lib/file-utils.ts", () => null, true)).toBeNull();
    });
  });

  describe("異常系", () => {
    it("存在しない相対パスは null を返す", () => {
      expect(updateFile("scripts/setup/absent-file.txt", () => "x", true)).toBeNull();
    });
  });
});

describe("listFilesRecursive", () => {
  describe("正常系", () => {
    it("名前順で再帰的に列挙する", () => {
      const dir = tempDir();
      writeAt(dir, "b.txt", "");
      writeAt(dir, "a.txt", "");
      writeAt(dir, "sub/c.txt", "");

      expect(listFilesRecursive(dir).map((p) => basename(p))).toEqual(["a.txt", "b.txt", "c.txt"]);
    });

    it("除外ディレクトリへは降りない", () => {
      const dir = tempDir();
      writeAt(dir, "a.txt", "");
      writeAt(dir, "node_modules/b.txt", "");

      const files = listFilesRecursive(dir, { excludedDirectories: new Set(["node_modules"]) });

      expect(files.map((p) => basename(p))).toEqual(["a.txt"]);
    });

    it("述語に合うファイルだけを残す", () => {
      const dir = tempDir();
      writeAt(dir, "a.go", "");
      writeAt(dir, "b.md", "");

      const files = listFilesRecursive(dir, { shouldIncludeFile: (p) => p.endsWith(".go") });

      expect(files.map((p) => basename(p))).toEqual(["a.go"]);
    });

    it("述語はフルパスを受け取る", () => {
      const dir = tempDir();
      writeAt(dir, "sub/a.txt", "");
      const seen: string[] = [];

      listFilesRecursive(dir, {
        shouldIncludeFile: (p) => {
          seen.push(p);
          return true;
        },
      });

      expect(seen[0].startsWith(dir)).toBe(true);
      expect(seen[0]).toContain("sub");
    });
  });

  describe("異常系", () => {
    it("ファイルが 1 つも無ければ空を返す", () => {
      expect(listFilesRecursive(tempDir())).toEqual([]);
    });
  });
});

describe("listChildFiles", () => {
  describe("正常系", () => {
    it("直下のファイルだけを名前順で返す", () => {
      const files = listChildFiles("scripts/setup/lib");
      const sorted = [...files].sort((a, b) => a.localeCompare(b));

      expect(files).toEqual(sorted);
      expect(files.every((p) => p.includes("/scripts/setup/lib/"))).toBe(true);
    });

    it("述語はファイル名を受け取り、合うものだけを残す", () => {
      const seen: string[] = [];

      const files = listChildFiles("scripts/setup/lib", (name) => {
        seen.push(name);
        return name.endsWith(".test.ts");
      });

      expect(seen.every((name) => !name.includes("/"))).toBe(true);
      expect(files.every((p) => p.endsWith(".test.ts"))).toBe(true);
      expect(files.length).toBeGreaterThan(0);
    });
  });

  describe("異常系", () => {
    it("どれにも合わない述語なら空を返す", () => {
      expect(listChildFiles("scripts/setup/lib", () => false)).toEqual([]);
    });
  });
});
