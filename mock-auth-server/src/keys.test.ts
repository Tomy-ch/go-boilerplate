// keys.test.ts は鍵ストアの状態遷移・golden JWKS 一致・profile の kid を検証する。
import { describe, expect, it, beforeEach } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { decodeProtectedHeader } from "jose";
import { keyStore, PRIMARY_KID, ROTATION_KID, RETIRED_KID } from "./keys.ts";
import { issueToken } from "./tokens.ts";
import type { OidcConfig } from "./types.ts";

const config: OidcConfig = {
  port: 4000,
  issuer: "http://localhost:4000",
  audience: "go-boilerplate-api",
  clientId: "go-boilerplate-client",
  devEndpointsEnabled: true,
};

// goldenJwks は fixtures/jwks/phase<n>.json（provider 生成の golden）を読む。
function goldenJwks(phase: number): unknown {
  const path = fileURLToPath(new URL(`../fixtures/jwks/phase${phase}.json`, import.meta.url));
  return JSON.parse(readFileSync(path, "utf8"));
}

beforeEach(() => {
  keyStore.reset();
});

it("state: 初期状態は Phase1（公開 = 署名 = PRIMARY_KID）", () => {
  expect(keyStore.state()).toEqual({ signing_kid: PRIMARY_KID, published_kids: [PRIMARY_KID] });
});

it("golden 一致: 各 Phase の jwks() が fixtures/jwks/phase<n>.json とバイト等価", () => {
  expect(keyStore.jwks()).toEqual(goldenJwks(1));

  keyStore.addKey(ROTATION_KID);
  keyStore.promote(ROTATION_KID);
  expect(keyStore.jwks()).toEqual(goldenJwks(2));

  keyStore.retire(PRIMARY_KID);
  expect(keyStore.jwks()).toEqual(goldenJwks(3));
});

it("promote: 昇格後に valid profile が新署名鍵 kid で署名する", async () => {
  keyStore.addKey(ROTATION_KID);
  keyStore.promote(ROTATION_KID);
  const token = await issueToken(config, "user-active", "valid");
  expect(decodeProtectedHeader(token).kid).toBe(ROTATION_KID);
});

it("profile unknown-kid: JWKS に存在しない kid で署名する", async () => {
  const token = await issueToken(config, "user-active", "unknown-kid");
  const kid = decodeProtectedHeader(token).kid;
  expect(kid).toBe("unknown-kid");
  const published = keyStore.jwks().keys.map((k) => k.kid);
  expect(!published.includes(kid)).toBeTruthy();
});

it("profile old-key: 退役鍵 kid で署名し公開集合に載らない", async () => {
  const token = await issueToken(config, "user-active", "old-key");
  const kid = decodeProtectedHeader(token).kid;
  expect(kid).toBe(RETIRED_KID);
  const published = keyStore.jwks().keys.map((k) => k.kid);
  expect(!published.includes(kid)).toBeTruthy();
});

it("material: 未登録の kid は例外を投げる", () => {
  expect(() => keyStore.material("no-such-kid")).toThrow(/unknown kid: no-such-kid/);
});
