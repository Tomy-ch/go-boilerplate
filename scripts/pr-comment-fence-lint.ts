#!/usr/bin/env -S tsx
// ワークフローが自前で組む Markdown フェンスの退行を検査する lint スクリプト。
//
// `upsert-pr-comment` が本文中の最長バッククォート連 + 1 をフェンス長とするのは `details-summary`
// 経路だけで、渡さない呼び出しでは本文を素通しする。そこでフェンスを自前に組むワークフローが
// フェンスの責任を負い、固定 3 連は PR 提出者が書いたソース行を引用する本文に閉じられる。
//
// inline code span も長さ 1 のフェンスでしかないので、同じ話が span にも及ぶ。値に含まれる 1 個の
// バッククォートが span を閉じ、以降が生 Markdown に戻る。git のパスに使えない文字は NUL と `/` だけ
// なので、ファイル名を span へ入れる素通し経路は同じ穴を持つ。
//
// 機械的に判定できる 3 点のみを見る。「その本文が攻撃者制御か」は判定しない。
//
//   1. `run:` ブロックが固定長のフェンスを出力していないこと
//   2. 複数のワークフローが持つ `fence_for` の実装が互いに完全一致していること
//   3. 素通し呼び出しを持つワークフローが、inline code span の内側へ値を補間していないこと
//
// 2 が要るのは、実装を composite action へ集約できないため。フェンス文字列を output 経由で受け取ると
// バッククォートがシェルの二重引用符文脈でコマンド置換になる。複製は意図的な選択であり、そのぶん
// 片方だけが直る事故を機械で止める。
//
// シェルファイルへ切り出して `source` する経路も同じく採らない。`actionlint` が読むのは
// `.github/workflows/**`、`actions-shellcheck` が読むのは `.github/actions/**` だけなので、集約先は
// どちらの検査対象でもなくなる。検査済みの複製を、リポジトリで唯一どの linter も読まないファイルへ
// 置き換えることになり、複製が持ち込む唯一のリスク（片方だけが直る drift）は 2 が既に止めている。
//
// 3 の粒度はワークフローファイル単位で、ステップから本文ファイルへのデータフローは追わない
// （`out=/tmp/...` の間接参照解決は脆く、既存の行ベース走査から逸脱する）。素通し呼び出しと
// `details-summary` 付き呼び出しが同居するファイルでは過剰検出になり得るが、意図した単純化として
// 除外リストで受ける。span を変数経由や jq の文字列連結で組む形は検出できない（偽陰性）。
//
// この入口はファイル入出力と終了コードだけを担う。走査と判定は lib/pr-comment-fence.ts にある。

import fs from "node:fs";
import path from "node:path";

import {
  PASS_THROUGH_EXCLUSIONS,
  compareImplementations,
  scanWorkflow,
} from "./lib/pr-comment-fence";
import { selectWorkflowFiles } from "./lib/workflow";

const REPO_ROOT = process.cwd();
const WORKFLOWS_DIR = ".github/workflows";

function listWorkflows(): string[] {
  return selectWorkflowFiles(fs.readdirSync(path.join(REPO_ROOT, WORKFLOWS_DIR)), WORKFLOWS_DIR);
}

const violations: string[] = [];
const implementations: Array<[string, string]> = [];
const workflowFiles = listWorkflows();

for (const file of workflowFiles) {
  const lines = fs.readFileSync(path.join(REPO_ROOT, file), "utf8").split("\n");
  const scan = scanWorkflow(file, lines);

  violations.push(...scan.violations);
  if (scan.implementation !== null) implementations.push([file, scan.implementation]);
}

violations.push(...compareImplementations(implementations));

if (violations.length > 0) {
  for (const violation of violations) console.error(`✗ ${violation}`);
  console.error(`\n✗ pr-comment-fence-lint: ${violations.length} 件の違反があります`);
  process.exit(1);
}

// 除外を黙ったまま緑にすると「検査が通った」と「見ていない」の区別が付かなくなる。
for (const [name, reason] of PASS_THROUGH_EXCLUSIONS) {
  console.log(`- span 検査を除外: ${name} — ${reason}`);
}

console.log(
  `✓ pr-comment-fence-lint: ${workflowFiles.length} ワークフロー / fence_for 実装 ${implementations.length} 件すべて OK`,
);
