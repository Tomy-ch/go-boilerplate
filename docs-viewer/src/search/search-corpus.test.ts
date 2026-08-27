import { describe, expect, it } from "vitest";
import type { PortalGroup } from "../docs-json/docs-json";
import { buildSearchCorpus, searchKeys } from "./search-corpus";

const groups: PortalGroup[] = [
  {
    title: "Architecture",
    slug: "architecture",
    sections: [
      {
        id: "adr",
        slug: "adr",
        title: "ADR",
        items: [{ name: "0001", path: "./guides/0001.md", lang: "ja" }],
        subgroups: [
          {
            title: "Layer",
            items: [{ name: "0021", path: "./guides/0021.md", lang: "ja" }],
          },
        ],
      },
    ],
  },
];

describe("buildSearchCorpus", () => {
  describe("正常系", () => {
    it("項目に所属する section と group の名前を畳み込む", () => {
      const corpus = buildSearchCorpus(groups);

      expect(corpus[0]).toMatchObject({
        name: "0001",
        sectionTitle: "ADR",
        groupTitle: "Architecture",
      });
    });
    it("subgroup 配下の項目も平坦化して含める", () => {
      const corpus = buildSearchCorpus(groups);

      expect(corpus.map((entry) => entry.name)).toEqual(["0001", "0021"]);
    });
    it("subgroups を持たない section を扱える", () => {
      const corpus = buildSearchCorpus([
        {
          title: "Architecture",
          slug: "architecture",
          sections: [
            {
              id: "adr",
              slug: "adr",
              title: "ADR",
              items: [{ name: "0001", path: "./guides/0001.md", lang: "ja" }],
            },
          ],
        },
      ]);

      expect(corpus).toHaveLength(1);
    });
    it("group が無ければ空のコーパスを返す", () => {
      expect(buildSearchCorpus([])).toEqual([]);
    });
  });
});

describe("searchKeys", () => {
  describe("正常系", () => {
    it("項目名と所属名と出所を検索対象に含める", () => {
      expect(searchKeys).toEqual(["name", "sectionTitle", "groupTitle", "source", "path"]);
    });
  });
});
