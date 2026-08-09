import { defineConfig } from "vitest/config";

import { EXCLUDED_FROM_CHECKS } from "./lib/untested-modules.ts";

// root を明示するのは、既定値が起動時のカレントディレクトリだから。ここを固定しておくと、
// どこから起動しても走査範囲は scripts/ の外へ広がらない。
export default defineConfig({
  root: import.meta.dirname,
  test: {
    environment: "node",
    include: ["**/*.test.ts"],
    coverage: {
      provider: "v8",
      // 母数は判定ロジックを持つモジュールに限る。入口を母数へ入れると、守るべき面の率が
      // 0% の入口に薄められて読めなくなる。外す対象と理由は lib/untested-modules.ts が持つ
      // （1:1 ゲートと同じ宣言を読む。2 箇所に書くと片方だけ直したときに黙ってずれる）。
      // 全 .ts を母数に取り、外すのは除外宣言だけにする。ディレクトリを列挙する形だと、
      // 新しいツールのディレクトリを足したとき黙って母数から漏れる。
      include: ["**/*.ts"],
      exclude: [...EXCLUDED_FROM_CHECKS],
      // text だけに絞る。HTML を出すとディスクへ書き出したものを .gitignore で面倒見る話に
      // なるが、Go 側の docs/coverage/ と違って公開する consumer がまだ無い。
      reporter: ["text", "lcov"],
      // 母数を判定モジュールへ絞ってあるぶん、100% は「網羅せよ」ではなく「検査されない分岐を
      // 残さない」を意味する。ここが下がるのは新しい判定を足して踏まないまま置いた場合で、
      // それはゲートが黙る方向の変更そのものなので、率ではなく不変条件として止める。
      thresholds: { statements: 100, branches: 100, functions: 100, lines: 100 },
    },
  },
});
