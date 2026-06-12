const fs = require("fs")
const path = require("path")
const { ROOT_DIR } = require("./runtime.cjs")

function toAbsolutePath(relativePath) {
  return path.join(ROOT_DIR, relativePath)
}

function toRelativePath(filePath) {
  return path.relative(ROOT_DIR, filePath)
}

function updateFile(relativePath, transformer, dryRun) {
  return updateAbsoluteFile(toAbsolutePath(relativePath), transformer, dryRun)
}

function updateAbsoluteFile(filePath, transformer, dryRun) {
  if (!fs.existsSync(filePath)) {
    return null
  }

  const original = fs.readFileSync(filePath, "utf8")
  const updated = transformer(original)

  if (updated === null || updated === original) {
    return null
  }

  if (!dryRun) {
    fs.writeFileSync(filePath, updated)
  }

  return toRelativePath(filePath)
}

// options.shouldIncludeFile はエントリの**フルパス**を受け取る（listChildFiles の predicate は basename を受け取る点に注意）。
function listFilesRecursive(dirPath, options = {}, files = []) {
  const excludedDirectories = options.excludedDirectories ?? new Set()
  const shouldIncludeFile = options.shouldIncludeFile ?? (() => true)
  const entries = fs.readdirSync(dirPath, { withFileTypes: true })
    .sort((a, b) => a.name.localeCompare(b.name))

  for (const entry of entries) {
    const entryPath = path.join(dirPath, entry.name)

    if (entry.isDirectory()) {
      if (excludedDirectories.has(entry.name)) {
        continue
      }

      listFilesRecursive(entryPath, options, files)
      continue
    }

    if (entry.isFile() && shouldIncludeFile(entryPath)) {
      files.push(entryPath)
    }
  }

  return files
}

// predicate はエントリの**ファイル名(basename)**を受け取る（listFilesRecursive の shouldIncludeFile はフルパスを受け取る点に注意）。
function listChildFiles(relativeDir, predicate = () => true) {
  const dirPath = toAbsolutePath(relativeDir)

  return fs.readdirSync(dirPath, { withFileTypes: true })
    .filter(entry => entry.isFile() && predicate(entry.name))
    .map(entry => path.join(dirPath, entry.name))
    .sort((a, b) => a.localeCompare(b))
}

module.exports = {
  toAbsolutePath,
  toRelativePath,
  updateFile,
  updateAbsoluteFile,
  listFilesRecursive,
  listChildFiles
}
