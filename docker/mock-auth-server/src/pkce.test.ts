// pkce.test.ts は PKCE S256 の challenge 計算と検証を確認する。
import { test } from "node:test";
import assert from "node:assert/strict";
import { s256Challenge, verifyS256 } from "./pkce.ts";

test("verifyS256: 一致する code_verifier を受理する", () => {
  const verifier = "test-code-verifier-abcdefghijklmnopqrstuvwxyz-0123456789";
  assert.equal(verifyS256(verifier, s256Challenge(verifier)), true);
});

test("verifyS256: 不一致の code_verifier を拒否する", () => {
  assert.equal(verifyS256("wrong-verifier", s256Challenge("right-verifier")), false);
});

test("s256Challenge: base64url で + / = を含まない", () => {
  assert.equal(/[+/=]/.test(s256Challenge("some-verifier")), false);
});
