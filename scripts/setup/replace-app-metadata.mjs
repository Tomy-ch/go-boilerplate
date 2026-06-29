import { newSetupCommand } from "./lib/runtime.mjs"
import {
  listChildFiles,
  updateAbsoluteFile,
  updateFile
} from "./lib/file-utils.mjs"

const ENV_DIR = "env"
const OPENAPI_FILE = "openapi/openapi.yaml"
const COPILOT_INSTRUCTIONS_FILE = ".github/copilot-instructions.md"

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

function run(options) {
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

const program = newSetupCommand("replace-app-metadata")
program
  .description("env ファイル・OpenAPI・Copilot 指示書のアプリ名/タイトルを置換する")
  .requiredOption("--app-name <name>", "アプリケーション名（env の APP_NAME に反映）")
  .requiredOption("--openapi-title <title>", "OpenAPI 仕様の title")
  .requiredOption("--copilot-title <title>", "Copilot 指示書の先頭見出し")
  .action(options => {
    run(options)
  })
  .parse()
