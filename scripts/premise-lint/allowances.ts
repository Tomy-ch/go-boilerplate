// 前提の言い回しに当たるが前提ではない行の宣言。
//
// 行番号でなく本文の一部で綴じるのは、行が動いても壊れないためであり、また「どの記述を
// 許したのか」が宣言だけで読めるようにするためでもある。理由を書けない行はここへ書けない。

import type { Allowance } from "./rules";

export const ALLOWANCES: readonly Allowance[] = [
  {
    file: "internal/cli/fixcollation/README.md",
    contains: "while the template carries a stale collation version",
    reason:
      "PostgreSQL の template データベース（`template1`）の話であり、リポジトリの頒布形態とは無関係。",
  },
];
