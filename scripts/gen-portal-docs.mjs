import fs from "fs"
import path from "path"
import yaml from "js-yaml"

const MANIFEST = "docs/portal/manifest.yaml"
const OUT_ROOT = "docs/portal/guides"

// --- load manifest ---
if (!fs.existsSync(MANIFEST)) {
  console.error("❌ manifest not found:", MANIFEST)
  process.exit(1)
}

const manifest = yaml.load(fs.readFileSync(MANIFEST, "utf8"))

// --- clean ---
console.log("🧹 Cleaning output directory...")
fs.rmSync(OUT_ROOT, { recursive: true, force: true })
fs.mkdirSync(OUT_ROOT, { recursive: true })

console.log("🚀 Generating docs from manifest...")

// --- generate ---
for (const [group, items] of Object.entries(manifest)) {
  for (const { src, dst } of items) {

    if (!fs.existsSync(src)) {
      console.warn(`⚠️  [${group}] src not found: ${src}`)
      continue
    }

    fs.mkdirSync(path.dirname(dst), { recursive: true })
    fs.copyFileSync(src, dst)

    console.log(`✔ [${group}] ${src} -> ${dst}`)
  }
}

console.log("✅ Docs generation completed.")
