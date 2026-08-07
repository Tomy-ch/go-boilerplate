import { describe, expect, it } from "vitest";

import { autoTitle, buildDocsJson, guideIdOf, slugify, type DiscoveredDocs, isSectionDirectory, isMarkdownFile, sortSectionNames } from "./docs-json";

const EMPTY: DiscoveredDocs = { directories: [], rootEnFiles: [], rootJaFiles: [] };

function build(manifest: Record<string, unknown>, discovered: DiscoveredDocs = EMPTY) {
  return buildDocsJson(manifest, discovered);
}

describe("autoTitle", () => {
  describe("正常系", () => {
    it("拡張子を落として語頭を大文字にする", () => {
      expect(autoTitle("rest-design.md")).toBe("Rest Design");
    });
    it("対訳の拡張子も落とす", () => {
      expect(autoTitle("rest-design.ja.md")).toBe("Rest Design");
    });
    it("アンダースコアも語の区切りにする", () => {
      expect(autoTitle("make_commands")).toBe("Make Commands");
    });
  });
});

describe("slugify", () => {
  describe("正常系", () => {
    it("英数字以外を繋ぎに畳む", () => {
      expect(slugify("Get Started")).toBe("get-started");
    });
    it("前後の繋ぎを落とす", () => {
      expect(slugify("  API Contracts  ")).toBe("api-contracts");
    });
  });
});

describe("guideIdOf", () => {
  describe("正常系", () => {
    it("EN と JA の対を同じ識別子へ寄せる", () => {
      expect(guideIdOf("docs/portal/guides/rules.md")).toBe("rules");
      expect(guideIdOf("docs/portal/guides/rules.ja.md")).toBe("rules");
    });
    it("html の 1 段だけを落とす（foo.html.md と foo.html を混ぜない）", () => {
      expect(guideIdOf("foo.html.md")).toBe("foo.html");
      expect(guideIdOf("foo.html")).toBe("foo");
    });
    it("拡張子を持たない名前はそのまま使う", () => {
      expect(guideIdOf("docs/coverage")).toBe("coverage");
    });
  });
});

