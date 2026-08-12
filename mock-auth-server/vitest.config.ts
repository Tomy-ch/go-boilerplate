import { defineConfig } from "vitest/config";

// root を明示するのは、既定値が起動時のカレントディレクトリだから。ここを固定しておくと、
// どこから起動しても走査範囲は mock-auth-server/ の外へ広がらない。
export default defineConfig({
  root: import.meta.dirname,
  test: {
    environment: "node",
    // 統合テスト（integration/）は実サーバーを起動する別系統で、単体の走査には混ぜない。
    include: ["src/**/*.test.ts"],
    coverage: {
      provider: "v8",
      include: ["src/**/*.ts"],
      exclude: [
        "src/**/*.test.ts",
        // orval の生成物。手で直さないため検査対象にしない。
        "src/generated/**",
        // 起動の入口。読み込んだ時点で listen するため、import しただけで副作用が出る。
        // 判断はすべて router.ts 側にあり、そちらは検査対象に残している。
        "src/server.ts",
      ],
      // text だけに絞る。HTML を出すとディスクへ書き出したものを .gitignore で面倒見る話に
      // なるが、公開する consumer がまだ無い。scripts / docs-viewer と同じ判断。
      reporter: ["text"],
      // node --test --experimental-test-coverage 時代の閾値をそのまま引き継ぐ。
      // 母数を判定モジュールへ絞ってあるぶん、100% は「網羅せよ」ではなく「検査されない分岐を
      // 残さない」を意味する。
      thresholds: { statements: 100, branches: 100, functions: 100, lines: 100 },
    },
  },
});
