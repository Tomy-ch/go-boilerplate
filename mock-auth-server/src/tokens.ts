// tokens.ts は、固定 Profile 方式で access token / id token を発行する。
// 任意 Claim 注入 API にはせず、再現性のため異常系を固定 Profile として提供する。
// 署名鍵は keyStore（現署名鍵）から都度取得するため、鍵ローテーション後は新署名鍵で発行される。
import { SignJWT, jwtVerify, decodeJwt, createLocalJWKSet } from "jose";
import { generateKeyPairSync, randomUUID } from "node:crypto";
import { keyStore, ALG, RETIRED_KID } from "./keys.ts";
import type { Claims, OidcConfig } from "./types.ts";

// SignKey は jose の SignJWT.sign が受理する鍵型（CryptoKey / KeyObject / Uint8Array 等）を導出する。
type SignKey = Parameters<SignJWT["sign"]>[0];

// ACCESS_TTL_SECONDS は access token の有効期間。1 本のトークンで通す一連の操作
// （DAST スキャン、手動の curl）の途中で失効させないため、長めに採る。
const ACCESS_TTL_SECONDS = 3600;
// ACCESS_SCOPE は Client に許された API 用途を表す標準 scope（Role とは別軸）。
const ACCESS_SCOPE = "openid profile email api.read api.write";
// ACCESS_TOKEN_TYPE は access token の typ ヘッダ（RFC 9068）。Go 側はこの typ で ID Token 誤用を拒否する。
const ACCESS_TOKEN_TYPE = "at+jwt";
// UNKNOWN_KID は unknown-kid profile が付す、JWKS に存在しない kid。Go 側は鍵解決に失敗し拒否する。
const UNKNOWN_KID = "unknown-kid";

// wrongSigningKey は invalid-signature プロファイル用の、JWKS に載らない別 RSA 鍵。
const { privateKey: wrongSigningKey } = generateKeyPairSync("rsa", { modulusLength: 2048 });
// symmetricSecret は unsupported-algorithm（HS256）プロファイル用の対称鍵。
const symmetricSecret = new TextEncoder().encode("mock-auth-server-unsupported-alg-secret");

// PROFILES は bypass/token がサポートする Token Profile の一覧。
export const PROFILES: readonly string[] = [
  "valid",
  "expired",
  "not-yet-valid",
  "wrong-issuer",
  "wrong-audience",
  "missing-subject",
  "invalid-signature",
  "unsupported-algorithm",
  "unknown-kid",
  "old-key",
  "id-token",
];

// baseAccessClaims は正常な access token の標準クレームを組み立てる。
function baseAccessClaims(config: OidcConfig, subject: string, now: number): Claims {
  return {
    iss: config.issuer,
    sub: subject,
    aud: config.audience,
    iat: now,
    nbf: now,
    exp: now + ACCESS_TTL_SECONDS,
    jti: randomUUID(),
    client_id: config.clientId,
    token_use: "access",
    scope: ACCESS_SCOPE,
  };
}

// idTokenClaims は ID Token のクレーム（aud=認可を要求した client_id / token_use=id）を組み立てる。
function idTokenClaims(config: OidcConfig, subject: string, now: number, clientId: string = config.clientId): Claims {
  return {
    iss: config.issuer,
    sub: subject,
    aud: clientId,
    iat: now,
    exp: now + ACCESS_TTL_SECONDS,
    token_use: "id",
    email: `${subject}@example.com`,
    name: subject,
  };
}

// sign は指定の署名鍵・alg・typ・kid でクレームに署名する。
function sign(claims: Claims, key: SignKey, alg: string, typ: string, kid: string): Promise<string> {
  return new SignJWT(claims).setProtectedHeader({ alg, kid, typ }).sign(key);
}

// signWithCurrent は現署名鍵で署名する（正常系の共通経路）。
function signWithCurrent(claims: Claims, typ: string): Promise<string> {
  const { signingKey, kid } = keyStore.signing();
  return sign(claims, signingKey, ALG, typ, kid);
}

