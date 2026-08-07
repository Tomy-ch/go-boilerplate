#!/usr/bin/env -S tsx
// `upsert-pr-comment` を使うジョブに secret が渡っていないかを検査する lint スクリプト。
//
// Actions のシークレットマスキングは、ランナーがジョブ出力をログ表示用に捕捉する経路にしか効かない。
// 検査ログを `tee` でファイルへ落としたバイトは素通りするため、そのファイルを本文にする
// `upsert-pr-comment` では、ログ上はマスク済みに見える値でも生のまま PR コメントに載る。
// マスキングを当てにできない以上、「本文を作るジョブに secret を渡さない」を規約として守るしかなく、
// この検査はその規約が将来 `env:` 1 行で破られることへの退行ガードにあたる。
//
// GITHUB_TOKEN はコメント投稿そのものに必要で、かつ Actions が発行する短命トークンなので許可する。
//
// 検出できるのは `${{ }}` 式に現れる secrets コンテキストの直接参照（`secrets.NAME` /
// `secrets['NAME']` / `toJSON(secrets)` のようなコンテキスト全体）に限る。別ジョブで secret を
// 読んで `needs.<job>.outputs` 経由で渡す間接参照は静的には追えないので、この検査は通る。
//
// 判断は含めず、ワークフロー定義から機械的に導けるものだけを見る。1 件でも違反があれば非 0 で終了する。

import fs from "node:fs";
import path from "node:path";

import {
  COMMENT_ACTION,
  describeSecret,
  secretReferences,
  usesCommentAction,
} from "./lib/pr-comment-secret";
import { splitJobs } from "./lib/workflow";

const REPO_ROOT = process.cwd();
const WORKFLOWS_DIR = ".github/workflows";

type Finding = {
  file: string;
  line: number;
  message: string;
};

const findings: Finding[] = [];

function report(file: string, line: number, message: string): void {
  findings.push({ file, line, message });
}

function listWorkflowFiles(): string[] {
  const dir = path.join(REPO_ROOT, WORKFLOWS_DIR);
  if (!fs.existsSync(dir)) return [];

  return fs
    .readdirSync(dir)
    .filter((name) => name.endsWith(".yaml") || name.endsWith(".yml"))
    .sort()
    .map((name) => path.join(WORKFLOWS_DIR, name));
}

const workflowFiles = listWorkflowFiles();

// 検査対象 0 件は「問題なし」ではなく「検査が働いていない」。実行位置の誤りを成功と report しない。
if (workflowFiles.length === 0) {
  console.error(
    `✘ pr-comment-secret-lint: ${WORKFLOWS_DIR}/ にワークフローが見つかりません（リポジトリルートで実行してください）`,
  );
  process.exit(2);
}

let checkedJobs = 0;

for (const rel of workflowFiles) {
  const source = fs.readFileSync(path.join(REPO_ROOT, rel), "utf8");
  const { jobs, preamble, found } = splitJobs(source);

  if (!found) {
    console.error(`✘ pr-comment-secret-lint: ${rel} に jobs: が見つかりません`);
    process.exit(2);
  }

  const commenting = jobs.filter(usesCommentAction);
  checkedJobs += commenting.length;
  if (commenting.length === 0) continue;

  for (const job of commenting) {
    for (const line of job.lines) {
      for (const { number, name } of secretReferences(line)) {
        report(
          rel,
          number,
          `ジョブ \`${job.id}\` は ${COMMENT_ACTION} を使うため ${describeSecret(name)} を渡せません（マスキングは tee したファイルに効かず、生値が PR コメントに載ります）`,
        );
      }
    }
  }

  for (const line of preamble) {
    for (const { number, name } of secretReferences(line)) {
      report(
        rel,
        number,
        `ワークフロー全体に及ぶ ${describeSecret(name)} は ${COMMENT_ACTION} を使うジョブにも届きます（マスキングは tee したファイルに効かず、生値が PR コメントに載ります）`,
      );
    }
  }
}

if (findings.length > 0) {
  console.error(`✘ pr-comment-secret-lint: ${findings.length} 件の違反\n`);
  let current: string | null = null;

  for (const finding of findings) {
    if (finding.file !== current) {
      if (current !== null) console.error("");
      console.error(`  ${finding.file}`);
      current = finding.file;
    }
    console.error(`    :${finding.line}  ${finding.message}`);
  }

  console.error(
    `\n検査 ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブ中 ${findings.length} 件 NG`,
  );
  process.exit(1);
}

console.log(
  `✓ pr-comment-secret-lint: ${workflowFiles.length} ワークフロー / ${checkedJobs} ジョブすべて OK`,
);
