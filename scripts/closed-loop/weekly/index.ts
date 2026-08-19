#!/usr/bin/env -S tsx
// Feedback Issue を集めて検討課題の順に並べる、週次の統合機構。
//
// 判定は score.ts / issue.ts が持ち、ここは gh の呼び出し・引数・出力・終了コードだけを担う。
//
// 期間は絶対日付で受け取る（--from / --to）。Actions が落ちた週を後から取り直せることが
// 目的なので、「いつ基準の 1 週間か」が実行時刻に依存してはならない。
//
// 使い方:
//   tsx scripts/closed-loop/weekly [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--json]

import { execFileSync } from "node:child_process";

import { parseObservation } from "../issue";
import { resolvePeriod } from "../report";
import { clusterIssues, failureRate, mergeWaitSec, waitDominated, type FeedbackIssue, type FindingKind } from "../score";

const KINDS: readonly FindingKind[] = [
  "skill",
  "architecture",
  "documentation",
  "tooling",
  "ai-misread",
  "ci",
  "developer-experience",
];

function flag(name: string): string | undefined {
  const i = process.argv.indexOf(`--${name}`);
  return i >= 0 ? process.argv[i + 1] : undefined;
}

type RawIssue = { number: number; title: string; body: string; createdAt: string; labels: { name: string }[] };

const period = resolvePeriod(flag("from"), flag("to"), Math.floor(Date.now() / 1000));

// `gh search` サブコマンドは REST の検索バケット（30/h）を消費する。list は GraphQL 側
// （5,000/h）なのでこちらを使う。
const raw: RawIssue[] = JSON.parse(
  execFileSync(
    "gh",
    ["issue", "list", "--label", "feedback", "--state", "all", "--limit", "200", "--json", "number,title,body,createdAt,labels"],
    { encoding: "utf8" },
  ),
);

const issues: FeedbackIssue[] = [];
let unparsed = 0;
for (const r of raw) {
  const at = Math.floor(Date.parse(r.createdAt) / 1000);
  if (at < period.from || at > period.to) continue;
  const observation = parseObservation(r.body);
  if (observation === undefined) {
    // 人が手で壊した issue 1 件で週次全体を落とさない。件数だけ報告する。
    unparsed += 1;
    continue;
  }
  const kinds = r.labels
    .map((l) => l.name)
    .filter((n) => n.startsWith("feedback/"))
    .map((n) => n.slice("feedback/".length))
    .filter((n): n is FindingKind => (KINDS as readonly string[]).includes(n));
  issues.push({ number: r.number, kinds, observation });
}

const clusters = clusterIssues(issues);
const waiting = waitDominated(issues);

if (process.argv.includes("--json")) {
  console.log(JSON.stringify({ period, issues: issues.length, unparsed, clusters, waitDominated: waiting }, null, 2));
  process.exit(0);
}

const asDate = (e: number) => new Date(e * 1000).toISOString().slice(0, 10);
console.log(`期間 ${asDate(period.from)} 〜 ${asDate(period.to)}`);
console.log(`Feedback Issue ${issues.length} 件${unparsed > 0 ? `（解析できず ${unparsed} 件）` : ""}`);

if (clusters.length > 0) {
  console.log("\n検討課題（スコア降順）");
  for (const c of clusters) {
    console.log(
      `  [${c.score.toString().padStart(4)}] ${c.key}  件数${c.frequency} 影響${c.impact} 介入${c.humanIntervention} ${c.isRecurring ? "反復" : "単発"}`,
    );
    console.log(`         ${c.issues.map((n) => `#${n}`).join(" ")}`);
  }
}

if (waiting.length > 0) {
  console.log(`\n待ち時間が実装時間を上回る窓: ${waiting.map((n) => `#${n}`).join(" ")}`);
  console.log("  実装を速くしても縮まない。レビューとマージの経路を見ること");
}

console.log("\n窓ごとの実測");
for (const i of issues) {
  const fr = failureRate(i.observation);
  const wait = mergeWaitSec(i.observation);
  console.log(
    `  #${i.number} ${i.observation.branch ?? "-"}` +
      `  失敗${fr === undefined ? "—" : `${(fr * 100).toFixed(1)}%`}` +
      ` 中断${i.observation.interrupts ?? "—"}` +
      ` 待ち${wait === undefined ? "—" : `${Math.round(wait / 60)}分`}`,
  );
}
