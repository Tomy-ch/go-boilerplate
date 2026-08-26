#!/usr/bin/env -S tsx
// 開発の窓とセッションを読み、期間ぶんの決定論的な事実を報告する。
//
// トランスクリプトは利用者のホーム配下にあり、tool-runner コンテナからは見えない。だから
// 読み先は既定を持たせず --transcripts で明示的に渡す。窓を閉じるときに送出する経路は
// ホストで動くのでそこから渡し、渡されなければ窓の報告だけを出す。
//
// 使い方:
//   tsx scripts/closed-loop [--from YYYY-MM-DD] [--to YYYY-MM-DD]
//                           [--transcripts <dir>] [--codex <dir>] [--json]

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { parseClaudeLine, parseCodexLine, summarizeSession, type Event, type SessionFacts } from "./events";
import { resolvePeriod, summarizePeriod, uncalledSkills } from "./report";
import {
  anomaliesOf,
  collectWindows,
  phasesOf,
  stampCount,
  totalDurationSec,
  MARK_ORDER,
  type MarksReader,
} from "./windows";

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const MARKS_DIR = path.join(REPO_ROOT, "tmp", "closed-loop", "marks");
const SKILLS_DIR = path.join(REPO_ROOT, ".claude", "skills");

function flag(name: string): string | undefined {
  const i = process.argv.indexOf(`--${name}`);
  return i >= 0 ? process.argv[i + 1] : undefined;
}

const marksReader: MarksReader = {
  listWindowIds: () =>
    fs.existsSync(MARKS_DIR)
      ? fs
          .readdirSync(MARKS_DIR, { withFileTypes: true })
          .filter((e) => e.isDirectory())
          .map((e) => e.name)
      : [],
  listFiles: (id) => fs.readdirSync(path.join(MARKS_DIR, id)),
  readFile: (id, file) => fs.readFileSync(path.join(MARKS_DIR, id, file), "utf8"),
};

function listJsonl(dir: string): string[] {
  const found: string[] = [];
  const walk = (d: string) => {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const full = path.join(d, e.name);
      if (e.isDirectory()) walk(full);
      else if (e.isFile() && e.name.endsWith(".jsonl")) found.push(full);
    }
  };
  walk(dir);
  return found.sort((a, b) => a.localeCompare(b));
}

function readSessions(dir: string, parse: (line: string) => Event[], client: "claude" | "codex"): SessionFacts[] {
  if (!fs.existsSync(dir)) return [];
  return listJsonl(dir).map((file) => {
    const events: Event[] = [];
    try {
      for (const line of fs.readFileSync(file, "utf8").split("\n")) events.push(...parse(line));
    } catch {
      // 読めないファイルは空のセッションとして扱う。1 件で期間全体を落とさない。
    }
    return summarizeSession(client, events);
  });
}

const declaredSkills = fs.existsSync(SKILLS_DIR)
  ? fs
      .readdirSync(SKILLS_DIR, { withFileTypes: true })
      .filter((e) => e.isDirectory())
      .map((e) => e.name)
      .sort()
  : [];

const period = resolvePeriod(flag("from"), flag("to"), Math.floor(Date.now() / 1000));
const windows = collectWindows(marksReader);
const claudeDir = flag("transcripts");
const codexDir = flag("codex");
const sessions = [
  ...(claudeDir === undefined ? [] : readSessions(claudeDir, parseClaudeLine, "claude")),
  ...(codexDir === undefined ? [] : readSessions(codexDir, parseCodexLine, "codex")),
];
const summary = summarizePeriod(sessions, period);
const uncalled = uncalledSkills(declaredSkills, summary);

if (process.argv.includes("--json")) {
  console.log(
    JSON.stringify(
      {
        summary,
        uncalledSkills: uncalled,
        windows: windows.map((w) => ({
          id: w.id,
          phases: phasesOf(w),
          totalDurationSec: totalDurationSec(w) ?? null,
          anomalies: anomaliesOf(w),
        })),
      },
      null,
      2,
    ),
  );
  process.exit(0);
}

const asDate = (epoch: number) => new Date(epoch * 1000).toISOString().slice(0, 10);
console.log(`期間 ${asDate(period.from)} 〜 ${asDate(period.to)}`);

if (sessions.length === 0) {
  console.log("セッション記録は渡されていません（--transcripts / --codex で指定する）");
} else {
  console.log(`\nセッション ${summary.sessions} 件 ${JSON.stringify(summary.byClient)}`);
  console.log(`  人の発話 ${summary.prompts} / ツール呼出 ${summary.toolCalls} / 中断 ${summary.interrupts} / compact ${summary.compactions}`);
  console.log(
    summary.toolFailures === undefined
      ? "  ツール失敗: 観測できず"
      : `  ツール失敗 ${summary.toolFailures} (${((summary.toolFailureRate ?? 0) * 100).toFixed(1)}%)`,
  );
  if (summary.skillCalls === undefined) {
    console.log("  スキル呼出: 観測できず");
  } else {
    const top = Object.entries(summary.skillCalls).sort((a, b) => b[1] - a[1]);
    console.log(`  スキル呼出 ${top.reduce((n, [, c]) => n + c, 0)} 件 / ${top.length} 種`);
    for (const [name, count] of top.slice(0, 8)) console.log(`    ${name}: ${count}`);
    if (uncalled.length > 0) {
      console.log(`  この期間に呼ばれなかったスキル ${uncalled.length} / ${declaredSkills.length}`);
      console.log(`    ${uncalled.join(" ")}`);
      console.log("    ※ 呼出ゼロだけでは削除の根拠にならない。skill-meta.yaml の Usage Class と併せて判断すること");
    }
  }
}

if (windows.length > 0) {
  console.log(`\n窓 ${windows.length} 件`);
  for (const w of windows) {
    const total = totalDurationSec(w);
    console.log(`  ${w.id}${total === undefined ? "" : `  合計 ${total}s`}`);
    for (const phase of phasesOf(w)) console.log(`    ${phase.from} → ${phase.to}: ${phase.durationSec}s`);
    for (const mark of MARK_ORDER) {
      const count = stampCount(w, mark);
      if (count > 1) console.log(`    ${mark} は ${count} 回打刻されている`);
    }
    for (const a of anomaliesOf(w)) console.log(`    ⚠ ${a.kind}: ${a.detail}`);
  }
}
