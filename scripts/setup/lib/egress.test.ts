import { describe, expect, it } from "vitest";

import { removeEgressSections } from "./egress";

describe("removeEgressSections", () => {
  const ssot = [
    "[class.base]",
    'hosts = ["a:443"]',
    "",
    '[job."sonarqube.yaml:sonarqube"]',
    'classes = ["mise"]',
    "extra = [",
    '  "sonarcloud.io:443",',
    "]",
    "",
    '[job."sql-lint.yaml:sql-lint"]',
    'classes = ["mise"]',
    "",
  ].join("\n");

  describe("正常系", () => {
    it("セクションを本文ごと落とし、後続のセクションは残す", () => {
      const result = removeEgressSections(ssot, ["sonarqube.yaml:sonarqube"]);

      expect(result).not.toContain("sonarqube");
      expect(result).toContain('[job."sql-lint.yaml:sql-lint"]');
      expect(result).toContain("[class.base]");
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
