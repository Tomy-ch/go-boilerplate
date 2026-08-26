import { describe, expect, it } from "vitest";
import type { PortalGroup } from "../docs-json/docs-json";
import { formatHashRoute, parseHashRoute, resolveActiveGroupSlug } from "./hash-route";

const groups: PortalGroup[] = [
  { title: "Get Started", slug: "get-started", sections: [] },
  { title: "Architecture", slug: "architecture", sections: [] },
];

describe("parseHashRoute", () => {
  describe("正常系", () => {
    it("group と section を含むハッシュを両方に分解する", () => {
      expect(parseHashRoute("#/architecture/adr")).toEqual({
        groupSlug: "architecture",
        sectionSlug: "adr",
      });
    });
    it("group だけのハッシュでは section を未指定にする", () => {
      expect(parseHashRoute("#/architecture")).toEqual({
        groupSlug: "architecture",
        sectionSlug: null,
      });
    });
    it("先頭のスラッシュが無いハッシュも解釈する", () => {
      expect(parseHashRoute("#architecture")).toEqual({
        groupSlug: "architecture",
        sectionSlug: null,
      });
    });
    it("空のハッシュを未指定として扱う", () => {
      expect(parseHashRoute("")).toEqual({ groupSlug: "", sectionSlug: null });
    });
  });
});

describe("formatHashRoute", () => {
  describe("正常系", () => {
    it("section を指定すると group と繋いだハッシュを組む", () => {
      expect(formatHashRoute("architecture", "adr")).toBe("#/architecture/adr");
    });
    it("section を省くと group だけのハッシュを組む", () => {
      expect(formatHashRoute("architecture")).toBe("#/architecture");
    });
  });
});

describe("resolveActiveGroupSlug", () => {
  describe("正常系", () => {
    it("要求された group が表示可能ならそれを選ぶ", () => {
      expect(resolveActiveGroupSlug(groups, "architecture")).toBe("architecture");
    });
    it("要求された group が消えていれば先頭の group へ寄せる", () => {
      expect(resolveActiveGroupSlug(groups, "removed")).toBe("get-started");
    });
    it("表示可能な group が無ければ null を返す", () => {
      expect(resolveActiveGroupSlug([], "architecture")).toBeNull();
    });
  });
});