// issueToken は subject と profile から Token 文字列を発行する。未知の profile は valid 扱い。
export function issueToken(config: OidcConfig, subject: string, profile: string): Promise<string> {
  const now = Math.floor(Date.now() / 1000);
  const claims = baseAccessClaims(config, subject, now);
  const currentKid = keyStore.signing().kid;

  switch (profile) {
    case "expired":
      claims.iat = now - 2 * ACCESS_TTL_SECONDS;
      claims.nbf = now - 2 * ACCESS_TTL_SECONDS;
      claims.exp = now - ACCESS_TTL_SECONDS;
      return signWithCurrent(claims, ACCESS_TOKEN_TYPE);
    case "not-yet-valid":
      claims.nbf = now + ACCESS_TTL_SECONDS;
      return signWithCurrent(claims, ACCESS_TOKEN_TYPE);
    case "wrong-issuer":
      claims.iss = "https://evil.example.com";
      return signWithCurrent(claims, ACCESS_TOKEN_TYPE);
    case "wrong-audience":
      claims.aud = "wrong-audience";
      return signWithCurrent(claims, ACCESS_TOKEN_TYPE);
    case "missing-subject":
      delete claims.sub;
      return signWithCurrent(claims, ACCESS_TOKEN_TYPE);
    case "invalid-signature":
      // JWKS に載らない別鍵で署名（kid は正規のまま）。Go 側は署名検証で拒否する。
      return sign(claims, wrongSigningKey, ALG, ACCESS_TOKEN_TYPE, currentKid);
    case "unsupported-algorithm":
      // 許可外の対称鍵アルゴリズム（HS256）。Go 側は alg allowlist で拒否する。
      return sign(claims, symmetricSecret, "HS256", ACCESS_TOKEN_TYPE, currentKid);
    case "unknown-kid":
      // JWKS に存在しない kid で署名（署名自体は現鍵で有効）。Go 側は鍵解決に失敗し拒否する。
      return sign(claims, keyStore.signing().signingKey, ALG, ACCESS_TOKEN_TYPE, UNKNOWN_KID);
    case "old-key":
      // JWKS に一度も載らない退役鍵で署名。Go 側は kid を解決できず拒否する（退役鍵拒否の再現）。
      return sign(claims, keyStore.material(RETIRED_KID).signingKey, ALG, ACCESS_TOKEN_TYPE, RETIRED_KID);
    case "id-token":
      // token_use=id / aud=client_id / typ!=at+jwt。Go 側は typ と aud の両方で拒否する。
      return signWithCurrent(idTokenClaims(config, subject, now), "JWT");
    case "valid":
    default:
      return signWithCurrent(claims, ACCESS_TOKEN_TYPE);
  }
}

// issueAccessToken は Authorization Code Flow の access token を発行する（typ=at+jwt）。
// scope を指定すると付与済み scope として反映する。
export function issueAccessToken(config: OidcConfig, subject: string, scope?: string): Promise<string> {
  const now = Math.floor(Date.now() / 1000);
  const claims = baseAccessClaims(config, subject, now);
  if (scope !== undefined && scope !== "") {
    claims.scope = scope;
  }
  return signWithCurrent(claims, ACCESS_TOKEN_TYPE);
}

// issueIdToken は OIDC の ID Token を発行する（typ=JWT）。aud には認可を要求した clientId を用い、
// nonce を指定すると反映する。clientId 省略時は config.clientId。
export function issueIdToken(
  config: OidcConfig,
  subject: string,
  nonce?: string,
  clientId: string = config.clientId,
): Promise<string> {
  const now = Math.floor(Date.now() / 1000);
  const claims = idTokenClaims(config, subject, now, clientId);
  if (nonce !== undefined && nonce !== "") {
    claims.nonce = nonce;
  }
  return signWithCurrent(claims, "JWT");
}

// verifyAccessToken は access token を検証する。typ=at+jwt でないもの（ID Token 等）は拒否し、
// 署名（現在公開中の JWKS の kid で解決）・iss・aud・alg・有効期限を検証する。
// 成功時 payload を返し、失敗時は例外を投げる。
export async function verifyAccessToken(config: OidcConfig, token: string): Promise<Claims> {
  const jwks = createLocalJWKSet(keyStore.jwks());
  const { payload } = await jwtVerify(token, jwks, {
    issuer: config.issuer,
    audience: config.audience,
    algorithms: [ALG],
    typ: ACCESS_TOKEN_TYPE,
    requiredClaims: ["sub"],
  });
  return payload;
}

// subjectFromToken は token を検証せず sub を取り出す（logout の id_token_hint からの subject 抽出用）。
export function subjectFromToken(token: string): string | undefined {
  try {
    const sub = decodeJwt(token).sub;
    return typeof sub === "string" ? sub : undefined;
  } catch {
    return undefined;
  }
}

export { ACCESS_TTL_SECONDS };
