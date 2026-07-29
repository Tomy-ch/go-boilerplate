// `.claude/**` のスキル / エージェント定義を意味的に検査する lint スクリプト。
// markdownlint は体裁しか見ないため、「書いてある内容が実態と合っているか」は誰も検査していない。
// スキル定義はエージェントの挙動を決める指示書であり、腐った参照はそのまま誤った手順の実行につながる。
//
// 検査は Makefile のターゲット一覧・ファイルシステム・見出し抽出から導出できるものだけに限る
// （判断を含めない）。node の標準ライブラリのみに依存し、ホストでもコンテナでも動く。
// 1 件でも違反があれば非 0 で終了する。
import fs from "node:fs"
import path from "node:path"

const REPO_ROOT = process.cwd()
const CLAUDE_DIR = ".claude"
const SKILLS_DIR = path.join(CLAUDE_DIR, "skills")
const AGENTS_DIR = path.join(CLAUDE_DIR, "agents")

// ファイル索引・参照検査から外すディレクトリ（生成物 / 実行時成果物 / 外部由来）。
const EXCLUDE_DIRS = new Set([".git", "node_modules", "vendor", "tmp"])

// 参照検査の対象外にする先頭セグメント。tmp/ 配下はスキル実行中に生成されるため、
// 静的なファイルシステム検査では存在しないのが正常。
const PATH_ROOT_DENY = new Set(["tmp", ".git"])

// 意図的に実在しない参照（仮定の例示・任意配置）を抑止するための行内ディレクティブ。
const IGNORE_DIRECTIVE = "<!-- skill-lint-ignore -->"

const findings = []

function report(file, line, rule, message) {
  findings.push({ file, line, rule, message })
}

// ---------------------------------------------------------------------------
// 共通ユーティリティ
// ---------------------------------------------------------------------------

function readFile(rel) {
  return fs.readFileSync(path.join(REPO_ROOT, rel), "utf8")
}

function listDirs(rel) {
  const abs = path.join(REPO_ROOT, rel)
  if (!fs.existsSync(abs)) return []
  return fs
    .readdirSync(abs, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => e.name)
    .sort()
}

// repoRoot 配下の全エントリ（ファイル + ディレクトリ）をリポジトリ相対パスで索引化する。
// glob / プレースホルダを含む参照は実パス解決ができないため、この索引に対する正規表現照合で判定する。
function buildEntryIndex() {
  const entries = []
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name)
      const rel = path.relative(REPO_ROOT, abs)
      entries.push(rel)
      if (entry.isDirectory()) {
        if (EXCLUDE_DIRS.has(entry.name)) continue
        walk(abs)
      }
    }
  }
  walk(REPO_ROOT)
  return entries
}

// `{a,b}` を展開して候補文字列の配列にする。
// Make ターゲットでは実ターゲット名の列挙（全て実在すべき）、パスでは glob の選択（どれか 1 つ
// 当たれば良い）と意味が異なるため、判定側で all / any を使い分ける。
function expandBraces(text) {
  const m = /\{([^{}]*)\}/.exec(text)
  if (!m) return [text]
  const expanded = m[1].split(",").flatMap((alt) => expandBraces(text.slice(0, m.index) + alt + text.slice(m.index + m[0].length)))
  return expanded
}

const WILDCARD_RE = /[*<]/

// ドキュメント中のプレースホルダ表記を正規表現へ変換する。
// `<name>` は書き手が埋める任意の 1 セグメント、`**` は任意階層、`*` は 1 セグメント内の任意文字列。
function placeholderToRegExp(text, { segmentSeparator }) {
  const anySegmentChars = segmentSeparator ? "[^/]*" : ".*"
  const placeholderChars = segmentSeparator ? "[^/]+" : ".+"
  let out = ""
  for (let i = 0; i < text.length; i++) {
    const ch = text[i]
    if (ch === "<") {
      const close = text.indexOf(">", i)
      if (close === -1) {
        out += "<"
        continue
      }
      out += placeholderChars
      i = close
      continue
    }
    if (ch === "*") {
      if (segmentSeparator && text.slice(i, i + 3) === "**/") {
        out += "(?:[^/]+/)*"
        i += 2
        continue
      }
      if (segmentSeparator && text.slice(i, i + 2) === "**") {
        out += ".*"
        i += 1
        continue
      }
      out += anySegmentChars
      continue
    }
    out += ch.replace(/[.+^${}()|[\]\\?]/g, "\\$&")
  }
  return new RegExp(`^${out}$`)
}

