// types.ts は mock-auth-server 内で共有する型定義。
import type { JWTPayload } from "jose";

// User は固定 User Fixture（fixtures/users.json）の 1 エントリ。
export interface User {
  subject: string;
  email: string;
  given_name: string;
  family_name: string;
  name: string;
  status: string;
}

// OidcConfig は provider の実行時設定。
export interface OidcConfig {
  port: number;
  issuer: string;
  audience: string;
  clientId: string;
  testEndpointsEnabled: boolean;
}

// Claims は発行する JWT のクレーム集合（標準 + IdP 拡張は index signature 経由）。
export type Claims = JWTPayload;
