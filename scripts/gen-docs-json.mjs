import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"
import yaml from "js-yaml"

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const docsDir = path.join(__dirname, "..", "docs")
const portalDir = path.join(docsDir, "portal")

// manifest 内のビューアー構造制御用予約キー
const META_KEY = "meta"

// auto-discovered な「ルート直下 *.md」をまとめる section id
const ROOT_MD_SECTION_ID = "architecture"

// --- helpers ---

function autoTitle(str) {
  return str
    .replace(/\.md$/, "")
    .replace(/\.ja$/, "")
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, c => c.toUpperCase())
}

function slugify(str) {
  return str
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
}

function langOfDst(dst) {
  // dst (manifest) or guides path
  return dst.includes("/ja/") || /\.ja\.md$/.test(dst) ? "ja" : "en"
}

// --- manifest 読み込み ---

const manifestPath = path.join(portalDir, "manifest.yaml")
const manifest = fs.existsSync(manifestPath)
  ? yaml.load(fs.readFileSync(manifestPath, "utf-8"))
  : null

const meta = (manifest && manifest[META_KEY]) || {}
const groupsConfig = Array.isArray(meta.groups) ? meta.groups : []
const sectionTitles = (meta.section_titles && typeof meta.section_titles === "object") ? meta.section_titles : {}
const referenceSectionIds = Array.isArray(meta.reference_links) ? meta.reference_links : []
const referenceSet = new Set(referenceSectionIds)
const subgroupsConfig = (meta.subgroups && typeof meta.subgroups === "object") ? meta.subgroups : {}

// path から guideId を導く: basename から .ja.md / .md を取り除いたもの (EN/JA 両方が同じ id を持つ)
function guideIdFromPath(p) {
  const base = p.split("/").pop() || ""
  return base.replace(/\.ja\.md$/i, "").replace(/\.md$/i, "").replace(/\.html$/i, "")
}

// --- セクション辞書を構築する ---
// 1 section id = 1 エントリ。items は EN/JA を混在で持ち、各 item に lang を付与する。
const sectionMap = new Map()

function ensureSection(id, fallbackTitle) {
  if (!sectionMap.has(id)) {
    sectionMap.set(id, {
      id,
      title: sectionTitles[id] ?? fallbackTitle,
      items: [],
    })
  } else if (sectionTitles[id]) {
    // 既存セクションがあれば title だけ上書き
    sectionMap.get(id).title = sectionTitles[id]
  }
  return sectionMap.get(id)
}

