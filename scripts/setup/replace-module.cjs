const path = require("path")
const { ROOT_DIR, parseCommonFlags, exitWithUsage } = require("./lib/runtime.cjs")
const {
  listFilesRecursive,
  updateAbsoluteFile,
  countOccurrences
} = require("./lib/file-utils.cjs")

const TARGET_EXTENSIONS = new Set([
  ".go",
  ".yaml",
  ".yml",
  ".mod",
  ".sum",
  ".md",
  ".js",
  ".json",
  ".cjs",
  ".html"
])
const TARGET_BASENAMES = new Set(["Dockerfile"])
const EXCLUDED_DIRECTORIES = new Set(["vendor", "tmp", "node_modules", ".git"])
const EXCLUDED_PATH_PREFIXES = [
  `docs${path.sep}`
]
const EXCLUDED_PATH_SUFFIXES = [
  ".gen.go",
  ".sql.go",
  "_mock.go",
  `${path.sep}openapi.gen.yaml`
]

function printUsage() {
  console.log(`使用方法:
  node scripts/setup/replace-module.cjs <old-module> <new-module> [--dry-run]

例:
  node scripts/setup/replace-module.cjs boilerplate-go example-api
  node scripts/setup/replace-module.cjs old-project new-project --dry-run

補足:
  Go モジュール名は boilerplate-go のような単純なプロジェクト名を想定しています。
  docs 配下と生成物 (*.gen.go, *.sql.go, *_mock.go, openapi/openapi.gen.yaml) は対象外です。`)
}

function ensureSimpleModuleName(value, flagName) {
  if (!/^[A-Za-z0-9._-]+$/.test(value)) {
    throw new Error(
      `${flagName} は boilerplate-go のような単純なプロジェクト名で指定してください。`
    )
  }
}

function parseArgs(argv) {
  const options = parseCommonFlags(argv)
  const positionals = options.rest

  if (options.help) {
    return options
  }

  if (positionals.length !== 2) {
    throw new Error("旧モジュール名と新モジュール名を指定してください。")
  }

  options.oldModule = positionals[0]
  options.newModule = positionals[1]
  ensureSimpleModuleName(options.oldModule, "旧モジュール名")
  ensureSimpleModuleName(options.newModule, "新モジュール名")

  return options
}

function shouldProcessFile(filePath) {
  const relativePath = path.relative(ROOT_DIR, filePath)
  const baseName = path.basename(filePath)
  const ext = path.extname(filePath)

  if (EXCLUDED_PATH_PREFIXES.some(prefix => relativePath.startsWith(prefix))) {
    return false
  }

  if (EXCLUDED_PATH_SUFFIXES.some(suffix => relativePath.endsWith(suffix))) {
    return false
  }

  if (TARGET_BASENAMES.has(baseName)) {
    return true
  }

  return TARGET_EXTENSIONS.has(ext)
}

function collectTargetFiles(dirPath, files = []) {
  return listFilesRecursive(
    dirPath,
    {
      excludedDirectories: EXCLUDED_DIRECTORIES,
      shouldIncludeFile: shouldProcessFile
    },
    files
  )
}

function replaceInFile(filePath, oldModule, newModule, dryRun) {
  let occurrences = 0
  const relativePath = updateAbsoluteFile(filePath, original => {
    if (!original.includes(oldModule)) {
      return null
    }

    occurrences = countOccurrences(original, oldModule)
    return original.split(oldModule).join(newModule)
  }, dryRun)

  if (!relativePath) {
    return null
  }

  return { relativePath, occurrences }
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

  if (options.oldModule === options.newModule) {
    console.error("エラー: 旧モジュール名と新モジュール名が同一です。")
    process.exit(1)
  }

  const files = collectTargetFiles(ROOT_DIR)
  const changedFiles = []
  let replacementCount = 0

  for (const filePath of files) {
    const result = replaceInFile(
      filePath,
      options.oldModule,
      options.newModule,
      options.dryRun
    )

    if (!result) {
      continue
    }

    changedFiles.push(result)
    replacementCount += result.occurrences
  }

  if (changedFiles.length === 0) {
    console.log("置換対象は見つかりませんでした。")
    return
  }

  console.log(
    `${options.dryRun ? "Dry Run" : "置換完了"}: ${changedFiles.length}ファイル / ${replacementCount}箇所`
  )

  for (const file of changedFiles) {
    console.log(`- ${file.relativePath} (${file.occurrences}箇所)`)
  }
}

main()
