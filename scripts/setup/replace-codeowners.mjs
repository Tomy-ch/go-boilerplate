import fs from "node:fs"
import { newSetupCommand } from "./lib/runtime.mjs"
import { toAbsolutePath, updateFile } from "./lib/file-utils.mjs"
import { ensureCodeOwners } from "./lib/validators.mjs"

const CODEOWNERS_FILE = ".github/CODEOWNERS"

const OWNER_TOKEN = String.raw`(?:@[^\s#]+|[^@\s#]+@[^\s#]+)`

// ルール行は `<パターン><空白><所有者...>[ #コメント]`。空白は列揃えを保つため保持する。
// 所有者の形をしたトークンだけを所有者欄と認める。空白のみを境界にすると、空白を
// エスケープしたパターン(`foo\ bar.txt @o`)や複数語のセクション見出し(`[My Team]`)を
// パターンと所有者の境目で切って壊す。
const RULE_LINE = new RegExp(
  String.raw`^(\S+)([ \t]+)(${OWNER_TOKEN}(?:[ \t]+${OWNER_TOKEN})*)([ \t]*#.*)?[ \t]*$`
)

// ヘッダーが記載例として挙げている所有者を保つため、コメント行は対象外とする。
const COMMENT_LINE = /^[ \t]*#/

// 所有者を持たないパターン行は継承の打ち消しであり、書き換え対象ではない。
const OWNERLESS_LINE = /^[ \t]*\S*[ \t]*$/

// skippedLines には、書き換え対象のはずが所有者を判定できなかった行番号を積む。
// 一括置換の取りこぼしが黙って残ると、レビューを予約したつもりで誰にも予約できて
// いない状態になるため、呼び出し元から必ず表示する。
function updateCodeowners(content, owners, skippedLines) {
  let replaced = 0

  const updated = content
    .split("\n")
    .map((rawLine, index) => {
      // CRLF のファイルで書き換えた行だけ LF になり改行が混在するのを防ぐ。
      const eol = rawLine.endsWith("\r") ? "\r" : ""
      const line = eol === "" ? rawLine : rawLine.slice(0, -1)

      if (COMMENT_LINE.test(line) || OWNERLESS_LINE.test(line)) {
        return rawLine
      }

      const matched = RULE_LINE.exec(line)

      if (!matched) {
        skippedLines.push(index + 1)

        return rawLine
      }

      replaced += 1

      return `${matched[1]}${matched[2]}${owners}${matched[4] ?? ""}${eol}`
    })
    .join("\n")

  if (replaced === 0) {
    throw new Error(`${CODEOWNERS_FILE} に所有者を持つルール行が見つかりませんでした。`)
  }

  return updated
}

function run(owners, dryRun) {
  if (!fs.existsSync(toAbsolutePath(CODEOWNERS_FILE))) {
    console.error(`✖ ${CODEOWNERS_FILE} が見つかりません。`)
    process.exit(1)
  }

  const skippedLines = []
  const result = updateFile(
    CODEOWNERS_FILE,
    content => updateCodeowners(content, owners, skippedLines),
    dryRun
  )

  if (skippedLines.length > 0) {
    console.log(`⚠ 所有者を判定できず未置換の行: ${skippedLines.join(", ")}（手で確認してください）`)
  }

  if (!result) {
    console.log("既に指定の所有者のため変更はありません。")
    return
  }

  console.log(`${dryRun ? "ドライラン" : "置換完了"}: ${CODEOWNERS_FILE}`)
  console.log(`- 所有者: ${owners}`)
  console.log("")
  console.log(
    "注意: 実際にレビューを必須化するにはブランチ保護の「Require review from Code Owners」が別途必要です。"
  )
  console.log(
    "また GitHub 上で解決できない所有者（組織そのもの・存在しない team・未検証メール）は黙って無視されます。"
  )
}

const program = newSetupCommand("replace-codeowners")
program
  .description(".github/CODEOWNERS の全ルールの所有者を一括で置換する")
  .requiredOption(
    "--owners <owners>",
    "所有者（空白区切りで複数指定可。例: '@example-org/tech-leads @example-org/security'）"
  )
  .action(options => {
    const owners = options.owners.trim().split(/\s+/).filter(Boolean)

    try {
      ensureCodeOwners(owners)
    } catch (error) {
      program.error(`エラー: ${error.message}`)
    }

    run(owners.join(" "), options.dryRun)
  })
  .parse()
