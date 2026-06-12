import path from "node:path"
import { fileURLToPath } from "node:url"
import { Command } from "commander"

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export const ROOT_DIR = path.resolve(__dirname, "../../..")

// --dry-run を共通で備えた commander Command を生成する（setup スクリプト共通）
export function newSetupCommand(name) {
  return new Command(name).option(
    "--dry-run",
    "実際には書き込まず、変更内容のみ表示する",
    false
  )
}
