// ----------------
// Imports
// ----------------
//
// CDN + ブラウザ内 Babel をやめ、esbuild でこのファイルを起点にバンドルする。
// React / marked / Fuse は初期描画に必要なため eager import。
// highlight.js は Markdown を開いた時だけ使うため動的 import() で遅延ロードする
// （esbuild の code splitting で別チャンク化）。
// mermaid は単体 3MB 超かつ図種ごとに多数の動的チャンクへ分割されるため、
// esbuild には含めず dist/ 配下の単一 UMD ファイルを遅延 script 注入で読み込む
// （コミット対象を 1 ファイルに集約しつつ初期ロードからは除外する）。

import React from "react"
import { createRoot } from "react-dom/client"
import { marked } from "marked"
import Fuse from "fuse.js"

// highlight.js のテーマ CSS。esbuild がバンドルして bundle.css として書き出す。
import "highlight.js/styles/github.css"
import "highlight.js/styles/github-dark.css"

// ----------------
// Lazy mermaid loader
// ----------------

// dist/mermaid.min.js (UMD) を初回のみ <script> 注入で読み込み、window.mermaid を返す。
let mermaidPromise = null
function loadMermaid() {
  if (window.mermaid) return Promise.resolve(window.mermaid)
  if (!mermaidPromise) {
    mermaidPromise = new Promise((resolve, reject) => {
      const script = document.createElement("script")
      script.src = "./dist/mermaid.min.js"
      script.onload = () => resolve(window.mermaid)
      script.onerror = () => reject(new Error("failed to load mermaid"))
      document.head.appendChild(script)
    })
  }
  return mermaidPromise
}

// ----------------
// Hash routing
// ----------------

