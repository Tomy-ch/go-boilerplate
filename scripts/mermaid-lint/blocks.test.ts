import { describe, expect, it } from "vitest";

import { extractMermaidBlocks, isExcludedPath, isTargetMarkdown, shouldDescend } from "./blocks";

function md(...lines: string[]): string {
  return lines.join("\n");
}

describe("extractMermaidBlocks", () => {
  describe("正常系", () => {
    it("フェンスの中身を開始行付きで取り出す", () => {
      const content = md("# 見出し", "", "```mermaid", "graph TD;", "A-->B;", "```", "本文");

      expect(extractMermaidBlocks(content)).toEqual([
        { startLine: 3, code: "graph TD;\nA-->B;" },
      ]);
    });

    it("同じファイルの複数フェンスを順に取り出す", () => {
      const content = md("```mermaid", "a", "```", "", "```mermaid", "b", "```");

      expect(extractMermaidBlocks(content).map((block) => block.startLine)).toEqual([1, 5]);
    });

    it("チルダのフェンスも対象にする", () => {
      const content = md("~~~mermaid", "graph TD;", "~~~");

      expect(extractMermaidBlocks(content)).toEqual([{ startLine: 1, code: "graph TD;" }]);
    });

    it("字下げされたフェンスも対象にする", () => {
      const content = md("- 手順", "  ```mermaid", "  graph TD;", "  ```");

      expect(extractMermaidBlocks(content)).toEqual([{ startLine: 2, code: "  graph TD;" }]);
    });

    it("4 連以上のフェンスは同じ文字数以上でしか閉じない", () => {
      const content = md("````mermaid", "graph TD;", "```", "A-->B;", "````");

      expect(extractMermaidBlocks(content)).toEqual([
        { startLine: 1, code: "graph TD;\n```\nA-->B;" },
      ]);
    });

    it("mermaid 以外の言語のフェンスは拾わない", () => {
      expect(extractMermaidBlocks(md("```ts", "const a = 1;", "```"))).toEqual([]);
    });

    it("フェンスが無ければ空を返す", () => {
      expect(extractMermaidBlocks("本文だけ")).toEqual([]);
    });

    it("空のフェンスも 1 ブロックとして数える", () => {
      expect(extractMermaidBlocks(md("```mermaid", "```"))).toEqual([{ startLine: 1, code: "" }]);
    });
  });

  describe("異常系", () => {
    it("閉じられていないフェンスは末尾までを中身にする", () => {
      const content = md("```mermaid", "graph TD;", "A-->B;");

      expect(extractMermaidBlocks(content)).toEqual([{ startLine: 1, code: "graph TD;\nA-->B;" }]);
    });

    it("情報文字列を持つ行は閉じ扱いにしない", () => {
      const content = md("```mermaid", "graph TD;", "```mermaid", "A-->B;", "```");

      expect(extractMermaidBlocks(content)).toEqual([
        { startLine: 1, code: "graph TD;\n```mermaid\nA-->B;" },
      ]);
    });
  });
});

describe("isTargetMarkdown", () => {
  describe("正常系", () => {
    it("通常の Markdown を対象にする", () => {
      expect(isTargetMarkdown("docs/rules.md")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("生成物ツリーの Markdown を外す", () => {
      expect(isTargetMarkdown("docs/portal/guides/rules.md")).toBe(false);
    });

    it("AGENTS.md を外す", () => {
      expect(isTargetMarkdown("AGENTS.md")).toBe(false);
    });

    it("Markdown 以外を外す", () => {
      expect(isTargetMarkdown("docs/rules.txt")).toBe(false);
    });
  });
});

describe("isExcludedPath", () => {
  describe("異常系", () => {
    it("除外ツリーそのものを外す", () => {
      expect(isExcludedPath("docs/portal/guides")).toBe(true);
    });

    it("除外ツリー配下を外す", () => {
      expect(isExcludedPath("docs/portal/guides/rules.md")).toBe(true);
    });

    it("接頭辞が一致するだけの別ディレクトリを外さない", () => {
      expect(isExcludedPath("docs/coverage-notes/a.md")).toBe(false);
    });
  });
});

describe("shouldDescend", () => {
  describe("正常系", () => {
    it("除外に当たらないディレクトリへは降りる", () => {
      expect(shouldDescend("controller", "internal/controller")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("除外ディレクトリ名にはどこにあっても降りない", () => {
      expect(shouldDescend("node_modules", "scripts/node_modules")).toBe(false);
      expect(shouldDescend(".git", ".git")).toBe(false);
    });

    it("除外パスに当たる場所へは降りない", () => {
      expect(shouldDescend("guides", "docs/portal/guides")).toBe(false);
    });
  });
});
