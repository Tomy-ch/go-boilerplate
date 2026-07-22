// keys.ts は署名鍵ストアを担う。複数の固定 RSA 鍵（kid 付き）を保持し、
// 「JWKS で公開する集合」と「現署名鍵 1 つ」を分離管理する（鍵ローテーションの再現）。
// 鍵は keys/*.pem に固定配置され、再起動しても不変（= 発行 Token の再現性）。状態のみ揮発する。
import { readFileSync } from "node:fs";
import { createPublicKey, type KeyObject } from "node:crypto";
import { fileURLToPath } from "node:url";
import { importPKCS8, exportJWK, type JWK } from "jose";

// ALG は署名アルゴリズム。RS256（非対称）のみを一級で扱う。
export const ALG = "RS256";

// PRIMARY_KID は初期署名鍵の kid（Phase1 の唯一の公開鍵）。既存の発行 Token 再現性のため不変。
export const PRIMARY_KID = "mock-key-1";
// ROTATION_KID は rotation で追加する 2 本目の鍵の kid（Phase2 以降の署名鍵）。
export const ROTATION_KID = "mock-key-2";
// RETIRED_KID は JWKS に一度も載らない退役鍵の kid（old-key profile 用）。回転プールには含めない。
export const RETIRED_KID = "mock-key-retired";

// SigningKey は jose の SignJWT.sign が受理する鍵型（CryptoKey / KeyObject 等）。
type SigningKey = Awaited<ReturnType<typeof importPKCS8>>;

// KeyMaterial は 1 つの鍵の署名・公開・JWKS 表現をまとめた不変の鍵素材。
interface KeyMaterial {
  kid: string;
  signingKey: SigningKey;
  publicKey: KeyObject;
  publicJwk: JWK;
}

// loadKey は keys/<kid>.pem を読み込み、署名鍵・公開鍵・公開 JWK を組み立てる。
async function loadKey(kid: string): Promise<KeyMaterial> {
  const path = fileURLToPath(new URL(`../keys/${kid}.pem`, import.meta.url));
  const pem = readFileSync(path, "utf8");
  const signingKey = await importPKCS8(pem, ALG);
  const publicKey = createPublicKey(pem);
  const publicJwk: JWK = { ...(await exportJWK(publicKey)), kid, use: "sig", alg: ALG };
  return { kid, signingKey, publicKey, publicJwk };
}

// materials は固定配置された全鍵の素材（kid→KeyMaterial）。回転プール + 退役鍵を含む。
const materials = new Map<string, KeyMaterial>(
  (await Promise.all([PRIMARY_KID, ROTATION_KID, RETIRED_KID].map(loadKey))).map((m) => [m.kid, m]),
);

// KeyStore は公開集合と署名鍵の可変状態を管理する。鍵素材そのものは不変で、状態（どれを公開 / 署名するか）だけが動く。
// 状態遷移は宣言的操作（addKey / promote / retire）で表し、テストが Phase を組み立てる。
class KeyStore {
  // publishedKids は JWKS で公開する kid の順序集合。
  private publishedKids: string[] = [PRIMARY_KID];
  // signingKid は現在の署名鍵 kid（publishedKids に含まれる 1 つ）。
  private signingKid: string = PRIMARY_KID;

  // material は kid の鍵素材を返す（未登録は例外）。profile 用の任意 kid 署名にも使う。
  material(kid: string): KeyMaterial {
    const m = materials.get(kid);
    if (m === undefined) {
      throw new Error(`unknown kid: ${kid}`);
    }
    return m;
  }

  // signing は現署名鍵の素材を返す。
  signing(): KeyMaterial {
    return this.material(this.signingKid);
  }

  // jwks は公開集合から JWK Set を組み立てる（/.well-known/jwks.json の応答）。
  jwks(): { keys: JWK[] } {
    return { keys: this.publishedKids.map((kid) => this.material(kid).publicJwk) };
  }

  // state は現在の鍵状態（署名 kid / 公開 kid 集合）を返す（rotate 応答）。
  state(): { signing_kid: string; published_kids: string[] } {
    return { signing_kid: this.signingKid, published_kids: [...this.publishedKids] };
  }

  // addKey は kid を公開集合へ追加する（署名鍵は不変）。回転プール外・重複公開は拒否する。
  addKey(kid: string): void {
    if (kid === RETIRED_KID || !materials.has(kid)) {
      throw new RotationError(`kid is not rotatable: ${kid}`);
    }
    if (this.publishedKids.includes(kid)) {
      throw new RotationError(`kid is already published: ${kid}`);
    }
    this.publishedKids.push(kid);
  }

  // promote は署名鍵を kid へ切り替える。公開集合に含まれることが前提。
  promote(kid: string): void {
    if (!this.publishedKids.includes(kid)) {
      throw new RotationError(`kid is not published, cannot promote: ${kid}`);
    }
    this.signingKid = kid;
  }

  // retire は kid を公開集合から退役させる。現署名鍵の退役は拒否する（署名不能を防ぐ）。
  retire(kid: string): void {
    if (kid === this.signingKid) {
      throw new RotationError(`cannot retire the current signing kid: ${kid}`);
    }
    if (!this.publishedKids.includes(kid)) {
      throw new RotationError(`kid is not published, cannot retire: ${kid}`);
    }
    this.publishedKids = this.publishedKids.filter((k) => k !== kid);
  }

  // reset は Phase1（公開 = 署名 = PRIMARY_KID のみ）へ戻す（admin reset 用）。
  reset(): void {
    this.publishedKids = [PRIMARY_KID];
    this.signingKid = PRIMARY_KID;
  }
}

// RotationError は不正な rotation 操作を表す（呼び出し側で 400 に写像する）。
export class RotationError extends Error {}

// keyStore は起動時に確定する署名鍵ストア（Phase1 初期状態）。
export const keyStore = new KeyStore();
