export type JWK = Record<string, unknown>;
export type JWTPayload = Record<string, unknown>;
export const importPKCS8: (...args: unknown[]) => Promise<unknown>;
export const exportJWK: (...args: unknown[]) => Promise<JWK>;
export const jwtVerify: (...args: unknown[]) => Promise<{ payload: JWTPayload }>;
export const decodeJwt: (...args: unknown[]) => JWTPayload;
export const createLocalJWKSet: (...args: unknown[]) => unknown;
export class SignJWT {
  constructor(...args: unknown[]);
  setProtectedHeader(...args: unknown[]): this;
  setIssuedAt(...args: unknown[]): this;
  setExpirationTime(...args: unknown[]): this;
  sign(key: unknown): Promise<string>;
}
