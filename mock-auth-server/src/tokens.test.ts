// tokens.test.ts は、Go 側の検証失敗系テストが依拠する固定 Profile が、それぞれ意図した壊し方の
// Token を発行することを検証する。鍵 kid に関わる profile（valid / unknown-kid / old-key）は
// keys.test.ts が、失効 Token の拒否は oidc.test.ts が受け持つ。
import { describe, expect, it } from "vitest";
import { decodeJwt, decodeProtectedHeader } from "jose";
import { loadConfig } from "./config.ts";
import { keyStore } from "./keys.ts";
import { issueAccessToken, issueToken, subjectFromToken, verifyAccessToken } from "./tokens.ts";

const config = loadConfig();
const SUBJECT = "user-john-doe";

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

it("subjectFromToken: 署名を検証せず sub を取り出す", async () => {
  const token = await issueToken(config, SUBJECT, "valid");
  expect(subjectFromToken(token)).toBe(SUBJECT);
});

it("subjectFromToken: JWT として解釈できない文字列は undefined を返す", () => {
  expect(subjectFromToken("not-a-jwt")).toBe(undefined);
});

it("subjectFromToken: sub を持たない Token は undefined を返す", async () => {
  const token = await issueToken(config, SUBJECT, "missing-subject");
  expect(subjectFromToken(token)).toBe(undefined);
});

it("issueAccessToken: 空文字の scope は既定の scope を上書きしない", async () => {
  const withEmpty = decodeJwt(await issueAccessToken(config, SUBJECT, ""));
  const withoutArgument = decodeJwt(await issueAccessToken(config, SUBJECT));

  expect(withEmpty.scope).toBe(withoutArgument.scope);
});

it("issueAccessToken: scope を渡すと claim に載せる", async () => {
  const claims = decodeJwt(await issueAccessToken(config, SUBJECT, "openid profile"));

  expect(claims.scope).toBe("openid profile");
});
