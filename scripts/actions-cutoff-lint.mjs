// ジョブが打ち切られたときの振る舞いが定義されているかを検査する lint スクリプト。
//
// 打ち切り（タイムアウト / キャンセル / ランナー障害）は 2 つの欠落が組み合わさって不可視になる。
// 片方が状況を作り、片方が状況を見えなくするため、両面を 1 つの検査として見る。
//
//   A. `upsert-pr-comment` を呼ぶステップの `if:` が cancelled に到達しない
//      Actions はステータスチェック関数を含まないカスタム `if:` に暗黙で `success() &&` を前置する。
//      打ち切られたジョブではこのステップがスキップされ、PR には何のコメントも残らない。
//      失敗ステップ側は `always()` を持つことが多いので、赤くはなるが理由が読めない状態になる。
//
//   B. ジョブに `timeout-minutes:` が無い
//      GitHub 既定の 360 分まで走り続け、ハングした 1 ジョブがランナーを 6 時間占有しうる。
//
// A の合格条件を `always()` / `cancelled()` に限り `failure()` を含めないのは、`failure()` が
// cancelled では false になるためである。「ステータス関数を持つか」で書くと、関数はあるのに
// 打ち切り時は沈黙するステップを取り逃がす。述語は cancelled 到達性で定義する。
//
// 静的に読めるものだけを見る。`!always()` のように到達性を打ち消す式は書けてしまうが、規約が正で
// この検査はその近似にあたる。node の標準ライブラリのみに依存し、ホストでもコンテナでも動く。
// 1 件でも違反があれば非 0 で終了する。
import fs from "node:fs"
import path from "node:path"

const REPO_ROOT = process.cwd()
const WORKFLOWS_DIR = ".github/workflows"
const COMMENT_ACTION = "./.github/actions/upsert-pr-comment"

