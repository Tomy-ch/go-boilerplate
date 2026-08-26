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
    // 正規表現側の 2 規則も同じ理由で落とす。detect-unsafe-regex が使う safe-regex は、後戻り
    // そのものではなく量指定子の入れ子の深さを数えるため、`(?:…+)?` の形をした任意の省略可能な
    // 組に当たる。実際に後戻りする書き方は Sonar の S8786 が指しており、そちらが挙げた 10 件は
    // 直した上で、残った 7 件はいずれも入れ子の深さだけを根拠にしている。detect-non-literal-regexp
    // は `new RegExp(変数)` の形を見るもので、ここで渡るのはアクションのパス・マーカー名・スキル
    // 名といったリポジトリが決めた文字列だけである（組み立て側はメタ文字を退避してもいる）。
    //
    // docs-viewer は 4 規則とも当たらないため、あちらの設定は触らない。プリセットの残りは
    // ここでも動いたままで、外したのはこの 4 つだけである。
    rules: {
      "security/detect-non-literal-fs-filename": "off",
      "security/detect-object-injection": "off",
      "security/detect-unsafe-regex": "off",
      "security/detect-non-literal-regexp": "off",
    },
  },
  {
    files: ["**/*.ts", "**/*.mts", "**/*.cts"],
    languageOptions: { parser: tseslint.parser },
  },
];
