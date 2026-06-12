const path = require("path")
const { Command } = require("commander")

const ROOT_DIR = path.resolve(__dirname, "../../..")

// --dry-run を共通で備えた commander Command を生成する（setup スクリプト共通）。
// 引数/オプションの検証・usage 生成・未知オプション検出は commander に委ねる。
function newSetupCommand(name) {
  return new Command(name).option(
    "--dry-run",
    "実際には書き込まず、変更内容のみ表示する",
    false
  )
}

module.exports = {
  ROOT_DIR,
  newSetupCommand,
}
