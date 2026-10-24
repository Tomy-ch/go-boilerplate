import { describe, expect, it } from "vitest";

import { removeEgressSections } from "./egress";

describe("removeEgressSections", () => {
  const ssot = [
    "[class.base]",
    'hosts = ["a"]',
    "",
    '[job."sonarqube.yaml:sonarqube"]',
    'classes = ["mise"]',
    "extra = [",
    '  "sonarcloud.io",',
    "]",
    'sudo = "needs docker"',
    "",
    '[ip."127.0.0.100"]',
    'reason = "loopback"',
    "",
    '[job."sql-lint.yaml:sql-lint"]',
    'classes = ["mise"]',
    "",
  ].join("\n");

  describe("正常系", () => {
    it("セクションを本文ごと落とし、後続のセクションは残す", () => {
      const result = removeEgressSections(ssot, ["sonarqube.yaml:sonarqube"]);

      expect(result).toBe(
        [
          "[class.base]",
          'hosts = ["a"]',
          "",
          '[ip."127.0.0.100"]',
          'reason = "loopback"',
          "",
          '[job."sql-lint.yaml:sql-lint"]',
          'classes = ["mise"]',
          "",
        ].join("\n"),
      );
    });

    it("複数のキーを渡すと該当するセクションをすべて落とす", () => {
      const result = removeEgressSections(ssot, ["sonarqube.yaml:sonarqube", "sql-lint.yaml:sql-lint"]);

      expect(result).toBe(["[class.base]", 'hosts = ["a"]', "", '[ip."127.0.0.100"]', 'reason = "loopback"', ""].join("\n"));
    });

    it("セクションの中の sudo の宣言も一緒に落とす", () => {
      expect(removeEgressSections(ssot, ["sonarqube.yaml:sonarqube"])).not.toContain("needs docker");
    });

    it("直後の IP のセクションを巻き込まない", () => {
      const result = removeEgressSections(ssot, ["sonarqube.yaml:sonarqube"]);

      expect(result).toContain('[ip."127.0.0.100"]');
      expect(result).toContain('reason = "loopback"');
    });

    it("キーが空なら本文を変えない", () => {
      expect(removeEgressSections(ssot, [])).toBe(ssot);
    });
  });

  describe("異常系", () => {
    it("存在しないセクションを渡しても他を巻き込まない", () => {
      expect(removeEgressSections(ssot, ["absent.yaml:absent"])).toBe(ssot);
    });
  });
});
