// ESLint はセキュリティ規則の実行だけを担う。typescript-eslint はパーサとしてのみ読み込み、
// その推奨規則は入れない（規則を足すかの判断基準は .github/workflows/eslint.yaml が持つ）。
import security from "eslint-plugin-security";
import tseslint from "typescript-eslint";

export default [
  {
    // 依存ツリーと、orval が OpenAPI から起こす zod スキーマ。編集できないものの指摘は
    // 行動に繋がらない。
    ignores: ["node_modules/**", "coverage/**", "src/generated/**"],
  },
  security.configs.recommended,
  {
    files: ["**/*.ts"],
    languageOptions: { parser: tseslint.parser },
  },
];
