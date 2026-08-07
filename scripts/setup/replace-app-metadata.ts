#!/usr/bin/env -S tsx
// env ファイル・OpenAPI・Copilot 指示書のアプリ名/タイトルを置換する。置換規則は
// lib/app-metadata.ts が持ち、ここは対象ファイルの列挙・書き込み・出力だけを担う。

import {
  isEnvFile,
  replaceCopilotTitle,
  replaceEnvAppName,
  replaceOpenapiTitle,
} from "./lib/app-metadata";
import { listChildFiles, updateAbsoluteFile, updateFile } from "./lib/file-utils";
import { type SetupOptions, newSetupCommand } from "./lib/runtime";

const ENV_DIR = "env";
const OPENAPI_FILE = "openapi/openapi.yaml";
const COPILOT_INSTRUCTIONS_FILE = ".github/copilot-instructions.md";

type Options = SetupOptions & {
  appName: string;
  openapiTitle: string;
  copilotTitle: string;
};

function run(options: Options): void {
  const changedFiles: string[] = [];

  for (const envFile of listChildFiles(ENV_DIR, isEnvFile)) {
    const result = updateAbsoluteFile(
      envFile,
      (content) => replaceEnvAppName(content, options.appName),
      options.dryRun,
    );

    if (result) {
      changedFiles.push(result);
    }
  }

  const openapiResult = updateFile(
    OPENAPI_FILE,
    (content) => replaceOpenapiTitle(content, options.openapiTitle),
    options.dryRun,
  );

  if (openapiResult) {
    changedFiles.push(openapiResult);
  }

  const copilotResult = updateFile(
    COPILOT_INSTRUCTIONS_FILE,
    (content) => replaceCopilotTitle(content, options.copilotTitle),
    options.dryRun,
  );

  if (copilotResult) {
    changedFiles.push(copilotResult);
  }

  if (changedFiles.length === 0) {
    console.log("変更対象は見つかりませんでした。");
    return;
  }

  console.log(`${options.dryRun ? "ドライラン" : "置換完了"}: ${changedFiles.length}ファイル`);

  for (const file of changedFiles) {
    console.log(`- ${file}`);
  }

  console.log("");
  console.log("注意: OpenAPI の生成物を更新するには make gen-api を実行してください。");
}

newSetupCommand("replace-app-metadata")
  .description("env ファイル・OpenAPI・Copilot 指示書のアプリ名/タイトルを置換する")
  .requiredOption("--app-name <name>", "アプリケーション名（env の APP_NAME に反映）")
  .requiredOption("--openapi-title <title>", "OpenAPI 仕様の title")
  .requiredOption("--copilot-title <title>", "Copilot 指示書の先頭見出し")
  .action((options: Options) => {
    run(options);
  })
  .parse();
