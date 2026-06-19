// ポータル (docs/portal) のフロントエンドを esbuild でバンドルするスクリプト。
//
// 背景: 旧構成は CDN から React / Babel 等を読み込み、ブラウザ内 Babel で
// main.jsx を変換していた。Babel 8 で preset-react の既定が automatic JSX
// runtime になり `import "react/jsx-runtime"` を出力 → バンドラ無しのブラウザ
// では解決できず描画が丸ごと停止する事故が起きた。
//
// 本スクリプトは docs/portal/src/main.jsx を起点に依存を node_modules から解決し、
// 生成物を docs/portal/dist/ へ書き出す。これにより外部ネットワーク依存とブラウザ内
// Babel を廃止し、手書き（src/）と生成物（dist/）をディレクトリで明確に分離する。
// mermaid / highlight.js は src 側の遅延ロードで初期表示から外す（後述）。
//
// 実行は host ではなく node_tool_runner コンテナ内（make gen-portal-build）。

import * as esbuild from "esbuild"
import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.join(__dirname, "..")
const portalDir = path.join(repoRoot, "docs", "portal")
const srcEntry = path.join(portalDir, "src", "main.jsx")
// 生成物はすべて dist/ に集約する（手書きの index.html / styles.css / src/ と分離）。
const distDir = path.join(portalDir, "dist")
// 依存はコンテナの /app/scripts/node_modules（匿名ボリューム）に存在する。
const nodeModules = path.join(__dirname, "node_modules")

// mermaid は単体 3MB 超かつ図種ごとに多数の動的チャンクへ分割されるため、
// esbuild には含めず単一 UMD を dist/ へコピーして遅延 script 注入で使う。
const mermaidSrc = path.join(nodeModules, "mermaid", "dist", "mermaid.min.js")
const mermaidDest = path.join(distDir, "mermaid.min.js")

// dist/ を毎回まっさらにしてから生成する（ハッシュ名チャンクの残骸蓄積を防ぐ）。
function resetDistDir() {
  fs.rmSync(distDir, { recursive: true, force: true })
  fs.mkdirSync(distDir, { recursive: true })
}

// mermaid UMD を docs/portal/dist/ へコピーする。
function vendorMermaid() {
  fs.copyFileSync(mermaidSrc, mermaidDest)
  const bytes = fs.statSync(mermaidDest).size
  console.log(`  ${path.relative(repoRoot, mermaidDest)}  ${(bytes / 1024).toFixed(1)} KiB (vendored)`)
}

async function main() {
  resetDistDir()
  vendorMermaid()

  const result = await esbuild.build({
    entryPoints: [srcEntry],
    outdir: distDir,
    bundle: true,
    format: "esm",
    splitting: true,
    minify: true,
    sourcemap: false,
    target: ["es2020"],
    jsx: "automatic", // バンドラが react/jsx-runtime を node_modules から解決する
    nodePaths: [nodeModules],
    entryNames: "bundle",
    chunkNames: "chunk-[name]-[hash]",
    assetNames: "asset-[name]-[hash]",
    legalComments: "none",
    define: { "process.env.NODE_ENV": '"production"' },
    metafile: true,
  })

  // 出力サマリを表示（生成物のサイズ把握用）。
  for (const [file, meta] of Object.entries(result.metafile.outputs)) {
    console.log(`  ${file}  ${(meta.bytes / 1024).toFixed(1)} KiB`)
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
