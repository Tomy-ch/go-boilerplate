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

    // ペアを持たない文書も対訳規約を語る。ここを歩かないと、残った言及が報告もされず作成先へ渡る。
    it("ja では対訳を持たない文書のリンクも正本の名前へ寄せる", () => {
      const plan = planRemoval(
        "ja",
        ["agent.md", "b/README.md", "b/README.ja.md"],
        reader({
          "agent.md": "[b](b/README.ja.md)\n",
          "b/README.md": "# B\n",
          "b/README.ja.md": "# ビー\n",
        }),
      );

      expect(plan.operations).toContainEqual({
        kind: "write",
        path: "agent.md",
        content: "[b](b/README.md)\n",
      });
    });

    it("ja では対訳に触れない文書を書き換え対象に挙げない", () => {
      const plan = planRemoval("ja", ["plain.md"], reader({ "plain.md": "# 平文\n" }));

      expect(plan.operations).toEqual([]);
    });

    it("ja では対訳を持たない文書の宣言なき散文も報告する", () => {
      const plan = planRemoval("ja", ["agent.md"], reader({ "agent.md": "skip `*.ja.md` files\n" }));

      expect(plan.undeclared).toEqual([
        { file: "agent.md", line: 1, text: "skip `*.ja.md` files" },
      ]);
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

    // 対訳の存在を前提にしているのは散文だけではない。検査スクリプトや取り込み表も同じ前提に立つ。
    it("Markdown 以外のマーカーも剥がす", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md", "scripts/x/rules.ts"],
        reader({
          "a.md": "# A\n",
          "a.ja.md": "# あ\n",
          "scripts/x/rules.ts": "const keep = 1;\n// doc-pair:begin\nconst gone = 2;\n// doc-pair:end\n",
        }),
      );

      expect(plan.operations).toContainEqual({
        kind: "write",
        path: "scripts/x/rules.ts",
        content: "const keep = 1;\n",
      });
    });

    it("拡張子を持たない無視リストも対象にする", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md", ".graphifyignore"],
        reader({ "a.md": "# A\n", "a.ja.md": "# あ\n", ".graphifyignore": "*.ja.md\nkeep\n" }),
        new Set(),
        [],
        [{ file: ".graphifyignore", from: "*.ja.md\n", to: "" }],
      );

      expect(plan.operations).toContainEqual({
        kind: "write",
        path: ".graphifyignore",
        content: "keep\n",
      });
    });

    // 綴りを見張る検査が Markdown だけを見ていると、検査スクリプトに残った言及が
    // 報告もされないまま作成先へ渡る。
    it("Markdown 以外に残った対訳への言及も報告する", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md", "scripts/x/rules.ts"],
        reader({
          "a.md": "# A\n",
          "a.ja.md": "# あ\n",
          "scripts/x/rules.ts": 'const pair = "README.ja.md";\n',
        }),
      );

      expect(plan.undeclared).toEqual([
        { file: "scripts/x/rules.ts", line: 1, text: 'const pair = "README.ja.md";' },
      ]);
    });

    it("撤去ごと消えるパスは Markdown 以外でも走査しない", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md", "gone/rules.ts"],
        reader({ "a.md": "# A\n", "a.ja.md": "# あ\n", "gone/rules.ts": 'const p = "x.ja.md";\n' }),
        new Set(),
        ["gone"],
      );

      expect(plan.undeclared).toEqual([]);
    });

    it("モードが合わない宣言は効かせない", () => {
      const plan = planRemoval(
        "en",
        ["t.md", "t.ja.md"],
        reader({ "t.md": "英語のみ\n", "t.ja.md": "訳\n" }),
        new Set(),
        [],
        [{ file: "t.md", from: "英語のみ", to: "置換後", mode: "ja" }],
      );

      expect(plan.operations).not.toContainEqual(
        expect.objectContaining({ kind: "write", path: "t.md" }),
      );
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

    it("存在しないファイルを指す差し替え宣言を空振りとして報告する", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md"],
        reader({ "a.md": "# A\n", "a.ja.md": "# あ\n" }),
        new Set(),
        [],
        [{ file: "gone.ts", from: "何か", to: "" }],
      );

      expect(plan.staleReplacements).toEqual([{ file: "gone.ts", from: "何か", to: "" }]);
    });

    // 孤児の対訳は正本を持たないので、移植元が無いまま改名だけを行う。
    it("ja で正本の無い対訳も改名する", () => {
      const plan = planRemoval("ja", ["orphan.ja.md"], reader({ "orphan.ja.md": "# 訳\n" }));

      expect(plan.operations).toEqual([
        { kind: "rename", from: "orphan.ja.md", to: "orphan.md", content: "# 訳\n" },
      ]);
    });

    it("読めない Markdown を黙って飛ばす", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md", "gone.md"],
        reader({ "a.md": "# A\n", "a.ja.md": "# あ\n" }),
      );

      expect(plan.operations).toEqual([{ kind: "delete", path: "a.ja.md" }]);
    });

    it("ja でも読めない Markdown を黙って飛ばす", () => {
      const plan = planRemoval(
        "ja",
        ["a.md", "a.ja.md", "gone.md"],
        reader({ "a.md": "# A\n", "a.ja.md": "# あ\n" }),
      );

      expect(plan.operations).toEqual([
        { kind: "rename", from: "a.ja.md", to: "a.md", content: "# あ\n" },
      ]);
    });

    it("読めない非 Markdown を黙って飛ばす", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md", "gone.ts"],
        reader({ "a.md": "# A\n", "a.ja.md": "# あ\n" }),
      );

      expect(plan.undeclared).toEqual([]);
    });

    // 撤去の出力とコミットの中身が走査順（ファイルシステムの都合）で揺れないようにする。
    it("非 Markdown の対象も名前順に並べる", () => {
      const plan = planRemoval(
        "en",
        ["a.md", "a.ja.md", "z/b.ts", "a/b.ts"],
        reader({
          "a.md": "# A\n",
          "a.ja.md": "# あ\n",
          "z/b.ts": 'const p = "z.ja.md";\n',
          "a/b.ts": 'const p = "a.ja.md";\n',
        }),
      );

      expect(plan.undeclared.map(({ file }) => file)).toEqual(["a/b.ts", "z/b.ts"]);
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