// 行を走査しつつコードフェンス（``` / ~~~）の内外を判定する。
// フェンス内は例示・出力サンプルであり実在性を保証しない前提のため、検査対象から外す。
// スキル本文は Markdown を含む Markdown（```markdown の中に ```json）を書くため、閉じ判定は
// CommonMark どおり「情報文字列を持たない同種・同長以上のフェンス行」に限る。
function* eachLineOutsideFence(content) {
  const lines = content.split("\n")
  let fence = null
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const marker = /^\s*(`{3,}|~{3,})(.*)$/.exec(line)
    if (fence) {
      const closes = marker && marker[1][0] === fence[0] && marker[1].length >= fence.length && marker[2].trim() === ""
      if (closes) fence = null
      continue
    }
    if (marker) {
      fence = marker[1]
      continue
    }
    yield { line, lineNo: i + 1 }
  }
}

// 1 行からインラインコードスパン（`...`）の中身を抜き出す。
function extractInlineCode(line) {
  const spans = []
  const re = /(`+)([^`]+?)\1/g
  let m
  while ((m = re.exec(line)) !== null) spans.push(m[2].trim())
  return spans
}

// ---------------------------------------------------------------------------
// frontmatter
// ---------------------------------------------------------------------------

// 先頭の `---` で囲まれた frontmatter を切り出す。無ければ null。
function splitFrontmatter(content) {
  const lines = content.split("\n")
  if (lines[0] !== "---") return null
  for (let i = 1; i < lines.length; i++) {
    if (lines[i] === "---") return { lines: lines.slice(1, i), endLine: i + 1 }
  }
  return null
}

// frontmatter のトップレベルキーと値を取り出す。折り畳みスカラ（`key: >-`）は後続の
// インデント行を連結して値とする（YAML パーサを持ち込まずに済む範囲に限定した簡易解析）。
function parseFrontmatterKeys(fmLines) {
  const keys = new Map()
  for (let i = 0; i < fmLines.length; i++) {
    const m = /^([A-Za-z0-9_-]+):\s*(.*)$/.exec(fmLines[i])
    if (!m) continue
    let value = m[2].trim()
    if (value === ">-" || value === ">" || value === "|" || value === "|-") {
      const folded = []
      for (let j = i + 1; j < fmLines.length; j++) {
        if (fmLines[j].trim() !== "" && !/^\s/.test(fmLines[j])) break
        folded.push(fmLines[j].trim())
      }
      value = folded.join(" ").trim()
    }
    keys.set(m[1], value)
  }
  return keys
}

// name / description の必須検査とディレクトリ（ファイル）名との一致検査。
function checkFrontmatter(rel, content, expectedName) {
  const fm = splitFrontmatter(content)
  if (!fm) {
    report(rel, 1, "frontmatter", "frontmatter (`---` で囲まれたブロック) がありません")
    return
  }
  const keys = parseFrontmatterKeys(fm.lines)
  for (const required of ["name", "description"]) {
    if (!keys.has(required) || keys.get(required) === "") {
      report(rel, 1, "frontmatter", `frontmatter に \`${required}\` がありません（または空です）`)
    }
  }
  const name = keys.get("name")
  if (name !== undefined && name !== "" && name !== expectedName) {
    report(rel, 1, "frontmatter", `frontmatter の \`name: ${name}\` が配置名 \`${expectedName}\` と一致しません`)
  }
}

// ---------------------------------------------------------------------------
// 対訳ペア
// ---------------------------------------------------------------------------

