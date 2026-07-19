// keys.ts は、固定 RSA 秘密鍵の読み込みと JWKS（公開鍵）の組み立てを担う。
// 鍵は keys/mock-key-1.pem に固定配置され、再起動しても不変（= 発行 Token の再現性）。
import { readFileSync } from "node:fs";
import { createPublicKey } from "node:crypto";
import { fileURLToPath } from "node:url";
import { importPKCS8, exportJWK } from "jose";

// KID は JWKS と Token ヘッダで一致させる固定の鍵 ID。
export const KID = "mock-key-1";
// ALG は署名アルゴリズム。RS256（非対称）のみを一級で扱う。
export const ALG = "RS256";

const privateKeyPath = fileURLToPath(new URL("../keys/mock-key-1.pem", import.meta.url));
const privateKeyPem = readFileSync(privateKeyPath, "utf8");

// signingKey は access token / id token の署名に用いる秘密鍵。
export const signingKey = await importPKCS8(privateKeyPem, ALG);

// publicJwk は JWKS で公開する公開鍵（秘密情報を含まない）。
const publicJwk = {
  ...(await exportJWK(createPublicKey(privateKeyPem))),
  kid: KID,
  use: "sig",
  alg: ALG,
};

// jwks は /.well-known/jwks.json で返す JWK Set。
export const jwks = { keys: [publicJwk] };
