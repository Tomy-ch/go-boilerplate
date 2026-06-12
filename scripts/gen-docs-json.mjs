import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"
import yaml from "js-yaml"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const docsDir = path.join(__dirname, "..", "docs")
const portalDir = path.join(docsDir, "portal")

function title(str) {
  return str
    .replace(".md", "")
    .replace(".ja", "")
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, c => c.toUpperCase())
}

// ディレクトリ直下の .md からセクションを構築する（ファイル名順）。
// 対象が存在しない / md が無ければ null を返す。
function mdSection(sectionTitle, dirPath, pathPrefix) {
  if (!fs.existsSync(dirPath)) {
    return null
  }

  const files = fs.readdirSync(dirPath)
    .filter(f => f.endsWith(".md"))
    .sort((a, b) => a.localeCompare(b))

  if (!files.length) {
    return null
  }

  return {
    title: sectionTitle,
    items: files.map(f => ({ name: title(f), path: `${pathPrefix}${f}` }))
  }
}

function generateSections() {
  const manifestPath = path.join(portalDir, "manifest.yaml")

  const sections = []
  let manifest = null

  if (fs.existsSync(manifestPath)) {
    manifest = yaml.load(fs.readFileSync(manifestPath, "utf-8"))
  } else {
    console.warn("manifest.yaml not found; using filesystem scan fallback")
  }

  if (manifest) {
    for (const [group, entries] of Object.entries(manifest)) {
      const enItems = []
      const jaItems = []

      for (const entry of entries) {
        const name = title(path.basename(entry.dst))
        const relativePath = entry.dst.replace(/^docs\/portal\//, "./")

        if (relativePath.includes("/ja/")) {
          jaItems.push({ name, path: relativePath })
        } else {
          enItems.push({ name, path: relativePath })
        }
      }

      if (enItems.length) {
        sections.push({
          title: `${title(group)} (English)`,
          items: enItems.sort((a, b) => a.name.localeCompare(b.name))
        })
      }

      if (jaItems.length) {
        sections.push({
          title: `${title(group)} (Japanese)`,
          items: jaItems.sort((a, b) => a.name.localeCompare(b.name))
        })
      }
    }
  }

  const rootDirs = fs.readdirSync(docsDir, { withFileTypes: true })
    .filter(d => d.isDirectory())
    .map(d => d.name)
    .filter(name => name !== "portal" && name !== "ja")
    .sort((a, b) => a.localeCompare(b))

  for (const dir of rootDirs) {
    const dirPath = path.join(docsDir, dir)

    // ① HTML (priority)
    const indexPath = path.join(dirPath, "index.html")
    if (fs.existsSync(indexPath)) {
      sections.push({
        title: title(dir),
        items: [
          {
            name: title(dir),
            path: `../${dir}/index.html`
          }
        ]
      })
      continue
    }

    // ② Markdown (EN + JA)
    const en = mdSection(`${title(dir)} (English)`, dirPath, `../${dir}/`)
    if (en) {
      sections.push(en)
    }

    const ja = mdSection(`${title(dir)} (Japanese)`, path.join(docsDir, "ja", dir), `../ja/${dir}/`)
    if (ja) {
      sections.push(ja)
    }
  }

  // root-level markdown (architecture.md など): manifest には載せず常に Architecture 節として surface する
  const rootEn = mdSection("Architecture (English)", docsDir, "../")
  if (rootEn) {
    sections.push(rootEn)
  }

  const rootJa = mdSection("Architecture (Japanese)", path.join(docsDir, "ja"), "../ja/")
  if (rootJa) {
    sections.push(rootJa)
  }

  return sections
}

const docsJson = {
  title: "go-boilerplate Documentation",
  subtitle: "This document portal provides access to the repository implementation, test results, E-R diagrams, and more.",
  sections: generateSections()
}

fs.writeFileSync(
  path.join(portalDir, "docs.json"),
  JSON.stringify(docsJson, null, 2) + "\n"
)

console.log("docs.json generated")