function parseHash() {
  const raw = window.location.hash.replace(/^#\/?/, "")
  return raw || ""
}

// ----------------
// Components
// ----------------

function Card({ name, path, source, onOpen }) {

  const isMarkdown = path.endsWith(".md")
  // markdown 以外 (HTML index / 外部 URL) は新タブで開く。
  const opensInNewTab = !isMarkdown

  return (
    <a
      className={`card ${isMarkdown ? "card-md" : "card-link"}`}
      href={path}
      target={opensInNewTab ? "_blank" : undefined}
      rel={opensInNewTab ? "noopener noreferrer" : undefined}
      onClick={(e) => {
        if (isMarkdown) {
          e.preventDefault()
          onOpen(path)
        }
      }}
    >
      <div className="card-title">{name}</div>
      <div className="card-desc">{source || path}</div>
    </a>
  )
}

function CardGrid({ items, keyPrefix, onOpen }) {
  return (
    <div className="cards">
      {items.map((item) => (
        <Card
          key={`${keyPrefix}-${item.name}-${item.path}`}
          {...item}
          onOpen={onOpen}
        />
      ))}
    </div>
  )
}

function Section({ slug, title, items, subgroups, onOpen }) {

  // subgroups が有効ならサブグループ単位で表示。無ければ従来通り flat。
  const useSubgroups = Array.isArray(subgroups) && subgroups.length > 0

  if (!useSubgroups && (!items || !items.length)) return null

  return (
    <section className="section" id={slug ? `section-${slug}` : undefined}>
      <h3 className="section-title">{title}</h3>
      {useSubgroups ? (
        subgroups
          .filter((sg) => sg.items && sg.items.length > 0)
          .map((sg) => (
            <div key={sg.title} className="subgroup">
              <h4 className="subgroup-title">{sg.title}</h4>
              <CardGrid
                items={sg.items}
                keyPrefix={`${title}-${sg.title}`}
                onOpen={onOpen}
              />
            </div>
          ))
      ) : (
        <CardGrid items={items} keyPrefix={title} onOpen={onOpen} />
      )}
    </section>
  )
}

function Sidebar({ groups, activeSlug, onSelectGroup, referenceLinks = [] }) {

  return (
    <aside className="sidebar">
      <nav className="sidebar-nav">
        {groups.map((group) => {
          const isActive = group.slug === activeSlug
          return (
            <div key={group.slug} className="sidebar-group">
              <button
                type="button"
                className={`sidebar-item ${isActive ? "active" : ""}`}
                onClick={() => onSelectGroup(group.slug)}
              >
                {group.title}
              </button>
              {isActive ? (
                <ul className="sidebar-sublist">
                  {group.sections.map((section) => {
                    const subHash = `#/${group.slug}/${section.slug}`
                    return (
                      <li key={section.slug}>
                        <a
                          href={subHash}
                          onClick={(e) => {
                            e.preventDefault()
                            // history 汚染を避けるため replaceState で URL を更新する。
                            // ブックマーク / 共有時に該当セクションまで復元可能になる。
                            if (window.location.hash !== subHash) {
                              window.history.replaceState(null, "", subHash)
                            }
                            const el = document.getElementById(`section-${section.slug}`)
                            if (el) el.scrollIntoView({ behavior: "smooth", block: "start" })
                          }}
                        >
                          {section.title}
                        </a>
                      </li>
                    )
                  })}
                </ul>
              ) : null}
            </div>
          )
        })}
      </nav>

      {referenceLinks.length ? (
        <div className="sidebar-reference">
          <div className="sidebar-reference-title">Reference</div>
          <ul className="sidebar-reference-list">
            {referenceLinks.map((link) => (
              <li key={link.sectionId}>
                <a
                  className="sidebar-reference-link"
                  href={link.path}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <span>{link.title}</span>
                  <span className="sidebar-reference-arrow" aria-hidden="true">↗</span>
                </a>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </aside>
  )
}

function Search({ allItems = [], onResult }) {

  const [query, setQuery] = React.useState("")

  // Fuse インスタンスは allItems が変わった時だけ再生成。
  // (毎キーストロークで再ビルドすると O(n) のインデックス構築が無駄に走る)
  const fuse = React.useMemo(
    () =>
      new Fuse(allItems, {
        keys: ["name", "sectionTitle", "groupTitle", "source", "path"],
        threshold: 0.3,
      }),
    [allItems]
  )

  React.useEffect(() => {

    if (!query.trim()) {
      onResult(null)
      return
    }

    const results = fuse.search(query).map((r) => r.item)
    onResult(results)

  }, [query, fuse, onResult])

  return (
    <div className="search">
      <input
        type="text"
        placeholder="Search documentation..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
    </div>
  )
}

function MarkdownViewer({ html, onClose }) {

  const ref = React.useRef(null)

  React.useEffect(() => {
    if (!ref.current) return
    const container = ref.current

    // markdown 内リンクは無効化 (portal は閲覧専用)
    container.querySelectorAll("a").forEach((link) => {
      link.addEventListener("click", (e) => e.preventDefault())
      link.style.pointerEvents = "none"
      link.style.color = "#6e7781"
      link.style.textDecoration = "none"
    })

    // ```mermaid フェンスを <div class="mermaid"> に置き換える
    const mermaidNodes = []
    container.querySelectorAll("code.language-mermaid").forEach((block) => {
      const parent = block.parentElement
      const graph = block.textContent
      const div = document.createElement("div")
      div.className = "mermaid"
      div.textContent = graph
      parent.replaceWith(div)
      mermaidNodes.push(div)
    })

    // Mermaid render — vendor の単一 UMD を遅延ロードし、
    // modal 内のノードだけを対象にすることで全体スキャンを避ける。
    if (mermaidNodes.length) {
      loadMermaid()
        .then((mermaid) => {
          mermaid.initialize({ startOnLoad: false })
          mermaid.run({ nodes: mermaidNodes })
        })
        .catch((err) => console.error(err))
    }

    // Code highlight — highlight.js (common 言語サブセット) を動的 import で
    // 遅延ロードし、modal 内の <pre><code> だけ highlightElement で処理する。
    const codeBlocks = container.querySelectorAll("pre code")
    if (codeBlocks.length) {
      import("highlight.js/lib/common").then(({ default: hljs }) => {
        codeBlocks.forEach((block) => hljs.highlightElement(block))
      })
    }
  }, [html])

  if (!html) return null

  return (
    <div className="md-modal">
      <div className="md-backdrop" onClick={onClose}></div>
      <div className="md-dialog">
        <div className="md-toolbar">
          <button onClick={onClose}>Close</button>
        </div>
        <div
          ref={ref}
          className="md-content"
          dangerouslySetInnerHTML={{ __html: html }}
        />
      </div>
    </div>
  )
}

// ----------------
// Lang filter
// ----------------

// effectiveLang はセクション単位で決定する。
// JA モードでも JA item が一切無いセクションは EN にフォールバックさせる。
// セクション内のサブグループは「決まった lang」を強制的に共有するため、
// サブグループごとに EN/JA が混在する事故 (例: 同じ Controller セクション内で
// Layer Top が JA、HTTP Stack が EN) を避けられる。
function effectiveLangFor(items, lang) {
  if (lang !== "JA") return "EN"
  const hasJa = (items || []).some(i => i.lang === "ja")
  return hasJa ? "JA" : "EN"
}

function filterItemsByLangStrict(items, effectiveLang) {
  if (!items || !items.length) return []

  const all = items.filter(i => i.lang === "all")
  if (effectiveLang === "EN") return [...all, ...items.filter(i => i.lang === "en")]
  return [...all, ...items.filter(i => i.lang === "ja")]
}

function applyLangFilter(groups, lang) {
  if (!Array.isArray(groups)) return []

  return groups
    .map((group) => ({
      title: group.title,
      slug: group.slug,
      sections: (group.sections || [])
        .map((section) => {
          const sectionLang = effectiveLangFor(section.items, lang)
          const filteredItems = filterItemsByLangStrict(section.items, sectionLang)
          const filteredSubgroups = Array.isArray(section.subgroups)
            ? section.subgroups
                .map((sg) => ({
                  title: sg.title,
                  items: filterItemsByLangStrict(sg.items, sectionLang),
                }))
                .filter((sg) => sg.items.length > 0)
            : null
          return {
            id: section.id,
            slug: section.slug,
            title: section.title,
            items: filteredItems,
            subgroups: filteredSubgroups,
          }
        })
        // items / subgroups どちらかにコンテンツが残っているセクションは保持する。
        // (将来 section.items を持たず subgroups だけで提供するスキーマでも飛ばない)
        .filter((section) => section.items.length > 0 || (section.subgroups && section.subgroups.length > 0)),
    }))
    .filter((group) => group.sections.length > 0)
}

// ----------------
// Search corpus
// ----------------

function buildAllItems(groups) {
  const items = []
  for (const group of groups) {
    for (const section of group.sections) {
      for (const item of section.items) {
        items.push({
          ...item,
          sectionTitle: section.title,
          groupTitle: group.title,
        })
      }
    }
  }
  return items
}

// ----------------
// Main App
// ----------------

// pushState/replaceState で hash を更新するヘルパ。
// 直接 window.location.hash に代入すると history エントリーが追加されて
// 戻るボタンが「直前の hash 状態」に戻る → useEffect が再書き込み → トラップ、
// になる。replaceState で同じ entry を上書きすればトラップを防げる。
function replaceHashSlug(slug) {
  const next = `#/${slug}`
  if (window.location.hash !== next) {
    window.history.replaceState(null, "", next)
    // hashchange は dispatchEvent で同期発火 (replaceState 自身は発火しない)
    window.dispatchEvent(new HashChangeEvent("hashchange"))
  }
}

function App() {

  const [docs, setDocs] = React.useState(null)
  const [filtered, setFiltered] = React.useState(null)
  const [lang, setLang] = React.useState("EN")
  const [mdHtml, setMdHtml] = React.useState(null)
  const [hashSlug, setHashSlug] = React.useState(() => parseHash())

  React.useEffect(() => {
    fetch("./docs.json")
      .then((res) => {
        if (!res.ok) throw new Error(`failed to load docs.json: ${res.status}`)
        return res.json()
      })
      .then(setDocs)
      .catch((err) => console.error(err))
  }, [])

  React.useEffect(() => {
    const onHashChange = () => setHashSlug(parseHash())
    window.addEventListener("hashchange", onHashChange)
    return () => window.removeEventListener("hashchange", onHashChange)
  }, [])

  const openMarkdown = React.useCallback((path) => {
    fetch(path)
      .then((res) => {
        if (!res.ok) throw new Error(`failed to load markdown: ${res.status}`)
        return res.text()
      })
      .then((md) => setMdHtml(marked.parse(md)))
      .catch((err) => console.error(err))
  }, [])

  // docs.json は v2 (groups) を期待。古い (sections) 形式が来た場合は単一 "All" グループに包む。
  const rawGroups = React.useMemo(() => {
    if (!docs) return []
    return Array.isArray(docs.groups)
      ? docs.groups
      : [{ title: "All", slug: "all", sections: docs.sections || [] }]
  }, [docs])

  // 言語フィルタ後の visible groups。allItems もここから派生させ、毎レンダーで
  // 新規参照が生まれないようメモ化する (Search の useEffect 依存配列を安定させ、
  // 入力ごとに setFiltered → re-render → 新 allItems → effect 再発火 → setFiltered…
  // という無限ループを防ぐ)。
  const visibleGroups = React.useMemo(() => applyLangFilter(rawGroups, lang), [rawGroups, lang])
  const allItems = React.useMemo(() => buildAllItems(visibleGroups), [visibleGroups])

  // 初期 hash 未設定 / hash が visibleGroups に該当しない場合は、
  // 言語フィルタ後の先頭グループに replaceState で揃える (history エントリ汚染を回避)。
  React.useEffect(() => {
    if (!docs || !visibleGroups.length) return
    const requestedGroupSlug = parseHash().split("/")[0]
    if (!requestedGroupSlug || !visibleGroups.find((g) => g.slug === requestedGroupSlug)) {
      replaceHashSlug(visibleGroups[0].slug)
    }
  }, [docs, visibleGroups])

  const onSelectGroup = React.useCallback((slug) => {
    setFiltered(null)
    replaceHashSlug(slug)
    window.scrollTo(0, 0)
  }, [])

  if (!docs) {
    return <div className="loading">Loading...</div>
  }

  // hash の先頭 (group slug) を抽出
  const requestedGroupSlug = hashSlug.split("/")[0] || ""
  const activeGroup =
    visibleGroups.find((g) => g.slug === requestedGroupSlug) ||
    visibleGroups[0] ||
    null
  const activeSlug = activeGroup ? activeGroup.slug : ""

  return (
    <div className="layout">

      <header>
        <div className="container header-inner">
          <h1>{docs.title}</h1>
          <p>{docs.subtitle}</p>

          <div className="toolbar">
            <Search
              allItems={allItems}
              onResult={setFiltered}
            />
            <div className="lang-toggle">
              <button
                type="button"
                className={lang === "EN" ? "active" : ""}
                onClick={() => { setLang("EN"); setFiltered(null) }}
              >EN</button>
              <button
                type="button"
                className={lang === "JA" ? "active" : ""}
                onClick={() => { setLang("JA"); setFiltered(null) }}
              >JA</button>
            </div>
          </div>
        </div>
      </header>

      <MarkdownViewer
        html={mdHtml}
        onClose={() => setMdHtml(null)}
      />

      <div className="container body">

        <Sidebar
          groups={visibleGroups}
          activeSlug={activeSlug}
          referenceLinks={Array.isArray(docs.referenceLinks) ? docs.referenceLinks : []}
          onSelectGroup={onSelectGroup}
        />

        <main className="content">
          {filtered ? (
            <div className="page">
              <h2 className="page-title">Search Results</h2>
              {filtered.length === 0 ? (
                <div className="empty">No results matched your query.</div>
              ) : (
                <Section
                  title={`${filtered.length} hit${filtered.length === 1 ? "" : "s"}`}
                  items={filtered}
                  onOpen={openMarkdown}
                />
              )}
            </div>
          ) : activeGroup ? (
            <div className="page">
              <h2 className="page-title">{activeGroup.title}</h2>
              <div className="page-body">
                {activeGroup.sections.map((section) => (
                  <Section
                    key={section.slug}
                    slug={section.slug}
                    title={section.title}
                    items={section.items}
                    subgroups={section.subgroups}
                    onOpen={openMarkdown}
                  />
                ))}
              </div>
            </div>
          ) : (
            <div className="empty">No content available.</div>
          )}
        </main>

      </div>

    </div>
  )
}

// ----------------
// Render
// ----------------

createRoot(document.getElementById("root")).render(<App />)
