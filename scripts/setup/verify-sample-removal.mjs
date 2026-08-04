import fs from "node:fs"
import path from "node:path"
import { execFileSync } from "node:child_process"
import { fileURLToPath } from "node:url"

// サンプル削除が「過不足なく」完了したことを検証し、最後に自身とスナップショットを自爆削除する。
// remove-sample-api.mjs は manifest（sample-api.mjs）ごと自消滅するため、削除確認は manifest ではなく
// remove-sample-api.mjs が書き出した .sample-removal-snapshot.json を唯一の入力として行う（bootstrap 対策）。
// この mjs 自体はサンプル削除ツールの最終地点なので、検証成功後に自消滅してコアのみの状態を残す。

const SELF_PATH = fileURLToPath(import.meta.url)
const SETUP_DIR = path.dirname(SELF_PATH)
const ROOT_DIR = path.resolve(SETUP_DIR, "../..")
const SNAPSHOT_PATH = path.join(SETUP_DIR, ".sample-removal-snapshot.json")

// 残留サンプル参照の検出条件。生成物とテストは CI で regen を省くため除外する。
const DANGLING_PATTERN = "usercount|userpurge|productimagegc|user_roles|prefecture"
const DANGLING_EXCLUDE = "_test\\.go|\\.gen\\.go"

function readRegisteredPaths() {
  if (!fs.existsSync(SNAPSHOT_PATH)) {
    throw new Error(`スナップショットが見つかりません: ${SNAPSHOT_PATH}（remove-sample-api.mjs 未実行の可能性）`)
  }
  const parsed = JSON.parse(fs.readFileSync(SNAPSHOT_PATH, "utf8"))
  if (!Array.isArray(parsed.registeredPaths) || parsed.registeredPaths.length === 0) {
    throw new Error("スナップショットの registeredPaths が空です")
  }
  return parsed.registeredPaths
}

// git status（porcelain）から削除エントリの相対パスを取り出す。
function deletedPathsFromGitStatus() {
  const output = execFileSync("git", ["status", "--porcelain"], { cwd: ROOT_DIR, encoding: "utf8" })
  return output
    .split("\n")
    .filter(line => line.length > 3 && (line[0] === "D" || line[1] === "D"))
    .map(line => line.slice(3))
}

// 不足検出: 登録パスがまだ残っていれば「消えていない」。
function checkNoShortage(registeredPaths) {
  const remaining = registeredPaths.filter(relativePath => fs.existsSync(path.join(ROOT_DIR, relativePath)))
  return remaining.map(relativePath => `未削除の登録パス: ${relativePath}`)
}

// 過剰検出: 登録パスに含まれない削除は想定外（サンプル以外を巻き込んでいる）。
function checkNoExcess(registeredPaths) {
  const isRegistered = deletedPath =>
    registeredPaths.some(reg => deletedPath === reg || deletedPath.startsWith(`${reg}/`))
  return deletedPathsFromGitStatus()
    .filter(deletedPath => !isRegistered(deletedPath))
    .map(deletedPath => `登録外の削除を検出: ${deletedPath}`)
}

// 削除ツール自身の make ターゲットが .mk のマーカー除去で消えていることを確認する。
function checkMakeTargetGone() {
  let help = ""
  try {
    help = execFileSync("make", ["help"], { cwd: ROOT_DIR, encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] })
  } catch (error) {
    help = error.stdout ?? ""
  }
  return help.includes("setup-remove-sample-api")
    ? ["make ターゲット setup-remove-sample-api が残っています"]
    : []
}

// 残留サンプル参照（実コード）を grep で検出する。生成物・テストは除外。
function checkNoDanglingReferences() {
  const script = `grep -rniE '${DANGLING_PATTERN}' internal/ cmd/ --include='*.go' | grep -vE '${DANGLING_EXCLUDE}' || true`
  const hits = execFileSync("bash", ["-c", script], { cwd: ROOT_DIR, encoding: "utf8" }).trim()
  return hits === "" ? [] : [`残留サンプル参照:\n${hits}`]
}

function selfDestruct() {
  fs.rmSync(SNAPSHOT_PATH, { force: true })
  fs.rmSync(SELF_PATH, { force: true })
}

function main() {
  console.log("🔍 サンプル削除の検証を開始します（過不足・残留参照・ツール自消滅）。")

  const registeredPaths = readRegisteredPaths()
  const failures = [
    ...checkNoShortage(registeredPaths),
    ...checkNoExcess(registeredPaths),
    ...checkMakeTargetGone(),
    ...checkNoDanglingReferences(),
  ]

  if (failures.length > 0) {
    console.error("❌ サンプル削除の検証に失敗しました:")
    for (const failure of failures) {
      console.error(`  - ${failure}`)
    }
    process.exit(1)
  }

  selfDestruct()
  console.log("✅ 過不足なく削除・残留なしを確認し、削除確認 mjs も自消滅しました。")
}

try {
  main()
} catch (error) {
  console.error(`❌ 検証エラー: ${error.message}`)
  process.exit(1)
}
