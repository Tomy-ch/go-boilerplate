#!/usr/bin/env -S tsx
// 開発の窓の打刻を読み、フェーズ区間と異常を報告する。
//
// 判定は windows.ts が持ち、ここはディレクトリ走査・引数・出力・終了コードだけを担う。
//
// 使い方:
//   tsx scripts/closed-loop [--json]

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { anomaliesOf, phasesOf, stampCount, toWindow, totalDurationSec, MARK_ORDER, type Window } from "./windows";

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const MARKS_DIR = path.join(REPO_ROOT, "tmp", "closed-loop", "marks");

function readWindows(): Window[] {
  if (!fs.existsSync(MARKS_DIR)) return [];
  return fs
    .readdirSync(MARKS_DIR, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => e.name)
    .sort()
    .map((id) => {
      const dir = path.join(MARKS_DIR, id);
      const files: Record<string, string> = {};
      for (const f of fs.readdirSync(dir)) {
        try {
          files[f] = fs.readFileSync(path.join(dir, f), "utf8");
        } catch {
          // 読めないファイルは無いものとして扱う。窓の報告を 1 ファイルで落とさない。
        }
      }
      return toWindow(id, files);
    });
}

const windows = readWindows();

if (process.argv.includes("--json")) {
  const payload = windows.map((w) => ({
    id: w.id,
    marks: Object.fromEntries(MARK_ORDER.filter((m) => stampCount(w, m) > 0).map((m) => [m, w.marks[m]])),
    phases: phasesOf(w),
    totalDurationSec: totalDurationSec(w) ?? null,
    anomalies: anomaliesOf(w),
  }));
  console.log(JSON.stringify(payload, null, 2));
  process.exit(0);
}

if (windows.length === 0) {
  console.log("closed-loop: 打刻された窓はありません");
  process.exit(0);
}

for (const w of windows) {
  const total = totalDurationSec(w);
  console.log(`\n窓 ${w.id}${total === undefined ? "" : `  合計 ${total}s`}`);
  for (const phase of phasesOf(w)) {
    console.log(`  ${phase.from} → ${phase.to}: ${phase.durationSec}s`);
  }
  for (const mark of MARK_ORDER) {
    const count = stampCount(w, mark);
    if (count > 1) console.log(`  ${mark} は ${count} 回打刻されている`);
  }
  for (const anomaly of anomaliesOf(w)) {
    console.log(`  ⚠ ${anomaly.kind}: ${anomaly.detail}`);
  }
}
console.log(`\n窓 ${windows.length} 件`);
