import { describe, expect, it } from "vitest";

import {
  canonicalOf,
  describesAPair,
  isDecorationOnly,
  isNavigationOnly,
  isScanTarget,
  isTranslationNotice,
  listDocPairs,
  namesAnyTranslation,
  redactReferences,
  resolveTarget,
  rewriteTranslationLinks,
  stripLeadingTranslationNote,
  transplantFrontmatter,
} from "./doc-language";

/** リポジトリの実物と同じ形の導入行。 */
const HEADER_LINE = "English | [日本語](README.ja.md)";

/** 撤去対象を 1 つだけ持つ述語。 */
function removes(...targets: readonly string[]): (target: string) => boolean {
  const removed = new Set(targets);

  return (target) => removed.has(target);
}

describe("isScanTarget", () => {
  describe("正常系", () => {
    it("走査対象の Markdown を受け入れる", () => {
      expect(isScanTarget("internal/domain/README.ja.md")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("生成物のポータル複製を外す", () => {
      expect(isScanTarget("docs/portal/guides/ja/overview.ja.md")).toBe(false);
    });

    it("依存物を外す", () => {
      expect(isScanTarget("vendor/example/README.md")).toBe(false);
    });

    it("Markdown 以外を外す", () => {
      expect(isScanTarget("internal/domain/user.go")).toBe(false);
    });
  });
});

describe("canonicalOf", () => {
  describe("正常系", () => {
    it("対訳の隣の正本パスを返す", () => {
      expect(canonicalOf("docs/rules.ja.md")).toBe("docs/rules.md");
    });
  });

  describe("異常系", () => {
    it("対訳でなければそのまま返す", () => {
      expect(canonicalOf("docs/rules.md")).toBe("docs/rules.md");
    });
  });
});

describe("listDocPairs", () => {
  describe("正常系", () => {
    it("対訳を起点にペアを組み、並びを固定する", () => {
      expect(listDocPairs(["b/README.ja.md", "a/README.ja.md", "a/README.md"])).toEqual([
        { canonical: "a/README.md", translation: "a/README.ja.md" },
        { canonical: "b/README.md", translation: "b/README.ja.md" },
      ]);
    });

    it("正本が実在しない孤児の対訳もペアとして返す", () => {
      expect(listDocPairs(["a/README.ja.md"])).toEqual([
        { canonical: "a/README.md", translation: "a/README.ja.md" },
      ]);
    });
  });

  describe("異常系", () => {
    it("対訳が 1 件も無ければ空を返す", () => {
      expect(listDocPairs(["a/README.md", "a/main.go"])).toEqual([]);
    });

    it("走査対象外の対訳を拾わない", () => {
      expect(listDocPairs(["docs/portal/guides/ja/overview.ja.md"])).toEqual([]);
    });
  });
});

describe("resolveTarget", () => {
  describe("正常系", () => {
    it("参照元ディレクトリを基点に解決する", () => {
      expect(resolveTarget("README.ja.md", "internal/domain")).toBe(
        "internal/domain/README.ja.md",
      );
    });

    it("親を辿る参照を解決する", () => {
      expect(resolveTarget("../README.md", "internal/domain")).toBe("internal/README.md");
    });

    it("アンカーを落として解決する", () => {
      expect(resolveTarget("rules.md#layer", "docs")).toBe("docs/rules.md");
    });

    // basename で突き合わせると、このリポジトリで最も多い名前が他ディレクトリの同名ファイルまで拾う。
    it("同名でもディレクトリが違えば別のパスになる", () => {
      expect(resolveTarget("SKILL.md", ".claude/skills/a")).not.toBe(
        resolveTarget("SKILL.md", ".claude/skills/b"),
      );
    });
  });

  describe("異常系", () => {
    it("外部 URL は解決しない", () => {
      expect(resolveTarget("https://example.com/README.md", "docs")).toBeNull();
    });

    it("空の参照は解決しない", () => {
      expect(resolveTarget("#anchor", "docs")).toBeNull();
    });
  });
});

describe("isDecorationOnly", () => {
  describe("正常系", () => {
    it("言語ラベルと区切り記号だけの残りを飾りと見なす", () => {
      expect(isDecorationOnly(" English | ")).toBe(true);
    });

    it("長いラベルを先に外して取り残しを作らない", () => {
      expect(isDecorationOnly("English canonical: ")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("語が残っていれば飾りではない", () => {
      expect(isDecorationOnly("See  for the version.")).toBe(false);
    });
  });
});

describe("isNavigationOnly", () => {
  describe("正常系", () => {
    it("リンクと飾りだけで出来た行を行き先の一覧と見なす", () => {
      expect(isNavigationOnly("[Controller README](../README.md) | 日本語: ")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("散文を含む行は一覧ではない", () => {
      expect(isNavigationOnly("See [the guide](../README.md) before editing.")).toBe(false);
    });
  });
});

describe("namesAnyTranslation", () => {
  describe("正常系", () => {
    it("対訳を名指す綴りに当たる", () => {
      expect(namesAnyTranslation("Update `README.ja.md` too.")).toBe(true);
    });

    it("グロブにも当たる", () => {
      expect(namesAnyTranslation("skip `*.ja.md` files")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("正本の名前だけなら当たらない", () => {
      expect(namesAnyTranslation("Update `README.md` too.")).toBe(false);
    });
  });
});

describe("describesAPair", () => {
  describe("正常系", () => {
    // 畳めば同じ 1 ファイルを 2 回挙げることになる行。
    it("正本と対訳を並べた行に当たる", () => {
      expect(describesAPair("- `env/README.md`, `env/README.ja.md` — 表の形式")).toBe(true);
    });

    it("当たるファイルが無くなるグロブに当たる", () => {
      expect(describesAPair("skip `*.ja.md` files")).toBe(true);
    });
  });

  describe("異常系", () => {
    // ja では改名で名前が付け替わるだけなので、単独の言及は生き残る。
    it("他の文書への単独の言及には当たらない", () => {
      expect(describesAPair("- `architecture.ja.md`")).toBe(false);
    });

    it("対訳に触れない行には当たらない", () => {
      expect(describesAPair("- `architecture.md`")).toBe(false);
    });
  });
});

describe("isTranslationNotice", () => {
  describe("正常系", () => {
    it("日本語の対訳注記に当たる", () => {
      expect(isTranslationNotice("> このファイルは `SKILL.md` の日本語参考訳です。")).toBe(true);
    });

    it("英語の対訳注記に当たる", () => {
      expect(
        isTranslationNotice("A Japanese reference translation is available at `SKILL.ja.md`."),
      ).toBe(true);
    });
  });

  describe("異常系", () => {
    it("対訳に触れない散文には当たらない", () => {
      expect(isTranslationNotice("This document defines the architecture rules.")).toBe(false);
    });
  });
});

describe("rewriteTranslationLinks", () => {
  describe("正常系", () => {
    it("対訳への参照を正本の名前へ寄せる", () => {
      expect(rewriteTranslationLinks("[x](../a/README.ja.md)")).toBe("[x](../a/README.md)");
    });
  });

  describe("異常系", () => {
    it("対訳を含まない本文を変えない", () => {
      expect(rewriteTranslationLinks("[x](../a/README.md)")).toBe("[x](../a/README.md)");
    });
  });
});

describe("stripLeadingTranslationNote", () => {
  describe("正常系", () => {
    // 改名して正本にした後もこれが残ると、正本が自分を訳だと名乗る。
    it("冒頭の引用注記を落とす", () => {
      expect(stripLeadingTranslationNote("> `SKILL.md` の訳です。\n\n# 見出し\n", "SKILL.md")).toBe(
        "# 見出し\n",
      );
    });

    it("複数行の引用注記をまとめて落とす", () => {
      expect(
        stripLeadingTranslationNote("> 1 行目 `X.md`\n> 2 行目\n\n# 見出し\n", "X.md"),
      ).toBe("# 見出し\n");
    });

    it("フロントマターの後ろにある注記も落とす", () => {
      expect(
        stripLeadingTranslationNote("---\nname: a\n---\n\n> `X.md` の訳\n\n# 見出し\n", "X.md"),
      ).toBe("---\nname: a\n---\n\n# 見出し\n");
    });
  });

  describe("異常系", () => {
    it("正本を指さない引用は残す", () => {
      const source = "> 補足です。\n\n# 見出し\n";

      expect(stripLeadingTranslationNote(source, "X.md")).toBe(source);
    });

    // 位置が規約で決まっているのは冒頭だけ。本文中の引用まで対象にすると巻き込む。
    it("本文中の引用は落とさない", () => {
      const source = "# 見出し\n\n> `X.md` を参照。\n";

      expect(stripLeadingTranslationNote(source, "X.md")).toBe(source);
    });

    it("閉じていないフロントマターには手を触れない", () => {
      const source = "---\nname: a\n\n> `X.md` の訳\n";

      expect(stripLeadingTranslationNote(source, "X.md")).toBe(source);
    });
  });
});

describe("transplantFrontmatter", () => {
  describe("正常系", () => {
    // 移植を落とすと、日本語を選んだだけで 84 スキルが 1 本残らず読み込まれなくなる。
    it("訳文が持たないフロントマターを正本から移す", () => {
      expect(transplantFrontmatter("---\nname: commit\n---\n\n# Commit\n", "# コミット\n")).toBe(
        "---\nname: commit\n---\n\n# コミット\n",
      );
    });
  });

  describe("異常系", () => {
    it("訳文が既に持っていればそちらを正とする", () => {
      const translation = "---\nname: ja\n---\n\n# 見出し\n";

      expect(transplantFrontmatter("---\nname: en\n---\n\n# Heading\n", translation)).toBe(
        translation,
      );
    });

    it("正本が持たなければ何も移さない", () => {
      expect(transplantFrontmatter("# Heading\n", "# 見出し\n")).toBe("# 見出し\n");
    });
  });
});

describe("redactReferences", () => {
  describe("正常系", () => {
    it("行き先の一覧だけになった行を落とす", () => {
      const result = redactReferences(
        `# Title\n\n${HEADER_LINE}\n\nBody.\n`,
        "README.md",
        removes("README.ja.md"),
        new Set(),
      );

      expect(result.content).toBe("# Title\n\nBody.\n");
      expect(result.undeclared).toEqual([]);
    });

    it("消した跡で空行が重ならないようにする", () => {
      const result = redactReferences(
        `a\n\n${HEADER_LINE}\n\nb\n`,
        "README.md",
        removes("README.ja.md"),
        new Set(),
      );

      expect(result.content).not.toContain("\n\n\n");
    });

    it("対訳の存在を述べる注記を落とす", () => {
      const result = redactReferences(
        "> このファイルは [README.md](README.md) の日本語訳です。\n\n# 見出し\n",
        "README.md",
        removes("README.md"),
        new Set(),
      );

      expect(result.content).toBe("# 見出し\n");
    });

    it("宣言された行を落とす", () => {
      const result = redactReferences(
        "keep\ndrop me\n",
        "README.md",
        removes("README.ja.md"),
        new Set(["drop me"]),
      );

      expect(result.content).toBe("keep\n");
    });

    it("消えた側を指さない参照を残す", () => {
      const source = "[other](../other/README.md)\n";

      expect(redactReferences(source, "README.md", removes("README.ja.md"), new Set()).content).toBe(
        source,
      );
    });
  });

  describe("異常系", () => {
    // 黙って剥がすと、規約を説明していた文が意味の通らない残骸になって残る。
    it("判断が要る散文は書き換えず報告する", () => {
      const source = "See [the translation](README.ja.md) before editing.\n";
      const result = redactReferences(source, "README.md", removes("README.ja.md"), new Set());

      expect(result.content).toBe(source);
      expect(result.undeclared).toEqual([
        { file: "README.md", line: 1, text: "See [the translation](README.ja.md) before editing." },
      ]);
    });

    it("リンクになっていない言及も報告する", () => {
      const result = redactReferences(
        "Update `README.ja.md` whenever the table changes.\n",
        "README.md",
        removes("README.ja.md"),
        new Set(),
        namesAnyTranslation,
      );

      expect(result.undeclared).toHaveLength(1);
    });

    // ja では相手の名前が残るため、同じ検査を当てると有効な相互参照を大量に報告する。
    it("名前ごと消えないなら言及を報告しない", () => {
      const result = redactReferences(
        "Update `README.md` whenever the table changes.\n",
        "README.md",
        removes("README.md"),
        new Set(),
      );

      expect(result.undeclared).toEqual([]);
    });
  });
});
