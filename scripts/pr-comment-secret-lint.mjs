// `upsert-pr-comment` を使うジョブに secret が渡っていないかを検査する lint スクリプト。
//
// Actions のシークレットマスキングは、ランナーがジョブ出力をログ表示用に捕捉する経路にしか効かない。
// 検査ログを `tee` でファイルへ落としたバイトは素通りするため、そのファイルを本文にする
// `upsert-pr-comment` では、ログ上はマスク済みに見える値でも生のまま PR コメントに載る。
// マスキングを当てにできない以上、「本文を作るジョブに secret を渡さない」を規約として守るしかなく、
// この検査はその規約が将来 `env:` 1 行で破られることへの退行ガードにあたる。
//
// GITHUB_TOKEN はコメント投稿そのものに必要で、かつ Actions が発行する短命トークンなので許可する。
//
// 検出できるのは `${{ }}` 式に現れる secrets コンテキストの直接参照（`secrets.NAME` /
// `secrets['NAME']` / `toJSON(secrets)` のようなコンテキスト全体）に限る。別ジョブで secret を
// 読んで `needs.<job>.outputs` 経由で渡す間接参照は静的には追えないので、この検査は通る。
//
// 判断は含めず、ワークフロー定義から機械的に導けるものだけを見る。node の標準ライブラリのみに依存し、
// ホストでもコンテナでも動く。1 件でも違反があれば非 0 で終了する。
import fs from "node:fs"
import path from "node:path"

const REPO_ROOT = process.cwd()
const WORKFLOWS_DIR = ".github/workflows"
const COMMENT_ACTION = "./.github/actions/upsert-pr-comment"
const ALLOWED_SECRET = "GITHUB_TOKEN"

// 行末コメントや引用符といった書式差で検出が外れると、規約が破られた瞬間に検査が沈黙する。
const JOBS_KEY = /^jobs:\s*(#.*)?$/
const JOB_HEADER = /^ {2}(?:"([A-Za-z0-9_-]+)"|'([A-Za-z0-9_-]+)'|([A-Za-z0-9_-]+)):\s*(#.*)?$/
const COMMENT_ACTION_USE = new RegExp(
  `uses:\\s*["']?${COMMENT_ACTION.replace(/[.\/]/g, "\\$&")}["']?\\s*(#.*)?$`,
)
const EXPRESSION = /\$\{\{([\s\S]*?)\}\}/g
const SECRET_REFERENCE = /\bsecrets\b\s*(?:\.\s*([A-Za-z0-9_-]+)|\[\s*["']([A-Za-z0-9_-]+)["']\s*\])?/g

const findings = []

function report(file, line, message) {
  findings.push({ file, line, message })
}

// ジョブ単位に切り出す。secret を渡すのがどのステップであれ、本文ファイルを作るステップと
// 同じジョブにいれば届き得るため、ステップ単位ではなくジョブ単位で見る。
function splitJobs(source) {
  const lines = source.split("\n")
  const jobsIndex = lines.findIndex((line) => JOBS_KEY.test(line))
  if (jobsIndex === -1) return { jobs: [], preamble: lines.map(toEntry), found: false }

  const jobs = []
  let current = null
  let end = lines.length

  for (let i = jobsIndex + 1; i < lines.length; i++) {
    const line = lines[i]
    // 桁 0 のコメントはトップレベルキーではないので、ここで jobs: を打ち切らせない。
    if (/^\S/.test(line) && !line.startsWith("#")) {
      end = i
      break
    }

    const header = line.match(JOB_HEADER)
    if (header) {
      current = {
        id: header[1] ?? header[2] ?? header[3],
        lines: [],
      }
      jobs.push(current)
      continue
    }
    if (current) current.lines.push({ number: i + 1, text: line })
  }

  // `jobs:` より前の top-level `env:` に束ねた secret はジョブ本文に現れないため、別途見る。
  const preamble = [
    ...lines.slice(0, jobsIndex).map((text, i) => ({ number: i + 1, text })),
    ...lines.slice(end).map((text, i) => ({ number: end + i + 1, text })),
  ]

  return { jobs, preamble, found: true }
}

function toEntry(text, i) {
  return { number: i + 1, text }
}

function usesCommentAction(job) {
  return job.lines.some(({ text }) => COMMENT_ACTION_USE.test(text))
}

// `${{ }}` 式の中だけを見る。散文中の「secrets」に反応させないため。
function secretReferences({ number, text }) {
  const referenced = []
  for (const expression of text.matchAll(EXPRESSION)) {
    for (const reference of expression[1].matchAll(SECRET_REFERENCE)) {
      const name = reference[1] ?? reference[2]
      if (name === ALLOWED_SECRET) continue
      referenced.push({ number, name })
    }
  }
  return referenced
}

function listWorkflowFiles() {
  const dir = path.join(REPO_ROOT, WORKFLOWS_DIR)
  if (!fs.existsSync(dir)) return []
  return fs
    .readdirSync(dir)
    .filter((name) => name.endsWith(".yaml") || name.endsWith(".yml"))
    .sort()
    .map((name) => path.join(WORKFLOWS_DIR, name))
}

function describe(name) {
  return name ? `\`secrets.${name}\`` : "`secrets` コンテキスト全体"
}

const workflowFiles = listWorkflowFiles()

// 検査対象 0 件は「問題なし」ではなく「検査が働いていない」。実行位置の誤りを成功と report しない。
if (workflowFiles.length === 0) {
  console.error(
    `✘ pr-comment-secret-lint: ${WORKFLOWS_DIR}/ にワークフローが見つかりません（リポジトリルートで実行してください）`,
  )
  process.exit(2)
}

let checkedJobs = 0

for (const rel of workflowFiles) {
  const source = fs.readFileSync(path.join(REPO_ROOT, rel), "utf8")
  const { jobs, preamble, found } = splitJobs(source)

  if (!found) {
    console.error(`✘ pr-comment-secret-lint: ${rel} に jobs: が見つかりません`)
    process.exit(2)
  }

  const commenting = jobs.filter(usesCommentAction)
  checkedJobs += commenting.length
  if (commenting.length === 0) continue

  for (const job of commenting) {
    for (const line of job.lines) {
      for (const { number, name } of secretReferences(line)) {
        report(
          rel,
          number,
          `ジョブ \`${job.id}\` は ${COMMENT_ACTION} を使うため ${describe(name)} を渡せません（マスキングは tee したファイルに効かず、生値が PR コメントに載ります）`,
        )
      }
    }
  }

  for (const line of preamble) {
    for (const { number, name } of secretReferences(line)) {
      report(
        rel,
        number,
        `ワークフロー全体に及ぶ ${describe(name)} は ${COMMENT_ACTION} を使うジョブにも届きます（マスキングは tee したファイルに効かず、生値が PR コメントに載ります）`,
      )
    }
  }
}

if (findings.length > 0) {
  console.error(`✘ pr-comment-secret-lint: ${findings.length} 件の違反\n`)
  let current = null
  for (const finding of findings) {
    if (finding.file !== current) {
      if (current !== null) console.error("")
      console.error(`  ${finding.file}`)
      current = finding.file
    }
    console.error(`    :${finding.line}  ${finding.message}`)
  }
  console.error(
    `\n検査 ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブ中 ${findings.length} 件 NG`,
  )
  process.exit(1)
}

console.log(
  `✓ pr-comment-secret-lint: ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブすべて OK`,
)
