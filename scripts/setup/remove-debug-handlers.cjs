const { parseCommonFlags, exitWithUsage } = require("./lib/runtime.cjs")
const { removeTarget, updateFile } = require("./lib/file-utils.cjs")

const DELETE_TARGETS = [
  "internal/controller/handler/debug",
  "openapi/paths/debug",
  "openapi/components/requests/debug",
  "openapi/components/request/debug",
  "openapi/components/responses/debug",
  "openapi/components/respone/degub"
]

const CONTROLLER_MODULE_FILE = "internal/di/module/controller.go"
const OPENAPI_FILE = "openapi/openapi.yaml"
const OPENAPI_README_FILES = ["openapi/README.md", "openapi/README.ja.md"]

function printUsage() {
  console.log(`使用方法:
  node scripts/setup/remove-debug-handlers.cjs [--dry-run]

説明:
  デバッグ用ハンドラと関連する OpenAPI 定義を削除します。
  - internal/controller/handler/debug
  - openapi/paths/debug
  - openapi/components/requests/debug
  - openapi/components/responses/debug
  - internal/di/module/controller.go の debug ハンドラ登録
  - openapi/openapi.yaml の debug path / schema 参照
  - openapi/README.md, openapi/README.ja.md の debug 説明
`)
}

function parseArgs(argv) {
  const options = parseCommonFlags(argv)

  for (const arg of options.rest) {
    throw new Error(`不明な引数です: ${arg}`)
  }

  return options
}

function updateControllerModule(content) {
  const lines = content.split("\n")
  const filtered = lines.filter(line => {
    if (line === '\t"boilerplate-go/internal/controller/handler/debug/cookie"') {
      return false
    }

    if (line.includes("デバッグ用のハンドラー")) {
      return false
    }

    if (line.includes("cookie.BindHandler")) {
      return false
    }

    return true
  })

  return `${filtered.join("\n").replace(/\n{3,}/g, "\n\n")}\n`
}

function updateOpenapi(content) {
  const lines = content.split("\n")
  const filtered = []
  let skipNextLineCount = 0

  for (const line of lines) {
    if (skipNextLineCount > 0) {
      skipNextLineCount -= 1
      continue
    }

    if (line.includes("デバッグ用のパス")) {
      continue
    }

    if (
      line === "  /debug/cookie:" ||
      line === "  /debug/cookie/copy:" ||
      line === "  /debug/cookie/stream:" ||
      line === "  /debug/cookie/ws:"
    ) {
      skipNextLineCount = 1
      continue
    }

    if (line.includes("デバッグ用の型定義")) {
      continue
    }

    if (
      line === "    DebugIssueCookieRequest:" ||
      line === "    DebugCookieInspectResponse:"
    ) {
      skipNextLineCount = 1
      continue
    }

    filtered.push(line)
  }

  return `${filtered.join("\n").replace(/\n{3,}/g, "\n\n")}\n`
}

function updateOpenapiReadme(content) {
  return content
    .replace(/\n## Debug API[\s\S]*$/m, "")
    .replace(/\n## Debug APIについて[\s\S]*$/m, "")
}

function main() {
  let options

  try {
    options = parseArgs(process.argv.slice(2))
  } catch (error) {
    exitWithUsage(error, printUsage)
  }

  if (options.help) {
    printUsage()
    return
  }

  const removedTargets = []
  const updatedFiles = []

  for (const relativePath of DELETE_TARGETS) {
    const result = removeTarget(relativePath, options.dryRun)

    if (result) {
      removedTargets.push(result)
    }
  }

  const controllerResult = updateFile(
    CONTROLLER_MODULE_FILE,
    updateControllerModule,
    options.dryRun
  )

  if (controllerResult) {
    updatedFiles.push(controllerResult)
  }

  const openapiResult = updateFile(
    OPENAPI_FILE,
    updateOpenapi,
    options.dryRun
  )

  if (openapiResult) {
    updatedFiles.push(openapiResult)
  }

  for (const file of OPENAPI_README_FILES) {
    const result = updateFile(file, updateOpenapiReadme, options.dryRun)

    if (result) {
      updatedFiles.push(result)
    }
  }

  if (removedTargets.length === 0 && updatedFiles.length === 0) {
    console.log("削除対象は見つかりませんでした。")
    return
  }

  console.log(`${options.dryRun ? "ドライラン" : "削除完了"}:`)

  if (removedTargets.length > 0) {
    console.log("削除対象:")
    for (const target of removedTargets) {
      console.log(`- ${target}`)
    }
  }

  if (updatedFiles.length > 0) {
    console.log("更新対象:")
    for (const file of updatedFiles) {
      console.log(`- ${file}`)
    }
  }

  console.log("")
  console.log("注意: 実行後は make gen-api を実行して生成物を再作成してください。")
}

main()
