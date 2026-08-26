import { describe, expect, it } from "vitest";
import { z } from "zod";
import { parseDocsJson } from "./docs-json";

const minimalDocs = {
  title: "Docs",
  subtitle: "boilerplate documentation",
  groups: [
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
  ],
};

describe("parseDocsJson", () => {
  describe("正常系", () => {
    it("group と section を持つ生成物を読み取る", () => {
      const docs = parseDocsJson(minimalDocs);

      expect(docs.title).toBe("Docs");
      expect(docs.groups[0]?.sections[0]?.items[0]?.name).toBe("0001");
    });
    it("referenceLinks が無い生成物を空配列として読み取る", () => {
      const docs = parseDocsJson(minimalDocs);

      expect(docs.referenceLinks).toEqual([]);
    });
    it("subgroups を省いた section と null を置いた section の双方を許す", () => {
      const docs = parseDocsJson({
        ...minimalDocs,
        groups: [
          {
            title: "Architecture",
            slug: "architecture",
            sections: [
              { id: "a", slug: "a", title: "A", items: [], subgroups: null },
              { id: "b", slug: "b", title: "B", items: [] },
            ],
          },
        ],
      });

      expect(docs.groups[0]?.sections).toHaveLength(2);
    });
  });

  describe("異常系", () => {
    it("未知の言語を持つ項目を拒否する", () => {
      const invalid = {
        ...minimalDocs,
        groups: [
          {
            title: "Architecture",
            slug: "architecture",
            sections: [
              {
                id: "adr",
                slug: "adr",
                title: "ADR",
                items: [{ name: "0001", path: "./guides/0001.md", lang: "fr" }],
              },
            ],
          },
        ],
      };

      expect(() => parseDocsJson(invalid)).toThrow(z.ZodError);
    });
    it("groups を欠いた生成物を拒否する", () => {
      expect(() => parseDocsJson({ title: "Docs", subtitle: "x" })).toThrow(z.ZodError);
    });
  });
});
