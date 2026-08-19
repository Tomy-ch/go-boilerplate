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
    // プリセットのうち 2 規則をこのワークスペースでだけ落とす。判断の材料は計測した baseline で、
    // code scanning 上の scripts の指摘 140 件のうち 123 件がこの 2 つだった
    // （detect-non-literal-fs-filename 91・detect-object-injection 32）。
    //
    // どちらも「呼び出し側から来た値が危ないか」ではなく「実引数がリテラルか」だけを見る。この
    // ワークスペースはリポジトリ自身を読んで検査する道具で、パスもキーも走査の結果として組み立て
    // るため、書き方を変えない限り必ず当たり、当たった 1 件ずつに取れる行動が無い。信用境界の外
    // から来る入力はここには無く、あるのはコミットされたファイルとその内容だけである。
    //
    // 落とすのはこの 2 つだけで、正規表現側の 2 規則（detect-unsafe-regex /
    // detect-non-literal-regexp）は残す。あちらは実引数の形ではなくパターンの危険性を見ており、
    // 実際に後戻りの効く指摘を出している。docs-viewer では 2 規則とも当たらないため、あちらの
    // 設定は触らない。
    rules: {
      "security/detect-non-literal-fs-filename": "off",
      "security/detect-object-injection": "off",
    },
  },
  {
    files: ["**/*.ts", "**/*.mts", "**/*.cts"],
    languageOptions: { parser: tseslint.parser },
  },
];
