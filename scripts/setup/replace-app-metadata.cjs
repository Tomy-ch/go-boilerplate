const { parseCommonFlags, exitWithUsage } = require("./lib/runtime.cjs")
const {
  listChildFiles,
  updateAbsoluteFile,
  updateFile
} = require("./lib/file-utils.cjs")

const ENV_DIR = "env"
const OPENAPI_FILE = "openapi/openapi.yaml"
const COPILOT_INSTRUCTIONS_FILE = ".github/copilot-instructions.md"

function printUsage() {
  console.log(`使用方法:
  node scripts/setup/replace-app-metadata.cjs --app-name <name> --openapi-title <title> --copilot-title <title> [--dry-run]

例:
  node scripts/setup/replace-app-metadata.cjs \\
    --app-name "Example API" \\
    --openapi-title "Example API with Onion Architecture" \\
    --copilot-title "example-api Copilot Instructions"
`)
}

function parseArgs(argv) {
  const options = parseCommonFlags(argv)
  const args = options.rest

  for (let i = 0; i < args.length; i += 1) {
    const arg = args[i]

    if (arg === "--app-name" || arg === "--openapi-title" || arg === "--copilot-title") {
      const value = args[i + 1]

      if (!value || value.startsWith("--")) {
        throw new Error(`${arg} の値を指定してください。`)
      }

      if (arg === "--app-name") {
        options.appName = value
      }

      if (arg === "--openapi-title") {
        options.openapiTitle = value
      }

      if (arg === "--copilot-title") {
        options.copilotTitle = value
      }

      i += 1
      continue
    }

    throw new Error(`不明な引数です: ${arg}`)
  }

  if (options.help) {
    return options
  }

  if (!options.appName || !options.openapiTitle || !options.copilotTitle) {
    throw new Error("--app-name, --openapi-title, --copilot-title は必須です。")
  }

  return options
}

function listEnvFiles() {
  return listChildFiles(ENV_DIR, name => name.startsWith(".env"))
}

function replaceEnvAppName(content, appName) {
  const pattern = /^APP_NAME=.*/m

  if (!pattern.test(content)) {
    return null
  }

  return content.replace(pattern, () => `APP_NAME=${appName}`)
}

function replaceOpenapiTitle(content, title) {
  const pattern = /^  title: .*/m

  if (!pattern.test(content)) {
    return null
  }

  return content.replace(pattern, () => `  title: ${title}`)
}

function replaceCopilotTitle(content, title) {
  const pattern = /^# .*/m

  if (!pattern.test(content)) {
    return null
  }

  return content.replace(pattern, () => `# ${title}`)
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

  const changedFiles = []

  for (const envFile of listEnvFiles()) {
    const result = updateAbsoluteFile(
      envFile,
      content => replaceEnvAppName(content, options.appName),
      options.dryRun
    )

    if (result) {
      changedFiles.push(result)
    }
  }

  const openapiResult = updateFile(
    OPENAPI_FILE,
    content => replaceOpenapiTitle(content, options.openapiTitle),
    options.dryRun
  )

  if (openapiResult) {
    changedFiles.push(openapiResult)
  }

  const copilotResult = updateFile(
    COPILOT_INSTRUCTIONS_FILE,
    content => replaceCopilotTitle(content, options.copilotTitle),
    options.dryRun
  )

  if (copilotResult) {
    changedFiles.push(copilotResult)
  }

  if (changedFiles.length === 0) {
    console.log("変更対象は見つかりませんでした。")
    return
  }

  console.log(`${options.dryRun ? "ドライラン" : "置換完了"}: ${changedFiles.length}ファイル`)

  for (const file of changedFiles) {
    console.log(`- ${file}`)
  }

  console.log("")
  console.log("注意: OpenAPI の生成物を更新するには make gen-api を実行してください。")
}

main()
