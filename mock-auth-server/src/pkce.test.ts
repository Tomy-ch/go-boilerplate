// pkce.test.ts は PKCE S256 の challenge 計算と検証を確認する。
import { describe, expect, it } from "vitest";
import { s256Challenge, verifyS256 } from "./pkce.ts";

describe("s256Challenge", () => {
  describe("正常系", () => {
    it("base64url で + / = を含まない", () => {
      expect(/[+/=]/.test(s256Challenge("some-verifier"))).toBe(false);
    });

    it("同じ code_verifier からは同じ challenge を返す", () => {
      expect(s256Challenge("same-verifier")).toBe(s256Challenge("same-verifier"));
    });

    it("異なる code_verifier からは異なる challenge を返す", () => {
      expect(s256Challenge("verifier-a")).not.toBe(s256Challenge("verifier-b"));
    });
  });
});

describe("verifyS256", () => {
  describe("正常系", () => {
    it("一致する code_verifier を受理する", () => {
      const verifier = "test-code-verifier-abcdefghijklmnopqrstuvwxyz-0123456789";
      expect(verifyS256(verifier, s256Challenge(verifier))).toBe(true);
    });
  });

  describe("異常系", () => {
    it("不一致の code_verifier を拒否する", () => {
      expect(verifyS256("wrong-verifier", s256Challenge("right-verifier"))).toBe(false);
    });

    it("code_verifier をそのまま code_challenge として渡す形は拒否する", () => {
      expect(verifyS256("plain-verifier", "plain-verifier")).toBe(false);
    });
  });
});
