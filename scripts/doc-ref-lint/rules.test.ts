import { describe, expect, it } from "vitest";

import { adrIndex, checkReferences, checkTranslationExclusions, checkTranslations, expectedTranslation, isEligible, normalizeReferences, translationExclusion } from "./rules";

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
  });
});

describe("expectedTranslation", () => {
  describe("正常系", () => {
    it("通常の docs を日本語ミラーへ対応付ける", () => expect(expectedTranslation("docs/design/worker.md")).toBe("docs/ja/design/worker.ja.md"));
  });
  describe("異常系", () => {
    it("対象外の領域と形式を除外する", () => {
      expect(expectedTranslation("internal/README.md")).toBeNull();
      expect(expectedTranslation("docs/design/worker.txt")).toBeNull();
      expect(expectedTranslation("docs/adr/0001-a.md")).toBeNull();
      expect(expectedTranslation("docs/godoc/a.md")).toBeNull();
      expect(expectedTranslation("docs/spec/user/domain.md")).toBeNull();
    });
  });
});

describe("translationExclusion", () => {
  describe("正常系", () => {
    it("意図的に英語のみの仕様領域の理由を返す", () => expect(translationExclusion("docs/spec/user/domain.md")).toContain("English-only"));
  });
  describe("異常系", () => {
    it("対象外の文書には除外理由を返さない", () => expect(translationExclusion("docs/design/worker.md")).toBeNull());
  });
});

describe("checkTranslations", () => {
  describe("正常系", () => {
    it("正本と翻訳が一対一なら通す", () => expect(checkTranslations(["docs/design/worker.md", "docs/ja/design/worker.ja.md"])).toEqual([]));
  });
  describe("異常系", () => {
    it("欠落と孤立した翻訳を報告する", () => {
      const findings = checkTranslations(["docs/design/worker.md", "docs/ja/design/orphan.ja.md"]);
      expect(findings.map(({ message }) => message)).toContain("missing docs/ja/design/worker.ja.md");
      expect(findings.map(({ message }) => message)).toContain("orphan translation for docs/design/orphan.md");
    });
    it("対象外の ADR 翻訳を孤立として扱わない", () => expect(checkTranslations(["docs/ja/adr/0001-a.ja.md"])).toEqual([]));
  });
});

describe("checkTranslationExclusions", () => {
  describe("正常系", () => {
    it("根拠のある現役の除外を通す", () => expect(checkTranslationExclusions(["docs/spec/user/domain.md"])).toEqual([]));
  });
  describe("異常系", () => {
    it("対象がない除外を古い設定として報告する", () => expect(checkTranslationExclusions([])[0].message).toBe("stale translation exclusion"));
  });
});
