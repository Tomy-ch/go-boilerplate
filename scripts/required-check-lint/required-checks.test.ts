import { describe, expect, it } from "vitest";

import { checkRequiredChecks } from "./required-checks";

const required = ["scan"];
const main = { file: ".github/workflows/scan.yaml", source: "jobs:\n  scan:\n    steps: []" };
const guard = {
  file: ".github/workflows/scan-guard.yaml",
  source: "# required-check guard\njobs:\n  scan:\n    steps: []",
};

describe("checkRequiredChecks", () => {
  describe("正常系", () => {
    it("本体と guard が各 1 件なら通す", () => {
      expect(checkRequiredChecks(required, [main, guard])).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("guard が無ければ失敗する", () => {
      expect(checkRequiredChecks(required, [main])[0]?.message).toContain("skip guard job は 1 件必要");
    });

    it("required にない guard job を失敗にする", () => {
      const extra = { ...guard, source: "# required-check guard\njobs:\n  other:\n    steps: []" };
      expect(checkRequiredChecks(required, [main, extra])[0]?.message).toContain("required context ではありません");
    });
  });
});
