import fs from "node:fs"
import path from "node:path"

const MAKEFILES_DIR = ".makefiles"

// .makefiles 配下の *.mk をフルパス昇順で列挙する
function listMkFiles(dir) {
  const files = []

  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const entryPath = path.join(dir, entry.name)

    if (entry.isDirectory()) {
      files.push(...listMkFiles(entryPath))
    } else if (entry.isFile() && entry.name.endsWith(".mk")) {
      files.push(entryPath)
    }
  }

  return files
}

const CATEGORY_RE = /^## (.*)/
const PHONY_WITH_DESC_RE = /^\.PHONY:\s+(\S+)\s*##\s*(.*)$/
const PHONY_RE = /^\.PHONY:/

console.log("📦 Makeターゲット一覧")
console.log("-------------------------------------------")

for (const file of listMkFiles(MAKEFILES_DIR).sort()) {
  const lines = fs.readFileSync(file, "utf8").split("\n")

  for (const line of lines) {
    const category = line.match(CATEGORY_RE)
    if (category) {
      // カテゴリ見出し行
      console.log("")
      console.log(`📂 ${category[1]}`)
      continue
    }

    const phony = line.match(PHONY_WITH_DESC_RE)
    if (phony) {
      // .PHONY 行（単一ターゲット + コメント付き）
      console.log(`🛠  ${phony[1].padEnd(24)} ${phony[2]}`)
      continue
    }

    if (PHONY_RE.test(line)) {
      // 説明（## ...）が無い / 形式不一致の .PHONY 行は help に出ないため警告する
      console.error(`⚠️  ${file}: 説明コメント(## ...)の無い .PHONY 行をスキップしました: ${line}`)
    }
  }
}
