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

// Client は登録済み OAuth クライアント（fixtures/clients.json）の 1 エントリ。
export interface Client {
  client_id: string;
  client_type: string;
  token_endpoint_auth_method: string;
  require_pkce: boolean;
  grant_types: string[];
  response_types: string[];
  redirect_uris: string[];
  post_logout_redirect_uris: string[];
  scopes: string[];
}

// OidcConfig は provider の実行時設定。
export interface OidcConfig {
  port: number;
  issuer: string;
  audience: string;
  clientId: string;
  // devEndpointsEnabled は /bypass/* ・ /admin/* を有効にするか（無効時は 404）。
  devEndpointsEnabled: boolean;
}

// Claims は発行する JWT のクレーム集合（標準 + IdP 拡張は index signature 経由）。
export type Claims = JWTPayload;
