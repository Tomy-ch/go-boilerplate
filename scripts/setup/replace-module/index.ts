#!/usr/bin/env -S tsx
// Go モジュール名をプロジェクト全体で一括置換する。判定（対象ファイル種別・置換規則）は
// lib/module-replace.ts が持ち、ここは走査・書き込み・出力だけを担う。

import path from "node:path";

import { listFilesRecursive, updateAbsoluteFile } from "../lib/file-utils";
import {
  EXCLUDED_DIRECTORIES,
  ensureModuleArguments,
  isReplacementTarget,
  replaceModuleOccurrences,
} from "./module-replace";
import { ROOT_DIR, type SetupOptions, newSetupCommand } from "../lib/runtime";

type ChangedFile = {
  relativePath: string;
  occurrences: number;
};

function collectTargetFiles(dirPath: string): string[] {
  return listFilesRecursive(dirPath, {
    excludedDirectories: EXCLUDED_DIRECTORIES,
    shouldIncludeFile: (entryPath) => isReplacementTarget(path.relative(ROOT_DIR, entryPath)),
  });
}

function replaceInFile(
  filePath: string,
  oldModule: string,
  newModule: string,
  dryRun: boolean,
): ChangedFile | null {
  let occurrences = 0;
  const relativePath = updateAbsoluteFile(
    filePath,
    (original) => {
      const replaced = replaceModuleOccurrences(original, oldModule, newModule);

      if (replaced === null) {
        return null;
      }

      occurrences = replaced.occurrences;

      return replaced.content;
    },
    dryRun,
  );

  if (!relativePath) {
    return null;
  }

  return { relativePath, occurrences };
}

function run(oldModule: string, newModule: string, dryRun: boolean): void {
  const files = collectTargetFiles(ROOT_DIR);
  const changedFiles: ChangedFile[] = [];
  let replacementCount = 0;

  for (const filePath of files) {
    const result = replaceInFile(filePath, oldModule, newModule, dryRun);

    if (!result) {
      continue;
    }

    changedFiles.push(result);
    replacementCount += result.occurrences;
  }

  if (changedFiles.length === 0) {
    console.log("置換対象は見つかりませんでした。");
    return;
  }

  console.log(
    `${dryRun ? "Dry Run" : "置換完了"}: ${changedFiles.length}ファイル / ${replacementCount}箇所`,
  );

  for (const file of changedFiles) {
    console.log(`- ${file.relativePath} (${file.occurrences}箇所)`);
  }
}

const program = newSetupCommand("replace-module");
program
  .description("Go モジュール名をプロジェクト全体で一括置換する")
  .argument("<old-module>", "旧モジュール名（例: go-boilerplate）")
  .argument("<new-module>", "新モジュール名（例: example-api）")
  .addHelpText(
    "after",
    `
補足:
  Go モジュール名は go-boilerplate のような単純なプロジェクト名を想定しています。
  docs 配下・scripts/setup・生成物 (*.gen.go, *.sql.go, mock ファイル, openapi/openapi.gen.yaml) は対象外です。
  生成物は置換後に make gen-api で再生成してください。`,
  )
  .action((oldModule: string, newModule: string, options: SetupOptions) => {
    try {
      ensureModuleArguments(oldModule, newModule);
    } catch (error) {
      program.error(`エラー: ${(error as Error).message}`);
    }

    run(oldModule, newModule, options.dryRun);
  })
  .parse();
