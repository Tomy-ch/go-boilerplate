// tokens.test.ts は、Go 側の検証失敗系テストが依拠する固定 Profile が、それぞれ意図した壊し方の
// Token を発行することを検証する。鍵 kid に関わる profile（valid / unknown-kid / old-key）は
// keys.test.ts が、失効 Token の拒否は oidc.test.ts が受け持つ。
import { describe, expect, it } from "vitest";
import { decodeJwt, decodeProtectedHeader } from "jose";
import { loadConfig } from "./config.ts";
import { keyStore } from "./keys.ts";
import {
  issueAccessToken,
  issueIdToken,
  issueToken,
  subjectFromToken,
  verifyAccessToken,
} from "./tokens.ts";

const config = loadConfig();
const SUBJECT = "user-john-doe";

describe("issueToken", () => {
  describe("正常系", () => {
    it("profile valid は現署名鍵の kid と at+jwt で発行する", async () => {
      const token = await issueToken(config, SUBJECT, "valid");
      expect(decodeProtectedHeader(token).kid).toBe(keyStore.signing().kid);
      expect(decodeProtectedHeader(token).typ).toBe("at+jwt");
    });

    it("未知の profile は valid と同じ扱いで発行する", async () => {
      const claims = decodeJwt(await issueToken(config, SUBJECT, "no-such-profile"));
      expect(claims.iss).toBe(config.issuer);
      expect(claims.aud).toBe(config.audience);
      expect(claims.sub).toBe(SUBJECT);
    });
  });

  describe("異常系", () => {
    it("profile expired: exp を過去にして失効済みの Token を発行する", async () => {
      const claims = decodeJwt(await issueToken(config, SUBJECT, "expired"));
      expect(claims.exp ?? 0).toBeLessThan(Math.floor(Date.now() / 1000));
    });

    it("profile not-yet-valid: nbf を未来にして未発効の Token を発行する", async () => {
      const claims = decodeJwt(await issueToken(config, SUBJECT, "not-yet-valid"));
      expect(claims.nbf).toBeDefined();
      expect(claims.nbf ?? 0).toBeGreaterThan(Math.floor(Date.now() / 1000));
    });

    it("profile wrong-issuer: iss を別 issuer に差し替える", async () => {
      const claims = decodeJwt(await issueToken(config, SUBJECT, "wrong-issuer"));
      expect(claims.iss).toBe("https://evil.example.com");
      expect(claims.iss).not.toBe(config.issuer);
    });

    it("profile wrong-audience: aud を別 audience に差し替える", async () => {
      const claims = decodeJwt(await issueToken(config, SUBJECT, "wrong-audience"));
      expect(claims.aud).toBe("wrong-audience");
      expect(claims.aud).not.toBe(config.audience);
    });

    it("profile missing-subject: sub を落とした Token を発行する", async () => {
      const claims = decodeJwt(await issueToken(config, SUBJECT, "missing-subject"));
      expect(claims.sub).toBe(undefined);
    });

    it("profile invalid-signature: kid は正規のまま JWKS 外の鍵で署名する", async () => {
      const token = await issueToken(config, SUBJECT, "invalid-signature");
      expect(decodeProtectedHeader(token).kid).toBe(keyStore.signing().kid);
      await expect(verifyAccessToken(config, token)).rejects.toThrow();
    });

    it("profile unsupported-algorithm: 許可外の対称鍵アルゴリズムで署名する", async () => {
      const token = await issueToken(config, SUBJECT, "unsupported-algorithm");
      expect(decodeProtectedHeader(token).alg).toBe("HS256");
      await expect(verifyAccessToken(config, token)).rejects.toThrow();
    });

    it("profile id-token: access token の位置に ID Token を発行する", async () => {
      const token = await issueToken(config, SUBJECT, "id-token");
      expect(decodeProtectedHeader(token).typ).toBe("JWT");
      const claims = decodeJwt(token);
      expect(claims.token_use).toBe("id");
      expect(claims.aud).toBe(config.clientId);
      await expect(verifyAccessToken(config, token)).rejects.toThrow();
    });
  });
});

