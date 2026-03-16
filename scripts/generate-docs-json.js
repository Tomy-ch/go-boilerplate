const fs = require("fs")
const path = require("path")

const docsDir = path.join(__dirname, "..", "docs")
const portalDir = path.join(docsDir, "portal")
const jaDir = path.join(docsDir, "ja")

const guidesDir = path.join(portalDir, "guides")
const guidesJaDir = path.join(guidesDir, "ja")

const RESERVED = ["portal", "ja"]

function title(str) {
  return str
    .replace(".md", "")
    .replace(".ja", "")
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, c => c.toUpperCase())
}

function listFiles(dir, ext) {
  if (!fs.existsSync(dir)) return []
  return fs.readdirSync(dir).filter(f => f.endsWith(ext))
}

function mdItems(dir, prefix) {
  return listFiles(dir, ".md").map(f => ({
    name: title(f),
    path: `${prefix}/${f}`
  }))
}

function htmlItems(dir, prefix) {

  if (!fs.existsSync(dir)) return []

  const indexPath = path.join(dir, "index.html")

  if (!fs.existsSync(indexPath)) return []

  return [{
    name: title(path.basename(dir)),
    path: `${prefix}/index.html`
  }]
}

function generateSections() {

  const sections = []

  // ----------------
  // Guides (portal aggregated docs)
  // ----------------

  const guideEN = mdItems(guidesDir, "./guides")
  const guideJA = mdItems(guidesJaDir, "./guides/ja")

  if (guideEN.length) {
    sections.push({
      title: "Guides (English)",
      items: guideEN
    })
  }

  if (guideJA.length) {
    sections.push({
      title: "Guides (Japanese)",
      items: guideJA
    })
  }

  // ----------------
  // Root docs
  // ----------------

  const enRoot = mdItems(docsDir, "..")
    .filter(item => !item.path.startsWith("./"))

  const jaRoot = mdItems(jaDir, "../ja")
    .filter(item => !item.path.startsWith("../ja/"))

  if (enRoot.length) {
    sections.push({
      title: "Architecture (English)",
      items: enRoot
    })
  }

  if (jaRoot.length) {
    sections.push({
      title: "Architecture (Japanese)",
      items: jaRoot
    })
  }

  // ----------------
  // Directory sections
  // ----------------

  const dirs = fs.readdirSync(docsDir, { withFileTypes: true })
    .filter(d => d.isDirectory())
    .map(d => d.name)
    .filter(d => !RESERVED.includes(d))

  for (const dir of dirs) {

    const enDir = path.join(docsDir, dir)
    const jaDirSub = path.join(jaDir, dir)

    const enMd = mdItems(enDir, `../${dir}`)
    const jaMd = mdItems(jaDirSub, `../ja/${dir}`)

    if (enMd.length) {
      sections.push({
        title: `${title(dir)} (English)`,
        items: enMd
      })
    }

    if (jaMd.length) {
      sections.push({
        title: `${title(dir)} (Japanese)`,
        items: jaMd
      })
    }

    const htmlDocs = htmlItems(enDir, `../${dir}`)

    if (htmlDocs.length) {
      sections.push({
        title: title(dir),
        items: htmlDocs
      })
    }
  }

  return sections
}

const docsJson = {
  title: "Go Boilerplate Documentation",
  subtitle: "Golang × Echo × OpenAPI × PostgreSQL",
  sections: generateSections()
}

fs.writeFileSync(
  path.join(portalDir, "docs.json"),
  JSON.stringify(docsJson, null, 2)
)

console.log("docs.json generated")
