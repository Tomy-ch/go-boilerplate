import fs from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { toAbsolutePath } from "../lib/file-utils";
import {
  COMMIT_SUBJECTS,
  DECLARED_LINES,
  DOC_REPLACEMENTS,
  MODES,
  REMOVED_PATHS,
  SELF_DESTRUCT_PATHS,
} from "./language-manifest";

/** 宣言が現物と噛み合っているかは、現物を読むことでしか答えられない。 */
function readRepoFile(relativePath: string): string | null {
  try {
    return fs.readFileSync(toAbsolutePath(relativePath), "utf8");
  } catch {
    return null;
  }
}

describe("MODES", () => {
  describe("正常系", () => {
    it("残す言語と「両方」を受け付ける", () => {
      expect([...MODES]).toEqual(["en", "ja", "both"]);
    });
  });
});

describe("COMMIT_SUBJECTS", () => {
  describe("正常系", () => {
    // commitlint の type-enum を外すと、撤去は済んでいるのにコミットできない状態になる。
    it("両モードの件名が commitlint の prefix を持つ", () => {
      for (const subject of Object.values(COMMIT_SUBJECTS)) {
        expect(subject).toMatch(/^Docs: /);
      }
    });
  });
});

describe("REMOVED_PATHS", () => {
  describe("正常系", () => {
    it("挙げた対象がすべて実在する", () => {
      for (const relativePath of REMOVED_PATHS) {
        expect(fs.existsSync(toAbsolutePath(relativePath)), relativePath).toBe(true);
      }
    });
  });
});

describe("SELF_DESTRUCT_PATHS", () => {
  describe("正常系", () => {
    it("撤去ツール自身を挙げている", () => {
      expect(SELF_DESTRUCT_PATHS).toContain("scripts/setup/remove-doc-language");
    });
  });
});

describe("DOC_REPLACEMENTS", () => {
  describe("正常系", () => {
    // 空振りのまま進めると、差し替えたつもりの記述がそのまま作成先へ残る。
    it("差し替え元が現物に完全一致で存在する", () => {
      for (const { file, from } of DOC_REPLACEMENTS) {
        expect(readRepoFile(file) ?? "", `${file}: ${from}`).toContain(from);
      }
    });

    it("差し替え先は対訳へ触れない", () => {
      for (const { file, to } of DOC_REPLACEMENTS) {
        expect(to, file).not.toContain(".ja.md");
      }
    });
  });
});

describe("DECLARED_LINES", () => {
  describe("正常系", () => {
    it("宣言した行が現物のどこかに完全一致で存在する", () => {
      const sources = [...REMOVED_PATHS]
        .flatMap((root) => listMarkdown(toAbsolutePath(root)))
        .map((file) => fs.readFileSync(file, "utf8"));

      for (const declared of DECLARED_LINES) {
        expect(
          sources.some((source) =>
            source.split("\n").some((line) => line.trim() === declared),
          ),
          declared,
        ).toBe(true);
      }
    });
  });
});

function listMarkdown(dir: string, files: string[] = []): string[] {
  if (!fs.existsSync(dir)) {
    return files;
  }

  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);

    if (entry.isDirectory()) {
      listMarkdown(full, files);
      continue;
    }

    if (entry.isFile() && entry.name.endsWith(".md")) {
      files.push(full);
    }
  }

  return files;
}
