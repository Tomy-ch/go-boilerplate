// Component
function Card({ name, path, onOpen }) {

  const isMarkdown = path.endsWith(".md")

  return (
    <a
      className={`card ${isMarkdown ? "card-md" : "card-link"}`}
      href={path}
      onClick={(e) => {
        if (isMarkdown) {
          e.preventDefault()
          onOpen(path)
        }
      }}
    >
      <div className="card-title">{name}</div>
      <div className="card-desc">{path}</div>
    </a>
  )
}

function Section({ title, items, onOpen }) {
  return (
    <section className="section">
      <h2>{title}</h2>
      <div className="cards">
        {(items || []).map((item) => (
          <Card
            key={`${title}-${item.name}-${item.path}`}
            {...item}
            onOpen={onOpen}
          />
        ))}
      </div>
    </section>
  )
}

function Search({ sections = [], onResult }) {

  const [query, setQuery] = React.useState("")

  React.useEffect(() => {

    if (!query.trim()) {
      onResult(null)
      return
    }

    const items = sections.flatMap((section) =>
      (section.items || []).map((item) => ({
        ...item,
        sectionTitle: section.title,
      }))
    )

    const fuse = new Fuse(items, {
      keys: ["name", "sectionTitle", "path"],
      threshold: 0.3,
    })

    const results = fuse.search(query).map((r) => r.item)

    onResult(results)

  }, [query, sections, onResult])

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
    // markdown内リンク無効化
    const links = ref.current.querySelectorAll("a")
    links.forEach(link => {
      link.addEventListener("click", (e) => {
        e.preventDefault()
      })
      link.style.pointerEvents = "none"
      link.style.color = "#6e7781"
      link.style.textDecoration = "none"
    })

    // Mermaid render
    const blocks = ref.current.querySelectorAll("code.language-mermaid")
    blocks.forEach((block) => {
      const parent = block.parentElement
      const graph = block.textContent
      const div = document.createElement("div")
      div.className = "mermaid"
      div.textContent = graph
      parent.replaceWith(div)
    })

	if (window.mermaid) {
      mermaid.initialize({ startOnLoad: false })
      mermaid.run()
    }

	// ----------------
    // Code highlight
    // ----------------

    if (window.hljs) {
      hljs.highlightAll()
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

// Utils

function normalizeSectionTitle(title) {
  return title
    .replace(" (English)", "")
    .replace(" (Japanese)", "")
}

function buildVisibleSections(allSections, lang) {

  if (!Array.isArray(allSections)) return []

  if (lang === "EN") {
    return allSections.filter((section) => !section.title.includes("Japanese"))
  }

  const jaSections = new Map()
  const enSections = new Map()

  allSections.forEach((section) => {

    const base = normalizeSectionTitle(section.title)

    if (section.title.includes("Japanese")) {
      jaSections.set(base, section)
      return
    }

    enSections.set(base, section)

  })

  const orderedBases = []

  allSections.forEach((section) => {
    const base = normalizeSectionTitle(section.title)
    if (!orderedBases.includes(base)) {
      orderedBases.push(base)
    }
  })

  return orderedBases
    .map((base) => jaSections.get(base) || enSections.get(base))
    .filter(Boolean)
}

function sortSections(sections) {

  return [...sections].sort((a, b) => {

    const aGuide = a.title.toLowerCase().includes("guide")
    const bGuide = b.title.toLowerCase().includes("guide")

    if (aGuide && !bGuide) return -1
    if (!aGuide && bGuide) return 1

    return 0
  })
}

// Main App

function App() {

  const [docs, setDocs] = React.useState(null)
  const [filtered, setFiltered] = React.useState(null)
  const [lang, setLang] = React.useState("EN")
  const [mdHtml, setMdHtml] = React.useState(null)

  React.useEffect(() => {

    fetch("./docs.json")
      .then((res) => {
        if (!res.ok) {
          throw new Error(`failed to load docs.json: ${res.status}`)
        }
        return res.json()
      })
      .then(setDocs)
      .catch((err) => {
        console.error(err)
      })

  }, [])

  function openMarkdown(path) {

    fetch(path)
      .then((res) => {
        if (!res.ok) {
          throw new Error(`failed to load markdown: ${res.status}`)
        }
        return res.text()
      })
      .then((md) => {
        const html = marked.parse(md)
        setMdHtml(html)
        window.scrollTo(0, 0)
      })
      .catch((err) => {
        console.error(err)
      })
  }

  if (!docs) {
    return <div className="loading">Loading...</div>
  }

  const visibleSections = sortSections(
    buildVisibleSections(docs.sections || [], lang)
  )

  return (
    <div className="container">

      <header>

        <h1>{docs.title}</h1>
        <p>{docs.subtitle}</p>

        <div className="toolbar">

          <Search
            sections={visibleSections}
            onResult={setFiltered}
          />

          <div className="lang-toggle">

            <button
              type="button"
              className={lang === "EN" ? "active" : ""}
              onClick={() => {
                setLang("EN")
                setFiltered(null)
              }}
            >
              EN
            </button>

            <button
              type="button"
              className={lang === "JA" ? "active" : ""}
              onClick={() => {
                setLang("JA")
                setFiltered(null)
              }}
            >
              JA
            </button>

          </div>

        </div>

      </header>

      <MarkdownViewer
        html={mdHtml}
        onClose={() => setMdHtml(null)}
      />

      {filtered ? (

        <Section
          title="Search Results"
          items={filtered}
          onOpen={openMarkdown}
        />

      ) : (

        visibleSections.map((section) => (
          <Section
            key={section.title}
            title={section.title}
            items={section.items}
            onOpen={openMarkdown}
          />
        ))

      )}

    </div>
  )
}

// Render the app

ReactDOM
  .createRoot(document.getElementById("root"))
  .render(<App />)
