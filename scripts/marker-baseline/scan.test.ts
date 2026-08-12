import { describe, expect, it } from "vitest";

import { EXCLUDED_DIRECTORIES, diffBaseline } from "./rules";
import { readBaseline, scanRepository } from "./scan";

describe("scanRepository", () => {
  describe("正常系", () => {
    it("実ツリーがベースラインと一致する", () => {
      expect(diffBaseline(scanRepository(), readBaseline()).join("\n")).toBe("");
    });

    it("除外ディレクトリ配下を 1 件も含まない", () => {
      const offenders = Object.keys(scanRepository()).filter((rel) =>
        [...EXCLUDED_DIRECTORIES].some((dir) => rel.startsWith(`${dir}/`) || rel.includes(`/${dir}/`)),
      );

      expect(offenders).toEqual([]);
    });
  });
});

describe("readBaseline", () => {
  describe("正常系", () => {
    it("0 件の項目を持たない", () => {
      const zeros = Object.entries(readBaseline())
        .filter(([, count]) => count <= 0)
        .map(([file]) => file);

      expect(zeros).toEqual([]);
    });
  });
});
