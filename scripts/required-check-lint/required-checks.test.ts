import { describe, expect, it } from "vitest";

import { checkRequiredChecks } from "./required-checks";

const required = ["scan"];
const main = {
  file: ".github/workflows/scan.yaml",
  source: ["on:", "  pull_request:", "", "jobs:", "  scan:", "    steps: []"].join("\n"),
};

describe("checkRequiredChecks", () => {
  describe("正常系", () => {
    it("フィルタの無い pull_request で報告する job が 1 件なら通す", () => {
      expect(checkRequiredChecks(required, [main])).toEqual([]);
    });

    it("push 側のフィルタは残っていてよい", () => {
      const withPush = {
        ...main,
        source: [
          "on:",
          "  push:",
          "    paths:",
          "      - '**/*.go'",
          "  pull_request:",
          "",
          "jobs:",
          "  scan:",
          "    steps: []",
        ].join("\n"),
      };
      expect(checkRequiredChecks(required, [withPush])).toEqual([]);
    });

    it("required でない job は起動条件を問わない", () => {
      const other = {
        file: ".github/workflows/other.yaml",
        source: ["on:", "  pull_request:", "    paths:", "      - 'x'", "", "jobs:", "  other:", "    steps: []"].join(
          "\n",
        ),
      };
      expect(checkRequiredChecks(required, [main, other])).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("jobs 宣言のないワークフローを失敗にする", () => {
      const missingJobs = { file: ".github/workflows/invalid.yaml", source: "name: invalid" };
      expect(checkRequiredChecks(required, [main, missingJobs])[0]).toMatchObject({
        file: missingJobs.file,
        line: 1,
        message: "jobs: が見つかりません",
      });
    });

    it("報告する job が無ければ失敗する", () => {
      expect(checkRequiredChecks(required, [])[0]?.message).toContain("job は 1 件必要です（実際: 0）");
    });

    it("報告する job が複数あれば失敗する", () => {
      const duplicate = { ...main, file: ".github/workflows/scan-copy.yaml" };
      expect(checkRequiredChecks(required, [main, duplicate])[0]?.message).toContain("（実際: 2）");
    });

    it("pull_request に paths が残っていれば失敗する", () => {
      const filtered = {
        ...main,
        source: [
          "on:",
          "  pull_request:",
          "    paths:",
          "      - '**/*.go'",
          "",
          "jobs:",
          "  scan:",
          "    steps: []",
        ].join("\n"),
      };
      const finding = checkRequiredChecks(required, [filtered])[0];
      expect(finding?.line).toBe(3);
      expect(finding?.message).toContain("`paths` が残っています");
    });

    it("pull_request に branches が残っていれば失敗する", () => {
      const filtered = {
        ...main,
        source: [
          "on:",
          "  pull_request:",
          "    branches:",
          "      - develop",
          "",
          "jobs:",
          "  scan:",
          "    steps: []",
        ].join("\n"),
      };
      expect(checkRequiredChecks(required, [filtered])[0]?.message).toContain("`branches` が残っています");
    });

    it("pull_request トリガーが無ければ失敗する", () => {
      const pushOnly = {
        ...main,
        source: ["on:", "  push:", "", "jobs:", "  scan:", "    steps: []"].join("\n"),
      };
      expect(checkRequiredChecks(required, [pushOnly])[0]?.message).toContain(
        "pull_request トリガーがありません",
      );
    });
  });
});
