import { fileURLToPath } from "node:url";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  test: {
    // ビューアーは DOM を持つ環境でしか成立しない部分（mount / dialog / 描画）を含む。
    environment: "jsdom",
    include: ["src/**/*.test.{ts,tsx}"],
    setupFiles: ["./vitest.setup.ts"],
    coverage: {
      provider: "v8",
      // text だけに絞る。HTML を出すとディスクへ書き出したものを .gitignore で面倒見る話に
      // なるが、Go 側の docs/coverage/ と違って公開する consumer がまだ無い。scripts 側と同じ判断。
      reporter: ["text"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: [
        "src/**/*.test.{ts,tsx}",
        // ビューアーの entry。読み込まれた時点で DOM を触るため import しただけで副作用が出る。
        // 判断はすべて mount/mount-portal.tsx 側にあり、そちらは検査対象に残している。
        "src/main.tsx",
      ],
    },
  },
});
