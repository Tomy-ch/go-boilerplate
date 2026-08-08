// ESLint はセキュリティ規則の実行だけを担う。型検査は tsc、整形は golines 相当の経路を
// 持たないこのワークスペースでは行わず、責務を 1 つに絞っている。typescript-eslint は
// パーサとしてのみ読み込み、その推奨規則は入れない。規則を足すのは初回導入の検出件数を
// 見てからで、最初から広げると全件が「見なかったことにする対象」になる。
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
