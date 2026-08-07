#!/usr/bin/env -S tsx
// README と OpenAPI の GitHub リポジトリ参照を <owner>/<repo> へ置換する。置換規則は
// lib/repository-reference.ts が持ち、ここは書き込み・出力だけを担う。

import { updateFile } from "../lib/file-utils";
import {
  replaceOpenapiTermsOfService,
  replaceReadmeReferences,
  REPOSITORY_REFERENCE_TARGETS,
} from "./repository-reference";
import { type SetupOptions, newSetupCommand } from "../lib/runtime";
import { ensureRepositoryReference } from "../lib/validators";

function run(repository: string, dryRun: boolean): void {
  const changedFiles: string[] = [];

  for (const file of REPOSITORY_REFERENCE_TARGETS.readmeFiles) {
    const result = updateFile(
      file,
      (content) => replaceReadmeReferences(content, repository),
      dryRun,
    );

    if (result) {
      changedFiles.push(result);
    }
  }

  const openapiResult = updateFile(
    REPOSITORY_REFERENCE_TARGETS.openapiFile,
    (content) => replaceOpenapiTermsOfService(content, repository),
    dryRun,
  );

  if (openapiResult) {
    changedFiles.push(openapiResult);
  }

  if (changedFiles.length === 0) {
    console.log("変更対象は見つかりませんでした。");
    return;
  }

  console.log(`${dryRun ? "ドライラン" : "置換完了"}: ${changedFiles.length}ファイル`);

  for (const file of changedFiles) {
    console.log(`- ${file}`);
  }

  console.log("");
  console.log("注意: OpenAPI の生成物を更新するには make gen-api を実行してください。");
}

const program = newSetupCommand("replace-repository-reference");
program
  .description("README と OpenAPI の GitHub リポジトリ参照を <owner>/<repo> へ置換する")
  .argument("<owner/repo>", "置換後のリポジトリ参照（例: example-org/example-api）")
  .action((repository: string, options: SetupOptions) => {
    try {
      ensureRepositoryReference(repository);
    } catch (error) {
      program.error(`エラー: ${(error as Error).message}`);
    }

    run(repository, options.dryRun);
  })
  .parse();
