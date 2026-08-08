// リポジトリを走査してマーカー行の分布を作る。判定は rules.ts、ここはファイル読みだけを担う。

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { type Baseline, EXCLUDED_DIRECTORIES, countMarkerLines, isBaselineTarget } from "./rules";

export const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

export const BASELINE_PATH = path.join(REPO_ROOT, "scripts/marker-baseline/baseline.json");

function walk(dir: string, acc: string[]): string[] {
  for (const entry of fs
    .readdirSync(dir, { withFileTypes: true })
    .sort((a, b) => a.name.localeCompare(b.name))) {
    const full = path.join(dir, entry.name);
    const rel = path.relative(REPO_ROOT, full).split(path.sep).join("/");

    if (entry.isDirectory()) {
      if (EXCLUDED_DIRECTORIES.has(entry.name)) continue;
      if (!isBaselineTarget(`${rel}/`)) continue;
      walk(full, acc);
      continue;
    }
    if (entry.isFile() && isBaselineTarget(rel)) acc.push(rel);
  }

  return acc;
}

/** 実ツリーのマーカー行分布。キーは相対パスの昇順。 */
export function scanRepository(): Baseline {
  const found: Array<[string, number]> = [];

  for (const rel of walk(REPO_ROOT, [])) {
    let content: string;

    try {
      content = fs.readFileSync(path.join(REPO_ROOT, rel), "utf8");
    } catch {
      continue;
    }

    const count = countMarkerLines(content);

    if (count > 0) found.push([rel, count]);
  }

  return Object.fromEntries(found.sort(([a], [b]) => a.localeCompare(b)));
}

/** コミット済みのベースライン。 */
export function readBaseline(): Baseline {
  return JSON.parse(fs.readFileSync(BASELINE_PATH, "utf8")) as Baseline;
}
