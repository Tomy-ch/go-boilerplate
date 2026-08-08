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

/**
 * 実ツリーのマーカー行分布。キーは相対パスの昇順。
 *
 * @remarks
 * 読めないファイルを握り潰しません。読めなければマーカー行数は**不明**であり、飛ばすことは
 * 「0 行」と記録するのと同じです。ベースラインに 0 として載れば、そこへ後からマーカーが
 * 増えても差分に出ません——この検査が塞ごうとしている無言の見落としと、同じ形の穴になります。
 * 読めないファイルが現れたなら、それ自体が知るべき事実なので、そのまま投げます。
 */
export function scanRepository(): Baseline {
  const found: Array<[string, number]> = [];

  for (const rel of walk(REPO_ROOT, [])) {
    const count = countMarkerLines(fs.readFileSync(path.join(REPO_ROOT, rel), "utf8"));

    if (count > 0) found.push([rel, count]);
  }

  return Object.fromEntries(found.sort(([a], [b]) => a.localeCompare(b)));
}

/** コミット済みのベースライン。 */
export function readBaseline(): Baseline {
  return JSON.parse(fs.readFileSync(BASELINE_PATH, "utf8")) as Baseline;
}
