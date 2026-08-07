// config.test.ts は env から provider の実行時設定を組み立てる規則を検証する。既定値は
// docker-compose / env ファイルを持たない素の起動を成立させる約束なので、キーごとに固定する。
import { describe, expect, it } from "vitest";
import { loadConfig } from "./config.ts";

describe("loadConfig", () => {
  describe("正常系", () => {
    it("env が空なら全項目を既定値で埋める", () => {
      expect(loadConfig({})).toEqual({
        port: 4000,
        issuer: "http://localhost:4000",
        audience: "go-boilerplate-api",
        clientId: "go-boilerplate-client",
        devEndpointsEnabled: true,
      });
    });

    it("指定された env を既定値より優先する", () => {
      expect(
        loadConfig({
          OIDC_PORT: "2010",
          OIDC_ISSUER: "http://mock-auth:2010",
          OIDC_AUDIENCE: "other-api",
          OIDC_CLIENT_ID: "other-client",
        }),
      ).toEqual({
        port: 2010,
        issuer: "http://mock-auth:2010",
        audience: "other-api",
        clientId: "other-client",
        devEndpointsEnabled: true,
      });
    });

    it("MOCK_AUTH_DEV_ENDPOINTS=disabled のときだけ dev endpoint を無効にする", () => {
      expect(loadConfig({ MOCK_AUTH_DEV_ENDPOINTS: "disabled" }).devEndpointsEnabled).toBe(false);
      expect(loadConfig({ MOCK_AUTH_DEV_ENDPOINTS: "enabled" }).devEndpointsEnabled).toBe(true);
    });

    it("disabled 以外の値は無効化と解釈しない", () => {
      expect(loadConfig({ MOCK_AUTH_DEV_ENDPOINTS: "false" }).devEndpointsEnabled).toBe(true);
      expect(loadConfig({ MOCK_AUTH_DEV_ENDPOINTS: "" }).devEndpointsEnabled).toBe(true);
    });
  });

  describe("異常系", () => {
    it("数値として読めない OIDC_PORT は既定値へ戻さず NaN のまま返す", () => {
      expect(loadConfig({ OIDC_PORT: "not-a-number" }).port).toBeNaN();
    });
  });
});
