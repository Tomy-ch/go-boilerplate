#!/usr/bin/env -S tsx
// fork 後も残る文書に「fork した瞬間に偽になる前提」が書かれていないかを検査する。
//
// 規則の出所は docs/rules.md の Documentation Rules「No premise the document will outlive」。
// 判定（対象の選別・マーカー除去・言い回しの照合）は rules.ts、許容の宣言は allowances.ts が持ち、
// ここはファイル列挙と終了コードだけを担う。
//
// 使い方:
//   tsx scripts/premise-lint

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { ALLOWANCES } from "./allowances";
import { type Finding, inspect, isChecked, survivingText } from "./rules";

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

/** 走査から外すディレクトリ名。依存の取得物と VCS の内部。 */
const EXCLUDED_DIRECTORIES = new Set([".git", "node_modules", "vendor"]);

/** 走査から外す相対パス接頭辞。いずれも生成物で、直しても再生成で戻る。 */
const EXCLUDED_PREFIXES = ["docs/portal/guides/", "docs/coverage/", "docs/db-schema/", "docs/godoc/"];

function listMarkdown(dir: string, acc: string[] = []): string[] {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    const full = path.join(dir, entry.name);
    const rel = path.relative(REPO_ROOT, full).split(path.sep).join("/");

    if (entry.isDirectory()) {
      if (EXCLUDED_DIRECTORIES.has(entry.name)) continue;
      if (EXCLUDED_PREFIXES.some((prefix) => `${rel}/`.startsWith(prefix))) continue;
      listMarkdown(full, acc);
      continue;
    }
    if (entry.isFile() && isChecked(rel)) acc.push(rel);
  }

  return acc;
}

const findings: Finding[] = [];

for (const rel of listMarkdown(REPO_ROOT)) {
  findings.push(...inspect(rel, survivingText(fs.readFileSync(path.join(REPO_ROOT, rel), "utf8")), ALLOWANCES));
}

if (findings.length === 0) {
  console.log("✓ premise-lint: fork 後も残る文書に失効予定の前提はありません");
  process.exit(0);
}

console.error(`✗ premise-lint: ${findings.length} 件\n`);
for (const finding of findings) {
  console.error(`  ${finding.file}:${finding.line}`);
  console.error(`    ${finding.text.trim().slice(0, 140)}`);
}
console.error(
  "\n前提を書いてよいのは README / docs/get-started/** と、マーカーで囲った領域だけです。" +
    "\n別語義であれば scripts/premise-lint/allowances.ts へ理由付きで宣言してください。",
);
process.exit(1);