// フェンス外の見出しを (レベル, テキスト) で抽出する。
function extractHeadings(content) {
  const headings = []
  for (const { line, lineNo } of eachLineOutsideFence(content)) {
    const m = /^(#{1,6})\s+(.*?)\s*$/.exec(line)
    if (m) headings.push({ level: m[1].length, text: m[2], lineNo })
  }
  return headings
}

// 対訳（SKILL.ja.md）が canonical（SKILL.md）と 1:1 であることを検査する。
// ファイルの有無だけでは節の欠落・ずれを検出できないため、見出しレベル列の一致まで見る。
function checkTranslationPair(canonicalRel, translationRel) {
  if (!fs.existsSync(path.join(REPO_ROOT, translationRel))) {
    report(canonicalRel, 1, "translation", `対訳 \`${path.basename(translationRel)}\` がありません`)
    return
  }
  const translation = readFile(translationRel)

  if (splitFrontmatter(translation)) {
    report(translationRel, 1, "translation", "対訳に frontmatter があります（スキルとして読み込まれるのは canonical 側だけです）")
  }

  const firstLine = translation.split("\n").find((l) => l.trim() !== "") ?? ""
  if (!firstLine.startsWith(">") || !firstLine.includes(path.basename(canonicalRel))) {
    report(translationRel, 1, "translation", `冒頭に canonical (\`${path.basename(canonicalRel)}\`) を指す翻訳注記（引用行）がありません`)
  }

  const canonicalHeadings = extractHeadings(readFile(canonicalRel))
  const translationHeadings = extractHeadings(translation)
  const max = Math.max(canonicalHeadings.length, translationHeadings.length)
  for (let i = 0; i < max; i++) {
    const en = canonicalHeadings[i]
    const ja = translationHeadings[i]
    if (en && ja && en.level === ja.level) continue
    const enDesc = en ? `L${en.lineNo} ${"#".repeat(en.level)} ${en.text}` : "（無し）"
    const jaDesc = ja ? `L${ja.lineNo} ${"#".repeat(ja.level)} ${ja.text}` : "（無し）"
    report(
      translationRel,
      ja ? ja.lineNo : 1,
      "translation",
      `見出し構造が canonical とずれています（${i + 1} 番目 / canonical ${canonicalHeadings.length} 見出し・対訳 ${translationHeadings.length} 見出し）\n` +
        `      canonical: ${enDesc}\n` +
        `      対訳:      ${jaDesc}`,
    )
    return
  }
}

// ---------------------------------------------------------------------------
// 参照: make ターゲット
// ---------------------------------------------------------------------------

// Makefile / .makefiles/**/*.mk からターゲット名を集める。
// `%` を含むパターンルール（`db-migrate-up-%` など）は正規表現として保持する。
function collectMakeTargets() {
  const files = []
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name)
      if (entry.isDirectory()) {
        walk(abs)
        continue
      }
      if (entry.name.endsWith(".mk")) files.push(abs)
    }
  }
  const makefilesDir = path.join(REPO_ROOT, ".makefiles")
  if (fs.existsSync(makefilesDir)) walk(makefilesDir)
  // ルートの makefile は綴りが処理系依存（`makefile` / `Makefile`）。macOS は大文字小文字を
  // 区別しないため、Linux コンテナで初めて読み落とすことがないよう実エントリ名で拾う。
  for (const name of fs.readdirSync(REPO_ROOT)) {
    if (name === "makefile" || name === "Makefile" || name === "GNUmakefile") files.push(path.join(REPO_ROOT, name))
  }

  const exact = new Set()
  const patterns = []
  const addTarget = (name) => {
    if (name === "" || name.startsWith(".")) return
    if (name.includes("%")) {
      patterns.push(new RegExp(`^${name.split("%").map((p) => p.replace(/[.*+^${}()|[\]\\?]/g, "\\$&")).join(".+")}$`))
      return
    }
    exact.add(name)
  }

  for (const file of files) {
    for (const line of fs.readFileSync(file, "utf8").split("\n")) {
      if (line.startsWith("\t")) continue
      const phony = /^\.PHONY:\s*(.+)$/.exec(line)
      if (phony) {
        for (const name of phony[1].split("##")[0].trim().split(/\s+/)) addTarget(name)
        continue
      }
      const rule = /^([A-Za-z0-9_%.+/ -]+):(?!=)/.exec(line)
      if (rule) {
        for (const name of rule[1].trim().split(/\s+/)) addTarget(name)
      }
    }
  }
  return { exact, patterns }
}

