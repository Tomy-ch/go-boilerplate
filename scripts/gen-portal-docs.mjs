import fs from "node:fs"
import path from "node:path"
import yaml from "js-yaml"
import { z } from "zod"

const MANIFEST = "docs/portal/manifest.yaml"
const OUT_ROOT = "docs/portal/guides"

// --- load manifest ---
if (!fs.existsSync(MANIFEST)) {
  console.error("❌ manifest not found:", MANIFEST)
  process.exit(1)
}

// manifest はグループ名をキーに、{ src, dst } の配列を値とするオブジェクト
const ManifestSchema = z.record(
  z.string(),
  z.array(z.object({ src: z.string(), dst: z.string() }))
)

// --- validate manifest shape ---
const parsed = ManifestSchema.safeParse(yaml.load(fs.readFileSync(MANIFEST, "utf8")))
if (!parsed.success) {
  console.error(`❌ manifest の形式が不正です（${MANIFEST}）:`)
  for (const issue of parsed.error.issues) {
    console.error(`  - ${issue.path.join(".") || "(root)"}: ${issue.message}`)
  }
  process.exit(1)
}
const manifest = parsed.data

const outRootAbs = path.resolve(OUT_ROOT)

// dst は出力ディレクトリ配下に限定する（パストラバーサル防止）
for (const [group, items] of Object.entries(manifest)) {
  for (const { dst } of items) {
    const dstAbs = path.resolve(dst)
    if (dstAbs !== outRootAbs && !dstAbs.startsWith(outRootAbs + path.sep)) {
      console.error(`❌ [${group}] dst が出力ディレクトリ(${OUT_ROOT})の外を指しています: ${dst}`)
      process.exit(1)
    }
  }
}

// --- preflight: src の実在チェック（clean 前に検証） ---
const missing = []

for (const [group, items] of Object.entries(manifest)) {
  for (const { src } of items) {
    if (!fs.existsSync(src)) {
      missing.push(`[${group}] ${src}`)
    }
  }
}

if (missing.length) {
  console.error("❌ manifest が参照する src が見つかりません（manifest の陳腐化を解消してください）:")
  for (const entry of missing) {
    console.error(`  - ${entry}`)
  }
  process.exit(1)
}

// --- clean ---
console.log("🧹 Cleaning output directory...")
fs.rmSync(OUT_ROOT, { recursive: true, force: true })
fs.mkdirSync(OUT_ROOT, { recursive: true })

console.log("🚀 Generating docs from manifest...")

// --- generate ---
for (const [group, items] of Object.entries(manifest)) {
  for (const { src, dst } of items) {
    fs.mkdirSync(path.dirname(dst), { recursive: true })
    fs.copyFileSync(src, dst)

    console.log(`✔ [${group}] ${src} -> ${dst}`)
  }
}

console.log("✅ Docs generation completed.")
