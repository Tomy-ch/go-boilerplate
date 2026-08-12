#!/usr/bin/env -S tsx

import fs from "node:fs";

import { toAbsolutePath, updateFile } from "../lib/file-utils";
import { LICENSE_FILE, replaceCopyright } from "./license-copyright";
import { type SetupOptions, newSetupCommand } from "../lib/runtime";
import { ensureFourDigitYear } from "../lib/validators";

type Options = SetupOptions & {
  holder: string;
  year: string;
};

function run(holder: string, year: string, dryRun: boolean): void {
  if (!fs.existsSync(toAbsolutePath(LICENSE_FILE))) {
    console.error(`✖ ${LICENSE_FILE} が見つかりません。`);
    process.exit(1);
  }

  const result = updateFile(
    LICENSE_FILE,
    (original) => replaceCopyright(original, holder, year),
    dryRun,
  );

  if (!result) {
    console.log("既に最新のため変更はありません。");
    return;
  }

  console.log(`${dryRun ? "ドライラン" : "置換完了"}: ${LICENSE_FILE}`);
  console.log(`- Copyright (c) ${year} ${holder}`);
}

const program = newSetupCommand("replace-license-copyright");
program
  .description("LICENSE の著作権表示（年・権利者）を更新する")
  .requiredOption("--holder <name>", "著作権者名")
  .option("--year <yyyy>", "著作権表示の年（既定: 現在の年）", String(new Date().getFullYear()))
  .action((options: Options) => {
    try {
      ensureFourDigitYear(options.year);
    } catch (error) {
      program.error(`エラー: ${(error as Error).message}`);
    }

    run(options.holder, options.year, options.dryRun);
  })
  .parse();
