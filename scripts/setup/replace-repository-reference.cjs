const { parseCommonFlags, exitWithUsage } = require("./lib/runtime.cjs")
const { updateFile } = require("./lib/file-utils.cjs")
const { ensureRepositoryReference } = require("./lib/validators.cjs")

const README_FILES = ["README.md", "README.ja.md"]
const OPENAPI_FILE = "openapi/openapi.yaml"

function printUsage() {
  console.log(`使用方法:
  node scripts/setup/replace-repository-reference.cjs <owner>/<repo> [--dry-run]

例:
  node scripts/setup/replace-repository-reference.cjs example-org/example-api
  node scripts/setup/replace-repository-reference.cjs example-org/example-api --dry-run
`)
}

function parseArgs(argv) {
  const options = parseCommonFlags(argv)
  const positionals = options.rest

  if (options.help) {
    return options
  }

  if (positionals.length !== 1) {
    throw new Error("置換後のリポジトリ参照を <owner>/<repo> 形式で指定してください。")
  }

  options.repository = positionals[0]
  ensureRepositoryReference(options.repository)

  return options
}

function updateReadme(content, repository) {
  const repoName = repository.split("/")[1]

  return content
    .replace(/^# .*/m, `# ${repoName}`)
    .replace(
      /https:\/\/img\.shields\.io\/github\/go-mod\/go-version\/[^\s)]+/g,
      `https://img.shields.io/github/go-mod/go-version/${repository}`
    )
    .replace(
      /https:\/\/img\.shields\.io\/github\/license\/[^\s)]+/g,
      `https://img.shields.io/github/license/${repository}`
    )
    .replace(
      /https:\/\/github\.com\/[^/\s>]+\/[^/\s>]+\.git/g,
      `https://github.com/${repository}.git`
    )
    .replace(/^cd .*/m, `cd ${repoName}`)
}

function updateOpenapi(content, repository) {
  return content.replace(
    /^  termsOfService: https:\/\/github\.com\/.*$/m,
    `  termsOfService: https://github.com/${repository}`
  )
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

  for (const file of README_FILES) {
    const result = updateFile(file, content => updateReadme(content, options.repository), options.dryRun)

    if (result) {
      changedFiles.push(result)
    }
  }

  const openapiResult = updateFile(
    OPENAPI_FILE,
    content => updateOpenapi(content, options.repository),
    options.dryRun
  )

  if (openapiResult) {
    changedFiles.push(openapiResult)
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