const makeTargets = collectMakeTargets()

function makeTargetExists(target) {
  return expandBraces(target).every((candidate) => {
    if (makeTargets.exact.has(candidate)) return true
    if (makeTargets.patterns.some((re) => re.test(candidate))) return true
    if (!WILDCARD_RE.test(candidate)) return false
    // 参照側がプレースホルダ（`gen-*-oapi` / `new-migrate-<name>`）の場合は、
    // それに当てはまる実ターゲットが 1 つでもあれば実在と見なす。
    const re = placeholderToRegExp(candidate, { segmentSeparator: false })
    return [...makeTargets.exact].some((t) => re.test(t)) || makeTargets.patterns.some((p) => re.test(p.source.replace(/[$^\\]/g, "")))
  })
}

// インラインコードの `make ...` からターゲット名を取り出す。
// 変数代入（`DB=local`）やシェル演算子（`2>&1` / `|`）以降は make の引数ではないため打ち切る。
function extractMakeTargets(span) {
  if (!/^make(\s|$)/.test(span)) return []
  const tokens = span.split(/\s+/).slice(1)
  const targets = []
  for (const token of tokens) {
    if (token.startsWith("-")) continue
    if (!/^[A-Za-z0-9_%.<>{},*/-]+$/.test(token)) break
    targets.push(token)
  }
  return targets
}

// ---------------------------------------------------------------------------
// 参照: ファイルパス
// ---------------------------------------------------------------------------

const entryIndex = buildEntryIndex()
const rootEntries = new Set(fs.readdirSync(REPO_ROOT).filter((name) => !PATH_ROOT_DENY.has(name)))
const basenameIndex = new Set(entryIndex.map((entry) => path.basename(entry)))

// ディレクトリを伴わない設定ファイル名（`mise.toml` / `tools.yaml`）の実在性を判定する。
// 設定ファイルはリポジトリ内で名前が一意に定まるため、配置を書かずに名前だけで参照されることが多く、
// SSOT が移動・改名しても本文だけが古い名前で残りやすい。
const CONFIG_FILE_RE = /^[.\w][\w.-]*\.(ya?ml|toml|json)$/

function configFileExists(name) {
  return expandBraces(name).some((candidate) => {
    if (!WILDCARD_RE.test(candidate)) return basenameIndex.has(candidate)
    const re = placeholderToRegExp(candidate, { segmentSeparator: true })
    return [...basenameIndex].some((base) => re.test(base))
  })
}

