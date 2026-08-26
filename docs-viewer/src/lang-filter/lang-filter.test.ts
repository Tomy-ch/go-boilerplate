import { describe, expect, it } from "vitest";
import type { PortalGroup, PortalItem } from "../docs-json/docs-json";
import { applyLangFilter, effectiveLangFor, filterItemsByLang } from "./lang-filter";

const en: PortalItem = { name: "guide", path: "./guides/guide.md", lang: "en" };
const ja: PortalItem = { name: "手引き", path: "./guides/guide.ja.md", lang: "ja" };
const always: PortalItem = { name: "coverage", path: "./coverage/index.html", lang: "all" };

function groupWith(sections: PortalGroup["sections"]): PortalGroup {
  return { title: "Architecture", slug: "architecture", sections };
}

describe("effectiveLangFor", () => {
  describe("正常系", () => {
    it("EN を選んでいれば項目の内訳によらず EN を返す", () => {
      expect(effectiveLangFor([ja], "EN")).toBe("EN");
    });
    it("JA を選び JA の項目がある section では JA を返す", () => {
      expect(effectiveLangFor([en, ja], "JA")).toBe("JA");
    });
    it("JA を選んでも JA の項目が無い section では EN へ落とす", () => {
      expect(effectiveLangFor([en, always], "JA")).toBe("EN");
    });
  });
});

describe("filterItemsByLang", () => {
  describe("正常系", () => {
    it("EN では en と all の項目だけを残す", () => {
      expect(filterItemsByLang([en, ja, always], "EN")).toEqual([always, en]);
    });
    it("JA では ja と all の項目だけを残す", () => {
      expect(filterItemsByLang([en, ja, always], "JA")).toEqual([always, ja]);
    });
    it("空の入力を空のまま返す", () => {
      expect(filterItemsByLang([], "EN")).toEqual([]);
    });
  });
});

describe("applyLangFilter", () => {
  describe("正常系", () => {
    it("section 単位で決めた言語を subgroup にも適用する", () => {
      const filtered = applyLangFilter(
        [
          groupWith([
            {
              id: "adr",
              slug: "adr",
              title: "ADR",
              items: [en, ja],
              subgroups: [{ title: "Layer", items: [en, ja] }],
            },
          ]),
        ],
        "JA",
      );

      expect(filtered[0]?.sections[0]?.items).toEqual([ja]);
      expect(filtered[0]?.sections[0]?.subgroups?.[0]?.items).toEqual([ja]);
    });
    it("JA の項目が無い section では subgroup も EN へ揃える", () => {
      const filtered = applyLangFilter(
        [
          groupWith([
            {
              id: "adr",
              slug: "adr",
              title: "ADR",
              items: [en],
              subgroups: [{ title: "Layer", items: [en] }],
            },
          ]),
        ],
        "JA",
      );

      expect(filtered[0]?.sections[0]?.subgroups?.[0]?.items).toEqual([en]);
    });
    it("items が空でも subgroup に中身が残る section を保持する", () => {
      const filtered = applyLangFilter(
        [
          groupWith([
            {
              id: "adr",
              slug: "adr",
              title: "ADR",
              items: [],
              subgroups: [{ title: "Layer", items: [en] }],
            },
          ]),
        ],
        "EN",
      );

      expect(filtered[0]?.sections).toHaveLength(1);
    });
    it("subgroups を持たない section を null のまま返す", () => {
      const filtered = applyLangFilter(
        [groupWith([{ id: "adr", slug: "adr", title: "ADR", items: [en] }])],
        "EN",
      );

      expect(filtered[0]?.sections[0]?.subgroups).toBeNull();
    });
  });

  describe("異常系", () => {
    it("中身が残らなかった section と、その結果空になった group を落とす", () => {
      const filtered = applyLangFilter(
        [groupWith([{ id: "adr", slug: "adr", title: "ADR", items: [ja], subgroups: [] }])],
        "EN",
      );

      expect(filtered).toEqual([]);
    });
    it("subgroups を持たない section で items も残らなければ落とす", () => {
      const filtered = applyLangFilter(
        [groupWith([{ id: "adr", slug: "adr", title: "ADR", items: [ja] }])],
        "EN",
      );

      expect(filtered).toEqual([]);
    });
  });
});
