const fs = require("fs")
const path = require("path")
const yaml = require("js-yaml")

const MANIFEST = "docs/portal/manifest.yaml"
const OUT_ROOT = "docs/portal/guides"

// --- load manifest ---
if (!fs.existsSync(MANIFEST)) {
  console.error("❌ manifest not found:", MANIFEST)
  process.exit(1)
}

const manifest = yaml.load(fs.readFileSync(MANIFEST, "utf8"))

// --- validate manifest shape ---
if (manifest === null || typeof manifest !== "object" || Array.isArray(manifest)) {
  console.error(`❌ manifest はグループ名をキーとするオブジェクトである必要があります: ${MANIFEST}`)
  process.exit(1)
}

const outRootAbs = path.resolve(OUT_ROOT)

for (const [group, items] of Object.entries(manifest)) {
  if (!Array.isArray(items)) {
    console.error(`❌ [${group}] は配列である必要があります`)
    process.exit(1)
  }

  for (const item of items) {
    if (!item || typeof item.src !== "string" || typeof item.dst !== "string") {
      console.error(`❌ [${group}] の各エントリは文字列の src / dst を持つ必要があります: ${JSON.stringify(item)}`)
      process.exit(1)
    }

    // dst は出力ディレクトリ配下に限定する（パス逸脱で guides 外へ書き込まない）
    const dstAbs = path.resolve(item.dst)
    if (dstAbs !== outRootAbs && !dstAbs.startsWith(outRootAbs + path.sep)) {
      console.error(`❌ [${group}] dst が出力ディレクトリ(${OUT_ROOT})の外を指しています: ${item.dst}`)
      process.exit(1)
    }
  }
}

// --- preflight: manifest が参照する src は全て実在する必要がある（出力を消す前に検証） ---
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