// インラインコードが検査可能なパス参照かどうかを判定する。
// 相対ファイル名（`SKILL.md` など文脈依存の記述）は解決先が一意に決まらないため対象外にし、
// 先頭セグメントが実在するルート直下エントリであるものだけを検査する。
// さらに、パスと同形だが実体がファイルではない記述を次の規則で除外する:
//   - 末尾セグメントに `.` も末尾 `/` も無いもの — Go の import パス（`database/sql`）と区別できない
//   - `...` を含むもの — 「以下同様」を表す省略記法
//   - `pkg/ptr.Copy` 形式 — パッケージパス + Go シンボル
function asRepoPath(span) {
  let text = span.trim()
  if (text.startsWith("./")) text = text.slice(2)
  if (!text.includes("/")) return null
  if (/[\s$\\#?!"'()|`:;@]/.test(text)) return null
  if (text.includes("...")) return null
  const isDirRef = text.endsWith("/")
  if (isDirRef) text = text.slice(0, -1)
  const head = text.split("/")[0]
  if (!rootEntries.has(head)) return null
  const base = path.basename(text)
  if (!isDirRef && !base.includes(".")) return null
  const symbol = /^(.*)\.[A-Z][A-Za-z0-9_]*$/.exec(text)
  if (symbol && fs.existsSync(path.join(REPO_ROOT, symbol[1]))) return null
  return text
}

// パス参照の実在性を判定する。スキルは自身が同梱するファイル（`scripts/run.sh` など）も
// 同じ表記で参照するため、リポジトリルート相対に加えて参照元ファイルのディレクトリ相対でも解決する。
function repoPathExists(candidate, fromDir) {
  const bases = [REPO_ROOT, path.join(REPO_ROOT, fromDir)]
  return expandBraces(candidate).some((text) => {
    if (!WILDCARD_RE.test(text)) return bases.some((base) => fs.existsSync(path.join(base, text)))
    const re = placeholderToRegExp(text, { segmentSeparator: true })
    return entryIndex.some((entry) => re.test(entry))
  })
}

// ---------------------------------------------------------------------------
// 実行
// ---------------------------------------------------------------------------

// `.claude/**` の Markdown 本文が参照する make ターゲット / ファイルパスの実在性を検査する。
function checkReferences(rel) {
  const content = readFile(rel)
  const fromDir = path.dirname(rel)
  for (const { line, lineNo } of eachLineOutsideFence(content)) {
    if (line.includes(IGNORE_DIRECTIVE)) continue
    for (const span of extractInlineCode(line)) {
      for (const target of extractMakeTargets(span)) {
        if (!makeTargetExists(target)) {
          report(rel, lineNo, "make-ref", `存在しない make ターゲットを参照しています: \`make ${target}\``)
        }
      }
      const repoPath = asRepoPath(span)
      if (repoPath !== null && !repoPathExists(repoPath, fromDir)) {
        report(rel, lineNo, "path-ref", `存在しないパスを参照しています: \`${span}\``)
        continue
      }
      if (repoPath === null && CONFIG_FILE_RE.test(span) && !configFileExists(span)) {
        report(rel, lineNo, "path-ref", `リポジトリに存在しない設定ファイルを参照しています: \`${span}\``)
      }
    }
  }
}

function collectClaudeMarkdown() {
  const out = []
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name)
      if (entry.isDirectory()) {
        if (EXCLUDE_DIRS.has(entry.name)) continue
        walk(abs)
        continue
      }
      if (entry.name.endsWith(".md")) out.push(path.relative(REPO_ROOT, abs))
    }
  }
  const abs = path.join(REPO_ROOT, CLAUDE_DIR)
  if (!fs.existsSync(abs)) {
    console.error(`✘ skill-lint: ${CLAUDE_DIR}/ が見つかりません（リポジトリルートで実行してください）`)
    process.exit(2)
  }
  walk(abs)
  return out.sort()
}

const skillDirs = listDirs(SKILLS_DIR)
for (const name of skillDirs) {
  const canonicalRel = path.join(SKILLS_DIR, name, "SKILL.md")
  if (!fs.existsSync(path.join(REPO_ROOT, canonicalRel))) {
    report(path.join(SKILLS_DIR, name), 1, "structure", "`SKILL.md` がありません")
    continue
  }
  checkFrontmatter(canonicalRel, readFile(canonicalRel), name)
  checkTranslationPair(canonicalRel, path.join(SKILLS_DIR, name, "SKILL.ja.md"))
}

const agentFiles = fs.existsSync(path.join(REPO_ROOT, AGENTS_DIR))
  ? fs
      .readdirSync(path.join(REPO_ROOT, AGENTS_DIR))
      .filter((name) => name.endsWith(".md") && !name.endsWith(".ja.md"))
      .sort()
  : []
for (const file of agentFiles) {
  const rel = path.join(AGENTS_DIR, file)
  checkFrontmatter(rel, readFile(rel), file.replace(/\.md$/, ""))
}

const markdownFiles = collectClaudeMarkdown()
for (const rel of markdownFiles) checkReferences(rel)

if (findings.length > 0) {
  console.error(`✘ skill-lint: ${findings.length} 件の違反\n`)
  let current = null
  for (const finding of findings.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line)) {
    if (finding.file !== current) {
      if (current !== null) console.error("")
      console.error(`  ${finding.file}`)
      current = finding.file
    }
    console.error(`    :${finding.line}  [${finding.rule}] ${finding.message}`)
  }
  console.error(`\n検査 ${skillDirs.length} スキル / ${agentFiles.length} エージェント / ${markdownFiles.length} Markdown 中 ${findings.length} 件 NG`)
  process.exit(1)
}

console.log(`✓ skill-lint: ${skillDirs.length} スキル / ${agentFiles.length} エージェント / ${markdownFiles.length} Markdown すべて OK`)