describe("buildDocsJson", () => {
  describe("正常系", () => {
    it("manifest の section をカードに変換する", () => {
      const { docs } = build({
        meta: { groups: [{ title: "Architecture", sections: ["adr"] }] },
        adr: [{ src: "docs/adr/0001-x.md", dst: "docs/portal/guides/adr-0001-x.md" }],
      });

      expect(docs.groups).toEqual([
        {
          title: "Architecture",
          slug: "architecture",
          sections: [
            {
              id: "adr",
              slug: "adr",
              title: "Adr",
              items: [
                {
                  name: "Adr 0001 X",
                  path: "./guides/adr-0001-x.md",
                  lang: "en",
                  source: "docs/adr/0001-x.md",
                  guideId: "adr-0001-x",
                },
              ],
            },
          ],
        },
      ]);
    });

    it("meta.section_titles で section 見出しを上書きする", () => {
      const { docs } = build({
        meta: {
          section_titles: { adr: "Decisions" },
          groups: [{ title: "G", sections: ["adr"] }],
        },
        adr: [{ src: "a.md", dst: "docs/portal/guides/a.md" }],
      });

      expect(docs.groups[0].sections[0].title).toBe("Decisions");
    });

    it("対訳を lang: ja として持ち、EN の後ろへ並べる", () => {
      const { docs } = build({
        meta: { groups: [{ title: "G", sections: ["adr"] }] },
        adr: [
          { src: "docs/ja/adr/b.md", dst: "docs/portal/guides/b.ja.md" },
          { src: "docs/adr/a.md", dst: "docs/portal/guides/a.md" },
        ],
      });

      expect(docs.groups[0].sections[0].items.map((item) => item.lang)).toEqual(["en", "ja"]);
    });

    it("走査で見つけた docs/<dir> を section にする", () => {
      const { docs } = build(
        { meta: { groups: [{ title: "G", sections: ["adr"] }] } },
        {
          directories: [
            { name: "adr", hasIndexHtml: false, enFiles: ["a.md"], jaFiles: ["a.md"] },
          ],
          rootEnFiles: [],
          rootJaFiles: [],
        },
      );

      expect(docs.groups[0].sections[0].items).toEqual([
        { name: "A", path: "../adr/a.md", lang: "en", source: "docs/adr/a.md", guideId: "a" },
        { name: "A", path: "../ja/adr/a.md", lang: "ja", source: "docs/ja/adr/a.md", guideId: "a" },
      ]);
    });

    it("index.html を持つ section は言語を問わない 1 枚のカードにする", () => {
      const { docs } = build(
        { meta: { groups: [{ title: "G", sections: ["godoc"] }] } },
        {
          directories: [{ name: "godoc", hasIndexHtml: true, enFiles: [], jaFiles: [] }],
          rootEnFiles: [],
          rootJaFiles: [],
        },
      );

      expect(docs.groups[0].sections[0].items).toEqual([
        {
          name: "Godoc",
          path: "../godoc/index.html",
          lang: "all",
          source: "docs/godoc/index.html",
          guideId: "godoc",
        },
      ]);
    });

    it("index.html のカードを EN / JA の後ろへ置く", () => {
      const { docs } = build(
        { meta: { groups: [{ title: "G", sections: ["godoc"] }] } },
        {
          directories: [{ name: "godoc", hasIndexHtml: true, enFiles: ["a.md"], jaFiles: ["b.md"] }],
          rootEnFiles: [],
          rootJaFiles: [],
        },
      );

      expect(docs.groups[0].sections[0].items.map((item) => item.lang)).toEqual(["en", "ja", "all"]);
    });

    it("docs 直下の Markdown を architecture セクションへ集約する", () => {
      const { docs } = build(
        { meta: { groups: [{ title: "G", sections: ["architecture"] }] } },
        { directories: [], rootEnFiles: ["rules.md"], rootJaFiles: [] },
      );

      expect(docs.groups[0].sections[0]).toMatchObject({
        id: "architecture",
        title: "Architecture Docs",
      });
    });

    it("docs/ja 直下の Markdown も同じ architecture セクションへ集約する", () => {
      const { docs } = build(
        { meta: { groups: [{ title: "G", sections: ["architecture"] }] } },
        { directories: [], rootEnFiles: [], rootJaFiles: ["rules.ja.md"] },
      );

      expect(docs.groups[0].sections[0].items).toEqual([
        {
          name: "Rules",
          path: "../ja/rules.ja.md",
          lang: "ja",
          source: "docs/ja/rules.ja.md",
          guideId: "rules",
        },
      ]);
    });

    it("全ての項目が小見出しへ収まれば Other を作らない", () => {
      const { docs } = build({
        meta: {
          groups: [{ title: "G", sections: ["adr"] }],
          subgroups: { adr: [{ title: "Layer Top", items: ["a"] }] },
        },
        adr: [{ src: "a.md", dst: "docs/portal/guides/a.md" }],
      });

      expect(docs.groups[0].sections[0].subgroups?.map((group) => group.title)).toEqual([
        "Layer Top",
      ]);
    });

    it("meta.subgroups で section 内を小見出しに割る", () => {
      const { docs } = build({
        meta: {
          groups: [{ title: "G", sections: ["controller"] }],
          subgroups: { controller: [{ title: "Layer Top", items: ["controller"] }] },
        },
        controller: [
          { src: "a.md", dst: "docs/portal/guides/controller.md" },
          { src: "b.md", dst: "docs/portal/guides/controller-server.md" },
        ],
      });

      expect(docs.groups[0].sections[0].subgroups?.map((group) => group.title)).toEqual([
        "Layer Top",
        "Other",
      ]);
    });

    it("reference_links は groups から抜き、代表項目へのリンクにする", () => {
      const { docs } = build({
        meta: { reference_links: ["openapi"] },
        openapi: [{ src: "docs/openapi/index.html", dst: "docs/portal/guides/openapi.md" }],
      });

      expect(docs.groups).toEqual([]);
      expect(docs.referenceLinks).toEqual([
        {
          sectionId: "openapi",
          title: "Openapi",
          path: "./guides/openapi.md",
          source: "docs/openapi/index.html",
        },
      ]);
    });

    it("空の manifest でも見出しだけは組み立てる", () => {
      const { docs, warnings } = build({});

      expect(docs.title).toBe("go-boilerplate Documentation");
      expect(docs.groups).toEqual([]);
      expect(warnings).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("meta.groups 未配置の section を Uncategorized へ集約して警告する", () => {
      const { docs, warnings } = build({
        adr: [{ src: "a.md", dst: "docs/portal/guides/a.md" }],
      });

      expect(docs.groups.at(-1)?.title).toBe("Uncategorized");
      expect(warnings.join("\n")).toContain("Uncategorized");
    });

    it("Uncategorized のセクションを見出し順に並べる（manifest の記載順に引きずられない）", () => {
      const { docs } = build({
        zeta: [{ src: "z.md", dst: "docs/portal/guides/z.md" }],
        alpha: [{ src: "a.md", dst: "docs/portal/guides/a.md" }],
      });

      expect(docs.groups.at(-1)?.sections.map((section) => section.id)).toEqual(["alpha", "zeta"]);
    });

    it("中身の無い docs/<dir> を section にしない", () => {
      const { docs, warnings } = build(
        {},
        {
          directories: [{ name: "empty", hasIndexHtml: false, enFiles: [], jaFiles: [] }],
          rootEnFiles: [],
          rootJaFiles: [],
        },
      );

      expect(docs.groups).toEqual([]);
      expect(warnings).toEqual([]);
    });

    it("存在しない section id を無視して警告する", () => {
      const { docs, warnings } = build({ meta: { groups: [{ title: "G", sections: ["ghost"] }] } });

      expect(docs.groups).toEqual([]);
      expect(warnings.join("\n")).toContain('section id "ghost" は存在しない');
    });

    it("複数グループへ書かれた section id は先勝ちにして警告する", () => {
      const { docs, warnings } = build({
        meta: {
          groups: [
            { title: "First", sections: ["adr"] },
            { title: "Second", sections: ["adr"] },
          ],
        },
        adr: [{ src: "a.md", dst: "docs/portal/guides/a.md" }],
      });

      expect(docs.groups.map((group) => group.title)).toEqual(["First"]);
      expect(warnings.join("\n")).toContain("複数グループに記載");
    });

    it("同じ path のカードを 2 枚出さずに警告する", () => {
      const { docs, warnings } = build(
        { meta: { groups: [{ title: "G", sections: ["adr"] }] }, adr: [] },
        {
          directories: [
            { name: "adr", hasIndexHtml: false, enFiles: ["a.md", "a.md"], jaFiles: [] },
          ],
          rootEnFiles: [],
          rootJaFiles: [],
        },
      );

      expect(docs.groups[0].sections[0].items).toHaveLength(1);
      expect(warnings.join("\n")).toContain("重複 item をスキップ");
    });

    it("存在しない subgroups の section id を無視して警告する", () => {
      const { warnings } = build({ meta: { subgroups: { ghost: [{ title: "T", items: [] }] } } });

      expect(warnings.join("\n")).toContain('meta.subgroups: section id "ghost"');
    });

    it("存在しない guide id を小見出しから外して警告する", () => {
      const { docs, warnings } = build({
        meta: {
          groups: [{ title: "G", sections: ["adr"] }],
          subgroups: { adr: [{ title: "T", items: ["ghost"] }] },
        },
        adr: [{ src: "a.md", dst: "docs/portal/guides/a.md" }],
      });

      expect(docs.groups[0].sections[0].subgroups?.map((group) => group.title)).toEqual(["Other"]);
      expect(warnings.join("\n")).toContain('guide id "ghost"');
    });

    it("guide id を導けない項目を空の guide id で小見出しへ束ねない", () => {
      const { docs, warnings } = build({
        meta: {
          groups: [{ title: "G", sections: ["adr"] }],
          subgroups: { adr: [{ title: "T", items: [""] }] },
        },
        adr: [{ src: "docs/adr/.md", dst: "docs/portal/guides/.md" }],
      });

      expect(docs.groups[0].sections[0].subgroups?.map((group) => group.title)).toEqual(["Other"]);
      expect(warnings.join("\n")).toContain('guide id "" は存在しない');
    });

    it("小見出しが 1 つも成立しない section に subgroups を生やさない", () => {
      const { docs } = build({
        meta: {
          groups: [{ title: "G", sections: ["adr"] }],
          subgroups: { adr: [{ title: "T", items: [] }] },
        },
        adr: [],
      });

      expect(docs.groups[0].sections[0]).not.toHaveProperty("subgroups");
    });

    it("存在しない reference_links の section id を無視して警告する", () => {
      const { docs, warnings } = build({ meta: { reference_links: ["ghost"] } });

      expect(docs.referenceLinks).toEqual([]);
      expect(warnings.join("\n")).toContain("meta.reference_links");
    });

    it("項目を持たない section は reference link にしない", () => {
      const { docs } = build({ meta: { reference_links: ["adr"] }, adr: [] });

      expect(docs.referenceLinks).toEqual([]);
    });

    // 複製側（resolveCopyEntries）はこれを拒否する。docs.json 側は生成を止めず、
    // 「カードが出ない」形で現れる。止める判断は複製側に一本化してある。
    it("meta 以外に map を置いた section をカードにしない", () => {
      const { docs } = build({ adr: { src: "a.md" } });

      expect(docs.groups).toEqual([]);
    });
  });
});

describe("isSectionDirectory", () => {
  describe("正常系", () => {
    it("通常のディレクトリは section にする", () => {
      expect(isSectionDirectory("get-started")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("ビューアー自身の生成先は section にしない", () => {
      expect(isSectionDirectory("portal")).toBe(false);
    });

    it("翻訳ツリーは section にしない", () => {
      expect(isSectionDirectory("ja")).toBe(false);
    });
  });
});

describe("isMarkdownFile", () => {
  describe("正常系", () => {
    it("Markdown を対象にする", () => {
      expect(isMarkdownFile("index.md")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("Markdown 以外は対象にしない", () => {
      expect(isMarkdownFile("index.html")).toBe(false);
      expect(isMarkdownFile("notes.md.bak")).toBe(false);
    });
  });
});

describe("sortSectionNames", () => {
  describe("正常系", () => {
    it("読み取り順に依らず表示順へ整列する", () => {
      expect(sortSectionNames(["design", "adr", "get-started"])).toEqual([
        "adr",
        "design",
        "get-started",
      ]);
    });

    it("渡された配列を書き換えない", () => {
      const input = ["b", "a"];

      sortSectionNames(input);

      expect(input).toEqual(["b", "a"]);
    });
  });

  describe("異常系", () => {
    it("空なら空を返す", () => {
      expect(sortSectionNames([])).toEqual([]);
    });
  });
});
