const path = require("path")
const { ROOT_DIR, parseCommonFlags, exitWithUsage } = require("./lib/runtime.cjs")
const {
  listFilesRecursive,
  updateAbsoluteFile
} = require("./lib/file-utils.cjs")

const TARGET_EXTENSIONS = new Set([
  ".go",
  ".yaml",
  ".yml",
  ".mod",
  ".md",
  ".js",
  ".json",
  ".cjs",
  ".html"
])
const TARGET_BASENAMES = new Set(["Dockerfile"])
const EXCLUDED_DIRECTORIES = new Set(["vendor", "tmp", "node_modules", ".git"])
const EXCLUDED_PATH_PREFIXES = [
  `docs${path.sep}`,
  `scripts${path.sep}setup${path.sep}`
]
const EXCLUDED_PATH_SUFFIXES = [
  ".gen.go",
  ".sql.go",
  `${path.sep}openapi.gen.yaml`
]
// mockgen 生成物（make gen-api で再生成されるため対象外）
const EXCLUDED_BASENAME_PATTERNS = [
  /^mock_.*\.go$/,
  /_mock\.go$/
]

function printUsage() {
  console.log(`使用方法:
  node scripts/setup/replace-module.cjs <old-module> <new-module> [--dry-run]

例:
  node scripts/setup/replace-module.cjs go-boilerplate example-api
  node scripts/setup/replace-module.cjs old-project new-project --dry-run

補足:
  Go モジュール名は go-boilerplate のような単純なプロジェクト名を想定しています。
  docs 配下・scripts/setup・生成物 (*.gen.go, *.sql.go, mock ファイル, openapi/openapi.gen.yaml) は対象外です。
  生成物は置換後に make gen-api で再生成してください。`)
}

function ensureSimpleModuleName(value, flagName) {
  if (!/^[A-Za-z0-9._-]+$/.test(value)) {
    throw new Error(
      `${flagName} は go-boilerplate のような単純なプロジェクト名で指定してください。`
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

  if (options.oldModule === options.newModule) {
    throw new Error("旧モジュール名と新モジュール名が同一です。")
  }

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

  if (EXCLUDED_BASENAME_PATTERNS.some(pattern => pattern.test(baseName))) {
    return false
  }

  if (TARGET_BASENAMES.has(baseName)) {
    return true
  }

  return TARGET_EXTENSIONS.has(ext)
}

function collectTargetFiles(dirPath) {
  return listFilesRecursive(dirPath, {
    excludedDirectories: EXCLUDED_DIRECTORIES,
    shouldIncludeFile: shouldProcessFile
  })
}

function replaceInFile(filePath, oldModule, newModule, dryRun) {
  let occurrences = 0
  const relativePath = updateAbsoluteFile(filePath, original => {
    const parts = original.split(oldModule)

    if (parts.length === 1) {
      return null
    }

    occurrences = parts.length - 1
    return parts.join(newModule)
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
