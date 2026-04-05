const path = require("path")

const ROOT_DIR = path.resolve(__dirname, "../../..")

function parseCommonFlags(argv) {
  const options = {
    dryRun: false,
    help: false,
    rest: []
  }

  for (const arg of argv) {
    if (arg === "--dry-run") {
      options.dryRun = true
      continue
    }

    if (arg === "--help" || arg === "-h") {
      options.help = true
      continue
    }

    options.rest.push(arg)
  }

  return options
}

function exitWithUsage(error, printUsage) {
  console.error(`エラー: ${error.message}`)
  console.error("")
  printUsage()
  process.exit(1)
}

module.exports = {
  ROOT_DIR,
  parseCommonFlags,
  exitWithUsage
}
