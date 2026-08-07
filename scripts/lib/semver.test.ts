import { describe, expect, it } from "vitest";

import { bumpVersion, isBumpType, normalizeVersion } from "./semver";

describe("normalizeVersion", () => {
  describe("正常系", () => {
    it("先頭の v を落とす", () => {
      expect(normalizeVersion("v1.2.3")).toBe("1.2.3");
    });
    it("v の無い表記をそのまま通す", () => {
      expect(normalizeVersion("1.2.3")).toBe("1.2.3");
    });
    it("各桁のゼロを受け付ける", () => {
      expect(normalizeVersion("v0.0.0")).toBe("0.0.0");
    });
  });

  describe("異常系", () => {
    it("桁が足りない表記を拒否する", () => {
      expect(() => normalizeVersion("v1.2")).toThrow("X.Y.Z");
    });
    it("先頭ゼロを拒否する（タグ名の揺れを次版へ持ち込ませない）", () => {
      expect(() => normalizeVersion("v01.2.3")).toThrow("X.Y.Z");
    });
    it("プレリリース識別子付きを拒否する", () => {
      expect(() => normalizeVersion("v1.2.3-rc.1")).toThrow("X.Y.Z");
    });
    it("空文字を拒否する", () => {
      expect(() => normalizeVersion("")).toThrow("X.Y.Z");
    });
  });
});

describe("isBumpType", () => {
  describe("正常系", () => {
    it("patch / minor / major を繰り上げ単位と判定する", () => {
      expect(["patch", "minor", "major"].every(isBumpType)).toBe(true);
    });
  });

  describe("異常系", () => {
    it("未知の単位を弾く", () => {
      expect(isBumpType("build")).toBe(false);
    });
  });
});

describe("bumpVersion", () => {
  describe("正常系", () => {
    it("patch は最下位だけを進める", () => {
      expect(bumpVersion("v1.2.3", "patch")).toBe("v1.2.4");
    });
    it("minor は patch を 0 へ戻す", () => {
      expect(bumpVersion("v1.2.3", "minor")).toBe("v1.3.0");
    });
    it("major は minor と patch を 0 へ戻す", () => {
      expect(bumpVersion("v1.2.3", "major")).toBe("v2.0.0");
    });
    it("繰り上がりを桁上げとして扱わない（9 の次は 10）", () => {
      expect(bumpVersion("v1.9.9", "patch")).toBe("v1.9.10");
      expect(bumpVersion("v1.9.9", "minor")).toBe("v1.10.0");
    });
    it("v の無い入力でも v 付きで返す", () => {
      expect(bumpVersion("1.2.3", "patch")).toBe("v1.2.4");
    });
  });

  describe("異常系", () => {
    it("不正なバージョンを繰り上げない", () => {
      expect(() => bumpVersion("latest", "patch")).toThrow("X.Y.Z");
    });
  });
});
