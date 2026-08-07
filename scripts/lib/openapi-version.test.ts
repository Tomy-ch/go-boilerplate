import { describe, expect, it } from "vitest";

import { deriveVersion, readVersion, replaceVersion } from "./openapi-version";

const SPEC = ["openapi: 3.1.0", "info:", "  title: go-boilerplate", "  version: 1.0.0", "paths: {}"].join("\n");

describe("deriveVersion", () => {
  describe("正常系", () => {
    it("リリースブランチ名から SemVer を取り出す", () => {
      expect(deriveVersion("release/v2.2.0")).toBe("2.2.0");
    });
    it("各桁のゼロを受け付ける", () => {
      expect(deriveVersion("release/v0.0.0")).toBe("0.0.0");
    });
  });

  describe("異常系", () => {
    it("リリースブランチ以外は刻まない", () => {
      expect(deriveVersion("feature/1234-something")).toBeNull();
    });
    it("空の ref を刻まない", () => {
      expect(deriveVersion("")).toBeNull();
    });
    it("SemVer として読めない release ブランチを刻まない", () => {
      expect(deriveVersion("release/v2.2")).toBeNull();
    });
    it("プレリリース識別子付きを刻まない（SHA 等を version に混ぜない）", () => {
      expect(deriveVersion("release/v2.2.0-rc1")).toBeNull();
    });
    it("先頭ゼロを刻まない", () => {
      expect(deriveVersion("release/v01.2.3")).toBeNull();
    });
  });
});

describe("readVersion", () => {
  describe("正常系", () => {
    it("info 直下の version を読む", () => {
      expect(readVersion(SPEC)).toBe("1.0.0");
    });
  });

  describe("異常系", () => {
    it("version 行が無ければ null を返す", () => {
      expect(readVersion("openapi: 3.1.0\ninfo:\n  title: x\n")).toBeNull();
    });
    it("桁の違う version 行（info 直下でない）を拾わない", () => {
      expect(readVersion("components:\n  schemas:\n    X:\n      version: 9.9.9\n")).toBeNull();
    });
  });
});

describe("replaceVersion", () => {
  describe("正常系", () => {
    it("version 行だけを差し替える", () => {
      expect(replaceVersion(SPEC, "2.2.0")).toBe(SPEC.replace("  version: 1.0.0", "  version: 2.2.0"));
    });
    it("置換パターンを含む値も文字列として書き込む", () => {
      expect(replaceVersion(SPEC, "$&")).toContain("  version: $&");
    });
  });
});