const JOBS_KEY = /^jobs:\s*(#.*)?$/
const JOB_HEADER = /^ {2}(?:"([A-Za-z0-9_-]+)"|'([A-Za-z0-9_-]+)'|([A-Za-z0-9_-]+)):\s*(#.*)?$/
// ジョブ直下のキーだけを見る。ステップの `uses:` / `if:` を拾わないよう桁で絞る。
const JOB_LEVEL_USES = /^ {4}uses:\s*\S/
const JOB_LEVEL_TIMEOUT = /^ {4}timeout-minutes:\s*\S/
const STEPS_KEY = /^ {4}steps:\s*(#.*)?$/
const STEP_ITEM = /^ {6}- /
// ステップのキーは列 8 に並ぶ。先頭キーだけは `      - ` に続けて同じ列から始まる。
const STEP_KEY_IF = /^(?: {6}- | {8})if:\s*(.*)$/
const COMMENT_ACTION_USE = new RegExp(
  `^(?: {6}- | {8})uses:\\s*["']?${COMMENT_ACTION.replace(/[.\/]/g, "\\$&")}["']?\\s*(#.*)?$`,
)
// 打ち切りで到達できるのはこの 2 つだけ。`failure()` は cancelled では false になるので数えない。
const REACHES_CANCELLED = /\b(?:always|cancelled)\s*\(\s*\)/

const findings = []

function report(file, line, message) {
  findings.push({ file, line, message })
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

function splitJobs(source) {
  const lines = source.split("\n")
  const jobsIndex = lines.findIndex((line) => JOBS_KEY.test(line))
  if (jobsIndex === -1) return null

  const jobs = []
  let current = null

  for (let i = jobsIndex + 1; i < lines.length; i++) {
    const line = lines[i]
    // 桁 0 のコメントはトップレベルキーではないので、ここで jobs: を打ち切らせない。
    if (/^\S/.test(line) && !line.startsWith("#")) break

    const header = line.match(JOB_HEADER)
    if (header) {
      current = {
        id: header[1] ?? header[2] ?? header[3],
        number: i + 1,
        lines: [],
      }
      jobs.push(current)
      continue
    }
    if (current) current.lines.push({ number: i + 1, text: line })
  }

  return jobs
}

// ステップ単位に切り出す。`if:` はそのステップのものだけを見る必要があるため、ジョブ本文を
// `- ` 始まりで区切る。
function splitSteps(job) {
  const start = job.lines.findIndex(({ text }) => STEPS_KEY.test(text))
  if (start === -1) return []

  const steps = []
  let current = null

  for (const line of job.lines.slice(start + 1)) {
    if (STEP_ITEM.test(line.text)) {
      current = { number: line.number, lines: [] }
      steps.push(current)
    }
    // steps: と同じかそれより浅い桁のキーが来たらステップ列の終わり。
    else if (/^ {0,4}\S/.test(line.text) && !/^\s*#/.test(line.text)) break

    if (current) current.lines.push(line)
  }

  return steps
}

// `if: >-` のような折り畳みスカラーでは値が次行以降にある。ステップ内の後続の深い行を続きとみなす。
function conditionOf(step) {
  const index = step.lines.findIndex(({ text }) => STEP_KEY_IF.test(text))
  if (index === -1) return null

  const head = step.lines[index].text.match(STEP_KEY_IF)[1].trim()
  const parts = head === "" || head === ">" || head === ">-" || head === "|" || head === "|-" ? [] : [head]

  for (const { text } of step.lines.slice(index + 1)) {
    if (!/^ {9,}\S/.test(text)) break
    parts.push(text.trim())
  }

  return { line: step.lines[index].number, value: parts.join(" ") }
}

const workflowFiles = listWorkflowFiles()

// 検査対象 0 件は「問題なし」ではなく「検査が働いていない」。実行位置の誤りを成功と report しない。
if (workflowFiles.length === 0) {
  console.error(
    `✘ actions-cutoff-lint: ${WORKFLOWS_DIR}/ にワークフローが見つかりません（リポジトリルートで実行してください）`,
  )
  process.exit(2)
}

let checkedJobs = 0
let checkedSteps = 0

for (const rel of workflowFiles) {
  const source = fs.readFileSync(path.join(REPO_ROOT, rel), "utf8")
  const jobs = splitJobs(source)

  if (jobs === null) {
    console.error(`✘ actions-cutoff-lint: ${rel} に jobs: が見つかりません`)
    process.exit(2)
  }

  for (const job of jobs) {
    // reusable workflow を呼ぶジョブは `timeout-minutes` を書けない（invalid key）。
    const callsReusable = job.lines.some(({ text }) => JOB_LEVEL_USES.test(text))
    if (!callsReusable) {
      checkedJobs += 1
      if (!job.lines.some(({ text }) => JOB_LEVEL_TIMEOUT.test(text))) {
        report(
          rel,
          job.number,
          `ジョブ \`${job.id}\` に timeout-minutes がありません（GitHub 既定の 360 分まで走ります）`,
        )
      }
    }

    for (const step of splitSteps(job)) {
      if (!step.lines.some(({ text }) => COMMENT_ACTION_USE.test(text))) continue

      checkedSteps += 1
      const condition = conditionOf(step)

      if (condition === null) {
        report(
          rel,
          step.number,
          `ジョブ \`${job.id}\` の ${COMMENT_ACTION} ステップに if: がありません（暗黙の success() で打ち切り時にスキップされます）`,
        )
        continue
      }
      if (!REACHES_CANCELLED.test(condition.value)) {
        report(
          rel,
          condition.line,
          `ジョブ \`${job.id}\` の ${COMMENT_ACTION} ステップの if: が打ち切りに到達しません（always() / cancelled() が要ります。failure() は cancelled では false です）`,
        )
      }
    }
  }
}

if (findings.length > 0) {
  console.error(`✘ actions-cutoff-lint: ${findings.length} 件の違反\n`)
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
    `\n検査 ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブ / ${checkedSteps} コメントステップ中 ${findings.length} 件 NG`,
  )
  process.exit(1)
}

console.log(
  `✓ actions-cutoff-lint: ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブ / ${checkedSteps} コメントステップすべて OK`,
)
