// orval.config.ts は、バンドル済み OpenAPI（openapi/openapi.gen.yaml）から zod スキーマを生成する設定。
// 'orval' からの値 import（defineConfig 等）は、この設定ファイルの位置から 'orval' が解決できることを要求する。
// orval は node_tools イメージ側に同梱され provider の node_modules には無いため、その要求を持たないプレーンなオブジェクトとして export する。
export default {
  mockAuth: {
    input: {
      target: "./openapi/openapi.gen.yaml",
    },
    output: {
      mode: "single",
      client: "zod",
      target: "./src/generated/schemas.ts",
      fileExtension: ".ts",
    },
  },
};
