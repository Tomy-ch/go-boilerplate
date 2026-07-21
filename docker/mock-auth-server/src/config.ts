// config.ts は環境変数から provider の実行時設定を読み込む。
import type { OidcConfig } from "./types.ts";

// loadConfig は環境変数（既定は process.env）から設定を組み立てる。テストは env を差し替えて注入できる。
export function loadConfig(env: NodeJS.ProcessEnv = process.env): OidcConfig {
  return {
    port: Number(env.OIDC_PORT ?? "4000"),
    issuer: env.OIDC_ISSUER ?? "http://localhost:4000",
    audience: env.OIDC_AUDIENCE ?? "go-boilerplate-api",
    clientId: env.OIDC_CLIENT_ID ?? "go-boilerplate-client",
    devEndpointsEnabled: (env.MOCK_AUTH_DEV_ENDPOINTS ?? "enabled") !== "disabled",
  };
}

// config は起動時に確定する既定の設定。
export const config = loadConfig();
