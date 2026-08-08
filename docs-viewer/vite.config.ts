import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  // 配信先のパス接頭辞を持たない。サイトのどの位置へ置いても動くようにして、
  // portal の URL を配信側の都合で決められる状態にする。
  base: "./",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      // design-system の部品が内部で使う `@/` を、このパッケージのソースへ解決する。
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  build: {
    // GitHub Pages は docs/ をそのままサイトルートとして配信するため、成果物は配信ツリーへ
    // 直接書き出してコミットする。index.html は docs/portal/ 直下、資産は dist/ 配下へ置く。
    outDir: "../docs/portal",
    assetsDir: "dist",
    // docs.json / guides/ / manifest.yaml と同じディレクトリへ出すため、outDir は空にできない。
    // 前回の資産は build 前に scripts/clean-dist.mjs が落とす。
    emptyOutDir: false,
  },
});
