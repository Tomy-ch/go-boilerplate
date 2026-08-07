// pkce.test.ts は PKCE S256 の challenge 計算と検証を確認する。
import { describe, expect, it } from "vitest";
import { s256Challenge, verifyS256 } from "./pkce.ts";

it("verifyS256: 一致する code_verifier を受理する", () => {
  const verifier = "test-code-verifier-abcdefghijklmnopqrstuvwxyz-0123456789";
  expect(verifyS256(verifier, s256Challenge(verifier))).toBe(true);
});

it("verifyS256: 不一致の code_verifier を拒否する", () => {
  expect(verifyS256("wrong-verifier", s256Challenge("right-verifier"))).toBe(false);
});

it("s256Challenge: base64url で + / = を含まない", () => {
  expect(/[+/=]/.test(s256Challenge("some-verifier"))).toBe(false);
});
