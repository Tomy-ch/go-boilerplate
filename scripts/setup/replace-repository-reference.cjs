const { newSetupCommand } = require("./lib/runtime.cjs")
const { updateFile } = require("./lib/file-utils.cjs")
const { ensureRepositoryReference } = require("./lib/validators.cjs")

const README_FILES = ["README.md", "README.ja.md"]
const OPENAPI_FILE = "openapi/openapi.yaml"

function updateReadme(content, repository) {
  const repoName = repository.split("/")[1]

  return content
    .replace(/^# .*/m, () => `# ${repoName}`)
    .replace(
      /https:\/\/img\.shields\.io\/github\/go-mod\/go-version\/[^\s)]+/g,
      () => `https://img.shields.io/github/go-mod/go-version/${repository}`
    )
    .replace(
      /https:\/\/img\.shields\.io\/github\/license\/[^\s)]+/g,
      () => `https://img.shields.io/github/license/${repository}`
    )
    .replace(
      /https:\/\/github\.com\/[^/\s>]+\/[^/\s>]+\.git/g,
      () => `https://github.com/${repository}.git`
    )
    .replace(/^cd .*/m, () => `cd ${repoName}`)
}

function updateOpenapi(content, repository) {
  return content.replace(
    /^  termsOfService: https:\/\/github\.com\/.*$/m,
    () => `  termsOfService: https://github.com/${repository}`
  )
}

function run(repository, dryRun) {
  const changedFiles = []

  for (const file of README_FILES) {
    const result = updateFile(file, content => updateReadme(content, repository), dryRun)

    if (result) {
      changedFiles.push(result)
    }
  }

  const openapiResult = updateFile(
    OPENAPI_FILE,
    content => updateOpenapi(content, repository),
    dryRun
  )

  if (openapiResult) {
    changedFiles.push(openapiResult)
  }

  if (changedFiles.length === 0) {
    console.log("変更対象は見つかりませんでした。")
    return
  }

  console.log(`${dryRun ? "ドライラン" : "置換完了"}: ${changedFiles.length}ファイル`)

  for (const file of changedFiles) {
    console.log(`- ${file}`)
  }

  console.log("")
  console.log("注意: OpenAPI の生成物を更新するには make gen-api を実行してください。")
}

const program = newSetupCommand("replace-repository-reference")
program
  .description("README と OpenAPI の GitHub リポジトリ参照を <owner>/<repo> へ置換する")
  .argument("<owner/repo>", "置換後のリポジトリ参照（例: example-org/example-api）")
  .action((repository, options) => {
    try {
      ensureRepositoryReference(repository)
    } catch (error) {
      program.error(`エラー: ${error.message}`)
    }

    run(repository, options.dryRun)
  })
  .parse()
