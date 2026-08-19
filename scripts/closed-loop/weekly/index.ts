#!/usr/bin/env -S tsx
// Feedback Issue を集めて検討課題の順に並べる、週次の統合機構。
//
// 期間は絶対日付で受け取る（--from / --to）。Actions が落ちた週を後から取り直せることが
// 目的なので、「いつ基準の 1 週間か」が実行時刻に依存してはならない。
//
// 使い方:
//   tsx scripts/closed-loop/weekly [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--json]

import { execFileSync } from "node:child_process";

import { parseObservation, parseSections, IMPROVEMENT_SECTION } from "../issue";
import { resolvePeriod, withinPeriod } from "../report";
import {
  clusterIssues,
  failureRate,
  labelsToKinds,
  mergeWaitSec,
  reevaluations,
  waitDominated,
  REEVALUATION_DAYS,
  type FeedbackIssue,
} from "../score";

function flag(name: string): string | undefined {
  const i = process.argv.indexOf(`--${name}`);
  return i >= 0 ? process.argv[i + 1] : undefined;
}

type RawIssue = {
  number: number;
  title: string;
  body: string;
  createdAt: string;
  closedAt: string | null;
  labels: { name: string }[];
};

const epoch = (iso: string | null): number | undefined => {
  if (iso === null) return undefined;
  const n = Math.floor(Date.parse(iso) / 1000);
  return Number.isFinite(n) ? n : undefined;
};

const period = resolvePeriod(flag("from"), flag("to"), Math.floor(Date.now() / 1000));

// `gh search` サブコマンドは REST の検索バケット（30/h）を消費する。list は GraphQL 側
// （5,000/h）なのでこちらを使う。
const raw: RawIssue[] = JSON.parse(
  execFileSync(
    "gh",
    [
      "issue",
      "list",
      "--label",
      "feedback",
      "--state",
      "all",
      "--limit",
      "200",
      "--json",
      "number,title,body,createdAt,closedAt,labels",
    ],
    { encoding: "utf8" },
  ),
);

// 期間の内と外を両方組み立てる。検討課題は期間内で並べるが、測り直しは「期間より前に閉じた
// 改善が、期間内で再発したか」を問うので、期間外の Issue が母数から落ちると答えが出せない。
const issues: FeedbackIssue[] = [];
const all: FeedbackIssue[] = [];
let unparsed = 0;
for (const r of raw) {
  const createdAt = epoch(r.createdAt);
  const observation = parseObservation(r.body);
  if (observation === undefined) {
    // 人が手で壊した issue 1 件で週次全体を落とさない。件数だけ報告する。
    unparsed += 1;
    continue;
  }
  const issue: FeedbackIssue = {
    number: r.number,
    kinds: labelsToKinds(r.labels.map((l) => l.name)),
    observation,
    createdAt,
    resolvedAt: epoch(r.closedAt),
    sections: parseSections(r.body),
  };
  all.push(issue);
  if (createdAt !== undefined && withinPeriod(createdAt, period)) issues.push(issue);
}

const clusters = clusterIssues(issues);
const waiting = waitDominated(issues);
const remeasured = reevaluations(all, period.to);

if (process.argv.includes("--json")) {
  console.log(
    JSON.stringify(
      { period, issues: issues.length, unparsed, clusters, waitDominated: waiting, reevaluations: remeasured },
      null,
      2,
    ),
  );
  process.exit(0);
}

const asDate = (e: number) => new Date(e * 1000).toISOString().slice(0, 10);
console.log(`期間 ${asDate(period.from)} 〜 ${asDate(period.to)}`);
console.log(`Feedback Issue ${issues.length} 件${unparsed > 0 ? `（解析できず ${unparsed} 件）` : ""}`);

if (clusters.length > 0) {
  console.log("\n検討課題（スコア降順）");
  const byNumber = new Map(issues.map((i) => [i.number, i]));
  for (const c of clusters) {
    console.log(
      `  [${c.score.toString().padStart(4)}] ${c.key}  件数${c.frequency} 影響${c.impact} 介入${c.humanIntervention} ${c.isRecurring ? "反復" : "単発"}`,
    );
    console.log(`         ${c.issues.map((n) => `#${n}`).join(" ")}`);
    // 順位だけでなく提案そのものを並べる。GitHub を開かずにレトロの議題が読めるようにする
    // （scripts/README.md）。
    for (const n of c.issues) {
      const proposal = byNumber.get(n)?.sections?.[IMPROVEMENT_SECTION];
      if (proposal === undefined) continue;
      for (const line of proposal.split("\n")) console.log(`         #${n} ${line}`);
    }
  }
}

if (remeasured.length > 0) {
  console.log(`\n測り直し（着地から ${REEVALUATION_DAYS} 日後に判定する）`);
  for (const r of remeasured) {
    const when = asDate(r.landedAt);
    const since = r.recurred.length === 0 ? "再発なし" : `再発 ${r.recurred.map((n) => `#${n}`).join(" ")}`;
    console.log(`  ${r.key}  #${r.landedIssue} を ${when} にクローズ → ${since}${r.due ? "" : "（判定はまだ早い）"}`);
  }
  console.log("  効いたかを決めるのはここではない。保持 / 簡素化 / 撤回はレトロで決めること");
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
