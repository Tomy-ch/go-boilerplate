#!/usr/bin/env -S tsx

import fs from "node:fs";

import { CODEOWNERS_FILE, parseOwners, replaceCodeowners } from "./codeowners";
import { toAbsolutePath, updateFile } from "../lib/file-utils";
import { type SetupOptions, newSetupCommand } from "../lib/runtime";
import { ensureCodeOwners } from "../lib/validators";

type Options = SetupOptions & {
  owners: string;
};

function run(owners: string, dryRun: boolean): void {
  if (!fs.existsSync(toAbsolutePath(CODEOWNERS_FILE))) {
    console.error(`✖ ${CODEOWNERS_FILE} が見つかりません。`);
    process.exit(1);
  }

  let skippedLines: number[] = [];
  const result = updateFile(
    CODEOWNERS_FILE,
    (content) => {
      const updated = replaceCodeowners(content, owners);
      skippedLines = updated.skippedLines;

      return updated.content;
    },
    dryRun,
  );

  if (skippedLines.length > 0) {
    console.log(
      `⚠ 所有者を判定できず未置換の行: ${skippedLines.join(", ")}（手で確認してください）`,
    );
  }

  if (!result) {
    console.log("既に指定の所有者のため変更はありません。");
    return;
  }

  console.log(`${dryRun ? "ドライラン" : "置換完了"}: ${CODEOWNERS_FILE}`);
  console.log(`- 所有者: ${owners}`);
  console.log("");
  console.log(
    "注意: 実際にレビューを必須化するにはブランチ保護の「Require review from Code Owners」が別途必要です。",
  );
  console.log(
    "また GitHub 上で解決できない所有者（組織そのもの・存在しない team・未検証メール）は黙って無視されます。",
  );
}

const program = newSetupCommand("replace-codeowners");
program
  .description(".github/CODEOWNERS の全ルールの所有者を一括で置換する")
  .requiredOption(
    "--owners <owners>",
    "所有者（空白区切りで複数指定可。例: '@example-org/tech-leads @example-org/security'）",
  )
  .action((options: Options) => {
    const owners = parseOwners(options.owners);

    try {
      ensureCodeOwners(owners);
    } catch (error) {
      program.error(`エラー: ${(error as Error).message}`);
    }

    run(owners.join(" "), options.dryRun);
  })
  .parse();
