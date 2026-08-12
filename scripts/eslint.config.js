// ESLint はセキュリティ規則の実行だけを担う。型検査は tsc が持ち、整形はこのワークスペースでは
// 行わない。typescript-eslint はパーサとしてのみ読み込み、その推奨規則は入れない
// （規則を足すかの判断基準は .github/workflows/eslint.yaml が持つ）。
import security from "eslint-plugin-security";
import tseslint from "typescript-eslint";

export default [
  {
    // 依存ツリーと生成物。編集できないものの指摘は行動に繋がらない。
    ignores: ["node_modules/**", "coverage/**"],
  },
  security.configs.recommended,
  {
    files: ["**/*.ts", "**/*.mts", "**/*.cts"],
    languageOptions: { parser: tseslint.parser },
  },
];
