// ポータル (docs/portal) のフロントエンド（src/main.jsx）を esbuild でバンドルして docs/portal/dist/ へ出力する。

import * as esbuild from "esbuild"
import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.join(__dirname, "..")
const portalDir = path.join(repoRoot, "docs", "portal")
const srcEntry = path.join(portalDir, "src", "main.jsx")
const distDir = path.join(portalDir, "dist")
const nodeModules = path.join(__dirname, "node_modules")

// mermaid は esbuild バンドルに含めず、単体 UMD を dist/ へコピーして遅延ロードで使う。
const mermaidSrc = path.join(nodeModules, "mermaid", "dist", "mermaid.min.js")
const mermaidDest = path.join(distDir, "mermaid.min.js")

function resetDistDir() {
  fs.rmSync(distDir, { recursive: true, force: true })
  fs.mkdirSync(distDir, { recursive: true })
}

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
    jsx: "automatic",
    nodePaths: [nodeModules],
    entryNames: "bundle",
    chunkNames: "chunk-[name]-[hash]",
    assetNames: "asset-[name]-[hash]",
    legalComments: "none",
    define: { "process.env.NODE_ENV": '"production"' },
    metafile: true,
  })

  for (const [file, meta] of Object.entries(result.metafile.outputs)) {
    console.log(`  ${file}  ${(meta.bytes / 1024).toFixed(1)} KiB`)
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
