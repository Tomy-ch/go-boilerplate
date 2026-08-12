// pkce.ts は PKCE（RFC 7636）の S256 検証を提供する。plain は扱わない。
import { createHash } from "node:crypto";

// s256Challenge は code_verifier から code_challenge を計算する。
export function s256Challenge(codeVerifier: string): string {
  return createHash("sha256").update(codeVerifier).digest("base64url");
}

// verifyS256 は code_verifier が code_challenge に S256 で一致するか検証する。
export function verifyS256(codeVerifier: string, codeChallenge: string): boolean {
  return s256Challenge(codeVerifier) === codeChallenge;
}
