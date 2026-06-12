const fs = require("fs")
const { newSetupCommand } = require("./lib/runtime.cjs")
const { toAbsolutePath, updateFile } = require("./lib/file-utils.cjs")
const { ensureFourDigitYear } = require("./lib/validators.cjs")

const LICENSE_FILE = "LICENSE"

function run(holder, year, dryRun) {
  if (!fs.existsSync(toAbsolutePath(LICENSE_FILE))) {
    console.error(`✖ ${LICENSE_FILE} が見つかりません。`)
    process.exit(1)
  }

  const pattern = /^Copyright \(c\) .*/m
  const result = updateFile(LICENSE_FILE, original => {
    if (!pattern.test(original)) {
      throw new Error("LICENSE に著作権表示が見つかりませんでした。")
    }

    return original.replace(pattern, () => `Copyright (c) ${year} ${holder}`)
  }, dryRun)

  if (!result) {
    console.log("既に最新のため変更はありません。")
    return
  }

  console.log(`${dryRun ? "ドライラン" : "置換完了"}: LICENSE`)
  console.log(`- Copyright (c) ${year} ${holder}`)
}

const program = newSetupCommand("replace-license-copyright")
program
  .description("LICENSE の著作権表示（年・権利者）を更新する")
  .requiredOption("--holder <name>", "著作権者名")
  .option("--year <yyyy>", "著作権表示の年（既定: 現在の年）", String(new Date().getFullYear()))
  .action(options => {
    try {
      ensureFourDigitYear(options.year)
    } catch (error) {
      program.error(`エラー: ${error.message}`)
    }

    run(options.holder, options.year, options.dryRun)
  })
  .parse()