describe("issueAccessToken", () => {
  describe("正常系", () => {
    it("scope を渡すと claim に載せる", async () => {
      const claims = decodeJwt(await issueAccessToken(config, SUBJECT, "openid profile"));

      expect(claims.scope).toBe("openid profile");
    });

    it("空文字の scope は既定の scope を上書きしない", async () => {
      const withEmpty = decodeJwt(await issueAccessToken(config, SUBJECT, ""));
      const withoutArgument = decodeJwt(await issueAccessToken(config, SUBJECT));

      expect(withEmpty.scope).toBe(withoutArgument.scope);
    });

    it("access token として検証を通る", async () => {
      const token = await issueAccessToken(config, SUBJECT);

      await expect(verifyAccessToken(config, token)).resolves.toMatchObject({ sub: SUBJECT });
    });
  });
});

describe("issueIdToken", () => {
  describe("正常系", () => {
    it("aud を省略すると config の client_id を採用する", async () => {
      const token = await issueIdToken(config, SUBJECT);
      const claims = decodeJwt(token);

      expect(decodeProtectedHeader(token).typ).toBe("JWT");
      expect(claims.aud).toBe(config.clientId);
      expect(claims.token_use).toBe("id");
    });

    it("認可を要求した client_id を aud に載せる", async () => {
      const claims = decodeJwt(await issueIdToken(config, SUBJECT, undefined, "other-client"));

      expect(claims.aud).toBe("other-client");
    });

    it("nonce を渡すと claim に載せる", async () => {
      const claims = decodeJwt(await issueIdToken(config, SUBJECT, "nonce-value"));

      expect(claims.nonce).toBe("nonce-value");
    });

    it("空文字の nonce は claim に載せない", async () => {
      const claims = decodeJwt(await issueIdToken(config, SUBJECT, ""));

      expect(claims.nonce).toBe(undefined);
    });
  });

  describe("異常系", () => {
    it("ID Token は access token としては受理されない", async () => {
      const token = await issueIdToken(config, SUBJECT);

      await expect(verifyAccessToken(config, token)).rejects.toThrow();
    });
  });
});

describe("verifyAccessToken", () => {
  describe("正常系", () => {
    it("現署名鍵で発行した valid Token の payload を返す", async () => {
      const payload = await verifyAccessToken(config, await issueToken(config, SUBJECT, "valid"));

      expect(payload.sub).toBe(SUBJECT);
      expect(payload.iss).toBe(config.issuer);
    });
  });

  describe("異常系", () => {
    it("sub を持たない Token を拒否する", async () => {
      const token = await issueToken(config, SUBJECT, "missing-subject");

      await expect(verifyAccessToken(config, token)).rejects.toThrow();
    });

    it("JWT として解釈できない文字列を拒否する", async () => {
      await expect(verifyAccessToken(config, "not-a-jwt")).rejects.toThrow();
    });
  });
});

describe("subjectFromToken", () => {
  describe("正常系", () => {
    it("署名を検証せず sub を取り出す", async () => {
      const token = await issueToken(config, SUBJECT, "valid");
      expect(subjectFromToken(token)).toBe(SUBJECT);
    });

    it("署名が JWKS 外の鍵でも sub を取り出す", async () => {
      const token = await issueToken(config, SUBJECT, "invalid-signature");
      expect(subjectFromToken(token)).toBe(SUBJECT);
    });
  });

  describe("異常系", () => {
    it("JWT として解釈できない文字列は undefined を返す", () => {
      expect(subjectFromToken("not-a-jwt")).toBe(undefined);
    });

    it("sub を持たない Token は undefined を返す", async () => {
      const token = await issueToken(config, SUBJECT, "missing-subject");
      expect(subjectFromToken(token)).toBe(undefined);
    });
  });
});
