import { describe, expect, it } from "vitest";

import { type ReadFile, planRemoval } from "./plan";

/** 本文の読み出しを固定表で差し替える。 */
function reader(files: Readonly<Record<string, string>>): ReadFile {
  return (relativePath) => files[relativePath] ?? null;
}

const CANONICAL = "# Title\n\nEnglish | [日本語](README.ja.md)\n\nBody.\n";
const TRANSLATION = "# 見出し\n\n[English](README.md) | 日本語\n\n本文。\n";

describe("planRemoval", () => {
  describe("正常系", () => {
    it("en では対訳を消し、正本から参照を落とす", () => {
      const plan = planRemoval(
        "en",
        ["README.md", "README.ja.md"],
        reader({ "README.md": CANONICAL, "README.ja.md": TRANSLATION }),
      );

      expect(plan.operations).toEqual([
        { kind: "delete", path: "README.ja.md" },
        { kind: "write", path: "README.md", content: "# Title\n\nBody.\n" },
      ]);
      expect(plan.undeclared).toEqual([]);
    });

    it("ja では対訳を正本の名前へ改名し、自己参照を落とす", () => {
      const plan = planRemoval(
        "ja",
        ["README.md", "README.ja.md"],
        reader({ "README.md": CANONICAL, "README.ja.md": TRANSLATION }),
      );

      expect(plan.operations).toEqual([
        { kind: "rename", from: "README.ja.md", to: "README.md", content: "# 見出し\n\n本文。\n" },
      ]);
    });

    // 移植を落とすと、日本語を選んだだけでスキルが 1 本残らず読み込まれなくなる。
    it("ja では正本のフロントマターを訳文へ移す", () => {
      const plan = planRemoval(
        "ja",
        ["SKILL.md", "SKILL.ja.md"],
        reader({ "SKILL.md": "---\nname: commit\n---\n\n# Commit\n", "SKILL.ja.md": "# コミット\n" }),
      );

      expect(plan.operations[0]).toMatchObject({
        kind: "rename",
        content: "---\nname: commit\n---\n\n# コミット\n",
      });
    });

    it("ja では対訳どうしのリンクを正本の名前へ寄せる", () => {
      const plan = planRemoval(
        "ja",
        ["a/README.md", "a/README.ja.md"],
        reader({ "a/README.md": "# A\n", "a/README.ja.md": "[b](../b/README.ja.md)\n" }),
      );

      expect(plan.operations[0]).toMatchObject({ content: "[b](../b/README.md)\n" });
    });

    it("撤去ごと消えるパスを畳む対象から外す", () => {
      const plan = planRemoval(
        "en",
        ["gone/SKILL.md", "gone/SKILL.ja.md"],
        reader({ "gone/SKILL.md": "See [x](SKILL.ja.md) always.\n", "gone/SKILL.ja.md": "訳\n" }),
        new Set(),
        ["gone"],
      );

      expect(plan.operations).toEqual([]);
      expect(plan.undeclared).toEqual([]);
    });

    it("表のセルのような場所を宣言どおり差し替える", () => {
      const plan = planRemoval(
        "en",
        ["t.md", "t.ja.md"],
        reader({ "t.md": "| a | `SKILL.ja.md` を除く |\n", "t.ja.md": "訳\n" }),
        new Set(),
        [],
        [{ file: "t.md", from: " `SKILL.ja.md` を除く", to: "" }],
      );

      expect(plan.operations).toContainEqual({
        kind: "write",
        path: "t.md",
        content: "| a | |\n",
      });
      expect(plan.staleReplacements).toEqual([]);
    });
  });

  describe("異常系", () => {
    // 半分だけ畳まれたツリーから復旧する手順は誰も持っていない。
    it("判断が要る散文があっても操作を組み立てず報告だけ返す", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md"],
        reader({ "a.md": "See [it](a.ja.md) before editing.\n", "a.ja.md": "訳\n" }),
      );

      expect(plan.undeclared).toHaveLength(1);
      expect(plan.undeclared[0]).toMatchObject({ file: "a.md", line: 1 });
    });

    it("本文に当たらない差し替え宣言を報告する", () => {
      const plan = planRemoval(
        "en",
        ["t.md", "t.ja.md"],
        reader({ "t.md": "本文\n", "t.ja.md": "訳\n" }),
        new Set(),
        [],
        [{ file: "t.md", from: "動いてしまった文言", to: "" }],
      );

      expect(plan.staleReplacements).toEqual([{ file: "t.md", from: "動いてしまった文言", to: "" }]);
    });

    it("読めないファイルを黙って飛ばす", () => {
      const plan = planRemoval("ja", ["gone.ja.md"], reader({}));

      expect(plan.operations).toEqual([]);
    });

    it("対訳が 1 件も無ければ何もしない", () => {
      const plan = planRemoval("en", ["README.md"], reader({ "README.md": "# Title\n" }));

      expect(plan.operations).toEqual([]);
    });
  });
});
