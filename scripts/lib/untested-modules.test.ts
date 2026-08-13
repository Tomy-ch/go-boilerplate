import { describe, expect, it } from "vitest";

import { ENTRYPOINT_PATTERNS, EXCLUDED_FROM_CHECKS, NON_DECIDING_MODULES } from "./untested-modules";

describe("ENTRYPOINT_PATTERNS", () => {
  describe("正常系", () => {
    it("判定を持たない CLI 入口のパターンを列挙する", () => {
      expect(ENTRYPOINT_PATTERNS).toEqual(["*/index.ts", "*/*/index.ts", "portal/gen-*.ts"]);
    });
  });

  describe("異常系", () => {
    it("判定モジュールを入口として除外しない", () => expect(ENTRYPOINT_PATTERNS).not.toContain("lib/one-to-one.ts"));
  });
});

describe("NON_DECIDING_MODULES", () => {
  describe("正常系", () => {
    it("判定を持たない runtime を除外する", () => expect(NON_DECIDING_MODULES).toEqual(["setup/lib/runtime.ts"]));
  });

  describe("異常系", () => {
    it("判定を持つ rules を除外しない", () => expect(NON_DECIDING_MODULES).not.toContain("doc-ref-lint/rules.ts"));
  });
});

describe("EXCLUDED_FROM_CHECKS", () => {
  describe("正常系", () => {
    it("入口と非判定モジュールを一つの除外集合にする", () => {
      expect(EXCLUDED_FROM_CHECKS).toEqual([...ENTRYPOINT_PATTERNS, ...NON_DECIDING_MODULES]);
    });
  });

  describe("異常系", () => {
    it("判定モジュールを除外集合に含めない", () => expect(EXCLUDED_FROM_CHECKS).not.toContain("doc-ref-lint/rules.ts"));
  });
});