// (a) manifest 由来のセクション
if (manifest) {
  for (const [group, entries] of Object.entries(manifest)) {
    if (group === META_KEY) continue
    if (!Array.isArray(entries)) continue

    const section = ensureSection(group, autoTitle(group.replace(/[-_]/g, " ")))

    for (const entry of entries) {
      const relativePath = entry.dst.replace(/^docs\/portal\//, "./")
      const baseName = path.basename(entry.dst)
      section.items.push({
        name: autoTitle(baseName),
        path: relativePath,
        lang: langOfDst(entry.dst),
        source: entry.src,
        guideId: guideIdFromPath(entry.dst),
      })
    }
  }
}

// (b) auto-discovered: docs/<dir>/index.html (静的 HTML)
// (c) auto-discovered: docs/<dir>/*.md  + docs/ja/<dir>/*.md
const rootDirs = fs.readdirSync(docsDir, { withFileTypes: true })
  .filter(d => d.isDirectory())
  .map(d => d.name)
  .filter(name => name !== "portal" && name !== "ja")
  .sort((a, b) => a.localeCompare(b))

for (const dir of rootDirs) {
  const dirAbs = path.join(docsDir, dir)

  // ① HTML index 優先
  const indexHtml = path.join(dirAbs, "index.html")
  if (fs.existsSync(indexHtml)) {
    const section = ensureSection(dir, autoTitle(dir))
    section.items.push({
      name: section.title,
      path: `../${dir}/index.html`,
      lang: "all",
      source: `docs/${dir}/index.html`,
      guideId: dir,
    })
    continue
  }

  // ② Markdown
  const enFiles = fs.existsSync(dirAbs)
    ? fs.readdirSync(dirAbs).filter(f => f.endsWith(".md")).sort()
    : []
  const jaAbs = path.join(docsDir, "ja", dir)
  const jaFiles = fs.existsSync(jaAbs)
    ? fs.readdirSync(jaAbs).filter(f => f.endsWith(".md")).sort()
    : []

  if (!enFiles.length && !jaFiles.length) continue

  const section = ensureSection(dir, autoTitle(dir))
  for (const f of enFiles) {
    section.items.push({
      name: autoTitle(f),
      path: `../${dir}/${f}`,
      lang: "en",
      source: `docs/${dir}/${f}`,
      guideId: guideIdFromPath(f),
    })
  }
  for (const f of jaFiles) {
    section.items.push({
      name: autoTitle(f),
      path: `../ja/${dir}/${f}`,
      lang: "ja",
      source: `docs/ja/${dir}/${f}`,
      guideId: guideIdFromPath(f),
    })
  }
}

// (d) auto-discovered: docs/*.md と docs/ja/*.md を architecture セクションに集約
{
  const enFiles = fs.readdirSync(docsDir)
    .filter(f => f.endsWith(".md"))
    .sort()
  const jaAbs = path.join(docsDir, "ja")
  const jaFiles = fs.existsSync(jaAbs)
    ? fs.readdirSync(jaAbs).filter(f => f.endsWith(".md")).sort()
    : []

  if (enFiles.length || jaFiles.length) {
    const section = ensureSection(ROOT_MD_SECTION_ID, sectionTitles[ROOT_MD_SECTION_ID] ?? "Architecture Docs")
    for (const f of enFiles) {
      section.items.push({
        name: autoTitle(f),
        path: `../${f}`,
        lang: "en",
        source: `docs/${f}`,
        guideId: guideIdFromPath(f),
      })
    }
    for (const f of jaFiles) {
      section.items.push({
        name: autoTitle(f),
        path: `../ja/${f}`,
        lang: "ja",
        source: `docs/ja/${f}`,
        guideId: guideIdFromPath(f),
      })
    }
  }
}

// --- items 内のソート: EN を先、JA を後、その中で name 昇順 ---
for (const section of sectionMap.values()) {
  section.items.sort((a, b) => {
    const langOrder = (l) => (l === "en" ? 0 : l === "ja" ? 1 : 2)
    const la = langOrder(a.lang)
    const lb = langOrder(b.lang)
    if (la !== lb) return la - lb
    return a.name.localeCompare(b.name)
  })
}

// --- subgroups の適用 (meta.subgroups に登録されたセクションのみ) ---
// section.items は維持しつつ、section.subgroups: [{title, items}] を追加生成する。
// 未割当 item は末尾の "Other" サブグループに入る。
for (const [sectionId, subgroupConfigs] of Object.entries(subgroupsConfig)) {
  const section = sectionMap.get(sectionId)
  if (!section) {
    console.warn(`⚠ meta.subgroups: section id "${sectionId}" は存在しないので無視します`)
    continue
  }
  if (!Array.isArray(subgroupConfigs)) continue

  const guideIdToItems = new Map()
  for (const item of section.items) {
    const gid = item.guideId
    if (!gid) continue
    if (!guideIdToItems.has(gid)) guideIdToItems.set(gid, [])
    guideIdToItems.get(gid).push(item)
  }

  const subgroups = []
  const placedGuideIds = new Set()

  for (const cfg of subgroupConfigs) {
    if (!cfg || typeof cfg.title !== "string") continue
    const guideIds = Array.isArray(cfg.items) ? cfg.items : []
    const items = []
    for (const gid of guideIds) {
      const matched = guideIdToItems.get(gid)
      if (!matched || !matched.length) {
        console.warn(`⚠ meta.subgroups[${sectionId}][${cfg.title}]: guide id "${gid}" は存在しないので無視します`)
        continue
      }
      placedGuideIds.add(gid)
      for (const it of matched) items.push(it)
    }
    if (items.length) {
      subgroups.push({ title: cfg.title, items })
    }
  }

  const others = section.items.filter((it) => !placedGuideIds.has(it.guideId))
  if (others.length) {
    subgroups.push({ title: "Other", items: others })
  }

  if (subgroups.length) {
    section.subgroups = subgroups
  }
}

// --- groups を組み立てる ---

const groups = []
const placedSectionIds = new Set()

for (const groupConfig of groupsConfig) {
  if (!groupConfig || typeof groupConfig.title !== "string") continue
  const sectionIds = Array.isArray(groupConfig.sections) ? groupConfig.sections : []

  const groupSections = []
  for (const id of sectionIds) {
    if (sectionMap.has(id)) {
      groupSections.push(sectionMap.get(id))
      placedSectionIds.add(id)
    } else {
      console.warn(`⚠ meta.groups: section id "${id}" は存在しないので無視します (group: ${groupConfig.title})`)
    }
  }

  if (groupSections.length) {
    groups.push({
      title: groupConfig.title,
      slug: slugify(groupConfig.title),
      sections: groupSections.map((s) => ({ ...s, slug: slugify(s.id) })),
    })
  }
}

// reference_links に指定された section id は groups に配置しない (placed 扱いにする)
for (const id of referenceSet) placedSectionIds.add(id)

// meta.groups に含まれていない残余セクション (manifest 編集忘れ等) は "Uncategorized" にまとめる
const orphans = []
for (const [id, section] of sectionMap.entries()) {
  if (!placedSectionIds.has(id)) orphans.push(section)
}
if (orphans.length) {
  groups.push({
    title: "Uncategorized",
    slug: "uncategorized",
    sections: orphans
      .sort((a, b) => a.title.localeCompare(b.title))
      .map((s) => ({ ...s, slug: slugify(s.id) })),
  })
  console.warn(`⚠ meta.groups 未配置のセクション (${orphans.map(s => s.id).join(", ")}) を "Uncategorized" に集約しました`)
}

// --- reference links を組み立てる ---
// meta.reference_links に書かれた順を維持する。各 section の代表 item (1 個目) を採用。

const referenceLinks = []
for (const id of referenceSectionIds) {
  const section = sectionMap.get(id)
  if (!section) {
    console.warn(`⚠ meta.reference_links: section id "${id}" は存在しないので無視します`)
    continue
  }
  const primary = section.items[0]
  if (!primary) continue
  referenceLinks.push({
    sectionId: id,
    title: section.title,
    path: primary.path,
    source: primary.source,
  })
}

// --- 出力 ---

const docsJson = {
  title: "go-boilerplate Documentation",
  subtitle: "This document portal provides access to the repository implementation, test results, E-R diagrams, and more.",
  groups,
  referenceLinks,
}

fs.writeFileSync(
  path.join(portalDir, "docs.json"),
  JSON.stringify(docsJson, null, 2) + "\n"
)

console.log("docs.json generated")
