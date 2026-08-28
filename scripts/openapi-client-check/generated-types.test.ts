import { describe, expect, it } from "vitest";

import { DegenerateOutputError, declaredTypeNames, missingTypes } from "./generated-types";

const SAMPLE = [
  "export interface DeliveryEvent {\n  eventId: string;\n}\n",
  "export type StreamCursor = string;\nexport interface ControlEvent {}\n",
];

describe("DegenerateOutputError", () => {
  describe("正常系", () => {
    it("名前とメッセージを保つ", () => {
      const error = new DegenerateOutputError("empty");
      expect(error.name).toBe("DegenerateOutputError");
      expect(error.message).toBe("empty");
      expect(error).toBeInstanceOf(Error);
    });
  });
});

describe("declaredTypeNames", () => {
  describe("正常系", () => {
    it("interface と type の宣言名を集める", () => {
      expect([...declaredTypeNames(SAMPLE)].sort()).toEqual(["ControlEvent", "DeliveryEvent", "StreamCursor"]);
    });

    it("宣言以外の行は数えない", () => {
      expect(declaredTypeNames(["const x = 1;\n// export interface Fake {}\n"]).size).toBe(0);
    });
  });
});

describe("missingTypes", () => {
  describe("正常系", () => {
    it("期待型がすべてあれば空", () => {
      expect(missingTypes(SAMPLE, ["DeliveryEvent", "ControlEvent", "StreamCursor"])).toEqual([]);
    });

    it("無い型だけを理由付きで返す", () => {
      expect(missingTypes(SAMPLE, ["DeliveryEvent", "Missing"])).toEqual([
        { expected: "Missing", reason: "生成物に宣言がありません" },
      ]);
    });
  });

  describe("異常系", () => {
    it("生成物が無ければ判定せず DegenerateOutputError", () => {
      expect(() => missingTypes([], ["DeliveryEvent"])).toThrow(DegenerateOutputError);
    });

    it("宣言が 1 つも無ければ判定せず DegenerateOutputError", () => {
      expect(() => missingTypes(["// nothing\n"], ["DeliveryEvent"])).toThrow(DegenerateOutputError);
    });
  });
});
