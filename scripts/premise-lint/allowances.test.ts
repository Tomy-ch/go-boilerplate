import { describe, expect, it } from "vitest";

import { ALLOWANCES } from "./allowances";
import { hasPremisePhrase } from "./rules";

describe("ALLOWANCES", () => {
  describe("正常系", () => {
    // 理由の書けない宣言は、次に読む人には「なぜ許されているか分からない穴」にしか見えない。
    it("すべての宣言が対象・本文・理由を持つ", () => {
      for (const entry of ALLOWANCES) {
        expect(entry.file, JSON.stringify(entry)).not.toBe("");
        expect(entry.contains, entry.file).not.toBe("");
        expect(entry.reason, entry.file).not.toBe("");
      }
    });

    // 前提の言い回しを含まない行を宣言しても、検査は何も止めない。宣言のほうが古い合図。
    it("宣言した本文が実際に前提の言い回しを含む", () => {
      for (const entry of ALLOWANCES) {
        expect(hasPremisePhrase(entry.contains), entry.file).toBe(true);
      }
    });
  });
});
