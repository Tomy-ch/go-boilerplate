import fs from "fs"
import path from "path"
import { fileURLToPath } from "url"
import yaml from "js-yaml"

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const docsDir = path.join(__dirname, "..", "docs")
const portalDir = path.join(docsDir, "portal")

function title(str) {
  return str
    .replace(".md", "")
    .replace(".ja", "")
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, c => c.toUpperCase())
}

function generateSections() {
  const manifestPath = path.join(portalDir, "manifest.yaml")

  if (!fs.existsSync(manifestPath)) {
    console.warn("manifest.yaml not found, fallback to legacy behavior")
    return []
  }

  const manifest = yaml.load(fs.readFileSync(manifestPath, "utf-8"))

  const sections = []

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


  const rootDirs = fs.readdirSync(docsDir, { withFileTypes: true })
    .filter(d => d.isDirectory())
    .map(d => d.name)
    .filter(name => name !== "portal")
    .sort((a, b) => a.localeCompare(b))

  for (const dir of rootDirs) {

    const indexPath = path.join(docsDir, dir, "index.html")

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

    }
  }

  return sections
}

const docsJson = {
  title: "boilerplate-go Documentation",
  subtitle: "This document portal provides access to the repository implementation, test results, E-R diagrams, and more.",
  sections: generateSections()
}

fs.writeFileSync(
  path.join(portalDir, "docs.json"),
  JSON.stringify(docsJson, null, 2) + "\n"
)

console.log("docs.json generated")
