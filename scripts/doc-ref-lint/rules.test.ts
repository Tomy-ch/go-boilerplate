import { describe, expect, it } from "vitest";

import { adrIndex, checkDocRoutes, checkPathReferences, checkReferences, checkStructuredReferences, isEligible, normalizeReferences } from "./rules";

const ADR = ["ADR", "-0006"].join("");
const ADRS = adrIndex(["docs/adr/0006-structural-safety-via-tooling.md"]);

describe("isEligible", () => {
  describe("正常系", () => {
    it("英語正本の検査対象ファイルを受け入れる", () => expect(isEligible("docs/design/worker.md")).toBe(true));
  });
  describe("異常系", () => {
    it("テスト・生成物・契約文書・非対象拡張子を除外する", () => {
      expect(isEligible("scripts/doc-ref-lint/rules.test.ts")).toBe(false);
      expect(isEligible("internal/a.gen.go")).toBe(false);
      expect(isEligible("database/gen/a.gen.sql")).toBe(false);
      expect(isEligible("AGENTS.md")).toBe(false);
      expect(isEligible("README")).toBe(false);
    });
  });
});

describe("adrIndex", () => {
  describe("正常系", () => {
    it("ADR の番号と slug を索引化する", () => expect(ADRS.get("0006")).toBe("structural-safety-via-tooling"));
  });
  describe("異常系", () => {
    it("ADR 形式でないパスを無視する", () => expect(adrIndex(["docs/rules.md"])).toEqual(new Map()));
  });
});

describe("normalizeReferences", () => {
  describe("正常系", () => {
    it("番号だけの参照に slug を補う", () => expect(normalizeReferences(ADR, ADRS)).toBe(`${ADR} (structural-safety-via-tooling)`));
    it("既に slug を持つ参照を保持する", () => expect(normalizeReferences(`${ADR} (structural-safety-via-tooling)`, ADRS)).toBe(`${ADR} (structural-safety-via-tooling)`));
  });
  describe("異常系", () => {
    it("未知の番号を変更しない", () => expect(normalizeReferences(["ADR", "-9999"].join(""), ADRS)).toBe("ADR-9999"));
  });
});

describe("checkReferences", () => {
  describe("正常系", () => {
    it("番号と slug が一致する参照を通す", () => expect(checkReferences("a.md", `${ADR} (structural-safety-via-tooling)`, ADRS)).toEqual([]));
  });
  describe("異常系", () => {
    it("未知の番号と slug 不一致を報告する", () => {
      expect(checkReferences("a.md", "ADR-9999", ADRS)[0].message).toContain("does not exist");
      expect(checkReferences("a.md", `${ADR} (other)`, ADRS)[0].message).toContain("must include");
    });
    it("参照の直後にない slug 注釈を一致とみなさない", () => {
      expect(checkReferences("a.md", `${ADR} と無関係な (structural-safety-via-tooling)`, ADRS)[0].message).toContain("must include");
    });
  });
});

describe("checkPathReferences", () => {
  describe("正常系", () => {
    it("番号どおりの slug を綴ったパス参照を通す", () => {
      expect(checkPathReferences("a.md", "docs/adr/0006-structural-safety-via-tooling.md", ADRS)).toEqual([]);
    });
    it("ADR を指さないパスは見ない", () => {
      expect(checkPathReferences("a.md", "docs/design/0006-other.md", ADRS)).toEqual([]);
    });
  });
  describe("異常系", () => {
    it("番号と slug が食い違うリンク先を報告する", () => {
      const source = `[${ADR} (structural-safety-via-tooling)](../docs/adr/0006-something-else.md)`;
      expect(checkPathReferences("a.md", source, ADRS)[0].message).toContain("must be 0006-structural-safety-via-tooling.md");
    });
    it("存在しない番号のパス参照を報告する", () => {
      expect(checkPathReferences("a.md", "docs/adr/9999-nope.md", ADRS)[0].message).toContain("does not exist");
    });
  });
});

describe("checkStructuredReferences", () => {
  const entry = (id: string, slug: string): string => `    interpreted_by:\n      - kind: adr\n        id: "${id}"\n        slug: ${slug}\n`;

  describe("正常系", () => {
    it("番号と slug が一致する構造化エントリを通す", () => {
      expect(checkStructuredReferences("a.yaml", entry("0006", "structural-safety-via-tooling"), ADRS)).toEqual([]);
    });
    it("adr 以外の kind を見ない", () => {
      const source = '      - kind: readme\n        path: internal/domain/README.md\n        section: "Entity"\n';
      expect(checkStructuredReferences("a.yaml", source, ADRS)).toEqual([]);
    });
    it("ファイル末尾で途切れる構造化エントリを読み切る", () => {
      const source = '      - kind: adr\n        id: "0006"\n        slug: structural-safety-via-tooling';
      expect(checkStructuredReferences("a.yaml", source, ADRS)).toEqual([]);
    });
    it("隣の要素のフィールドを自分のものとして拾わない", () => {
      const source = `${entry("0006", "structural-safety-via-tooling")}      - kind: readme\n        path: docs/rules.md\n`;
      expect(checkStructuredReferences("a.yaml", source, ADRS)).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("slug が別の番号のものである場合に番号側を直すよう報告する", () => {
      const source = entry("0009", "structural-safety-via-tooling");
      expect(checkStructuredReferences("a.yaml", source, ADRS)[0].message).toContain('is id "0006", not "0009"');
    });
    it("番号が実在し slug だけ食い違う場合を報告する", () => {
      expect(checkStructuredReferences("a.yaml", entry("0006", "other-slug"), ADRS)[0].message).toContain("not other-slug");
    });
    it("番号も slug も実在しない場合を報告する", () => {
      expect(checkStructuredReferences("a.yaml", entry("9999", "other-slug"), ADRS)[0].message).toContain("name no ADR");
    });
    it("slug を欠いた構造化エントリを報告する", () => {
      const source = '      - kind: adr\n        id: "0006"\n';
      expect(checkStructuredReferences("a.yaml", source, ADRS)[0].message).toContain("must carry both id and slug");
    });
  });
});

describe("checkDocRoutes", () => {
  const present = new Set(["docs/design/rest.md", "docs/design/auth.md"]);

  describe("正常系", () => {
    it("実在する文書へ経路を張った行を通す", () => {
      expect(checkDocRoutes("internal/controller/handler/* = docs/design/rest.md # why\n", present)).toEqual([]);
    });
    it("1 行に複数の文書を並べた経路を通す", () => {
      const source = "internal/x/* = docs/design/rest.md, docs/design/auth.md # why\n";
      expect(checkDocRoutes(source, present)).toEqual([]);
    });
    it("コメント行と空行を経路として読まない", () => {
      expect(checkDocRoutes("# a = docs/nope.md\n\n", present)).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("存在しない文書を指した経路を行番号付きで報告する", () => {
      const findings = checkDocRoutes("a/* = docs/design/rest.md\nb/* = docs/design/gone.md # why\n", present);

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("L2");
      expect(findings[0].message).toContain("docs/design/gone.md");
    });
    it("複数並べたうちの存在しない側だけを報告する", () => {
      const findings = checkDocRoutes("a/* = docs/design/rest.md, docs/design/gone.md\n", present);

      expect(findings).toHaveLength(1);
      expect(findings[0].message).toContain("docs/design/gone.md");
    });
  });
});
