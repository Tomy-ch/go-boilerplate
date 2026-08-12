// ESLint はセキュリティ規則の実行だけを担う。typescript-eslint はパーサとしてのみ読み込み、
// その推奨規則は入れない（規則を足すかの判断基準は .github/workflows/eslint.yaml が持つ）。
import security from "eslint-plugin-security";
import tseslint from "typescript-eslint";

export default [
  {
    // 依存ツリーとカバレッジ出力。編集できないものの指摘は行動に繋がらない。
    // vite の成果物はここに載らない（outDir が ../docs/portal で、走査範囲の外）。
    ignores: ["node_modules/**", "coverage/**"],
  },
  security.configs.recommended,
  {
    files: ["**/*.ts", "**/*.tsx"],
    languageOptions: { parser: tseslint.parser },
  },
];
