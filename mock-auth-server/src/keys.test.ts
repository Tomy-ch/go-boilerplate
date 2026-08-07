// keys.test.ts は鍵ストアの状態遷移・golden JWKS 一致・profile の kid を検証する。
import { describe, expect, it, beforeEach } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { decodeProtectedHeader } from "jose";
import { keyStore, PRIMARY_KID, ROTATION_KID, RETIRED_KID, RotationError } from "./keys.ts";
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

describe("keyStore", () => {
  describe("正常系", () => {
    it("初期状態は Phase1（公開 = 署名 = PRIMARY_KID）", () => {
      expect(keyStore.state()).toEqual({ signing_kid: PRIMARY_KID, published_kids: [PRIMARY_KID] });
    });

    it("各 Phase の jwks() が fixtures/jwks/phase<n>.json とバイト等価", () => {
      expect(keyStore.jwks()).toEqual(goldenJwks(1));

      keyStore.addKey(ROTATION_KID);
      keyStore.promote(ROTATION_KID);
      expect(keyStore.jwks()).toEqual(goldenJwks(2));

      keyStore.retire(PRIMARY_KID);
      expect(keyStore.jwks()).toEqual(goldenJwks(3));
    });

    it("昇格後は valid profile が新署名鍵 kid で署名する", async () => {
      keyStore.addKey(ROTATION_KID);
      keyStore.promote(ROTATION_KID);
      const token = await issueToken(config, "user-active", "valid");
      expect(decodeProtectedHeader(token).kid).toBe(ROTATION_KID);
    });
  });

  describe("異常系", () => {
    it("profile unknown-kid は JWKS に存在しない kid で署名する", async () => {
      const token = await issueToken(config, "user-active", "unknown-kid");
      const kid = decodeProtectedHeader(token).kid;
      expect(kid).toBe("unknown-kid");
      const published = keyStore.jwks().keys.map((k) => k.kid);
      expect(!published.includes(kid)).toBeTruthy();
    });

    it("profile old-key は退役鍵 kid で署名し公開集合に載らない", async () => {
      const token = await issueToken(config, "user-active", "old-key");
      const kid = decodeProtectedHeader(token).kid;
      expect(kid).toBe(RETIRED_KID);
      const published = keyStore.jwks().keys.map((k) => k.kid);
      expect(!published.includes(kid)).toBeTruthy();
    });

    it("未登録の kid は例外を投げる", () => {
      expect(() => keyStore.material("no-such-kid")).toThrow(/unknown kid: no-such-kid/);
    });
  });
});

describe("RotationError", () => {
  describe("正常系", () => {
    it("Error を継承し message をそのまま保持する", () => {
      const err = new RotationError("kid is not rotatable: x");

      expect(err).toBeInstanceOf(Error);
      expect(err.message).toBe("kid is not rotatable: x");
    });
  });

  describe("異常系", () => {
    it("回転プール外の kid の追加は RotationError で拒否する", () => {
      expect(() => keyStore.addKey(RETIRED_KID)).toThrow(RotationError);
    });

    it("公開済み kid の重複追加は RotationError で拒否する", () => {
      expect(() => keyStore.addKey(PRIMARY_KID)).toThrow(RotationError);
    });

    it("未公開 kid の昇格は RotationError で拒否する", () => {
      expect(() => keyStore.promote(ROTATION_KID)).toThrow(RotationError);
    });

    it("現署名鍵の退役は RotationError で拒否する", () => {
      expect(() => keyStore.retire(PRIMARY_KID)).toThrow(RotationError);
    });

    it("未公開 kid の退役は RotationError で拒否する", () => {
      expect(() => keyStore.retire(ROTATION_KID)).toThrow(RotationError);
    });

    it("鍵素材の解決失敗は RotationError にしない（400 へ写像させない）", () => {
      let caught: unknown;

      try {
        keyStore.material("no-such-kid");
      } catch (err) {
        caught = err;
      }

      expect(caught).toBeInstanceOf(Error);
      expect(caught).not.toBeInstanceOf(RotationError);
    });
  });
});
