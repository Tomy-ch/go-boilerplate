const fs = require("fs")

// コミットメッセージ先頭に要求する prefix（CLAUDE.md のコミット規約と一致させる）
const PREFIXES = [
  "Feat",
  "Fix",
  "Refactor",
  "Perf",
  "Docs",
  "Test",
  "Build",
  "CI",
  "Chore",
  "Style",
  "Revert"
]

// Merge / Revert の自動生成メッセージは prefix 規約の対象外として許可する
const ALLOWED = new RegExp(`^(Merge |Revert |(${PREFIXES.join("|")})(\\([^)]+\\))?: )`)

function main() {
  const msgFile = process.argv[2]

  if (!msgFile) {
    console.error("コミットメッセージファイルのパスを指定してください。")
    process.exit(1)
  }

  const firstLine = fs.readFileSync(msgFile, "utf8").split("\n")[0]

  if (!ALLOWED.test(firstLine)) {
    console.error(`✖ コミットメッセージ先頭は <Prefix>: で始めてください（${PREFIXES.join("/")}）`)
    console.error(`  受領: ${firstLine}`)
    process.exit(1)
  }
}

main()
