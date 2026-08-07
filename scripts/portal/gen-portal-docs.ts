#!/usr/bin/env -S tsx
// docs/portal/manifest.yaml に登録された canonical ドキュメントを docs/portal/guides/ へ複製する。

import { copyFileSync, existsSync, mkdirSync, readFileSync, rmSync } from "node:fs";
import { dirname, resolve } from "node:path";
import yaml from "js-yaml";

import {
  assertUniqueDestinations,
  assertWithinOutputRoot,
  assertWithinRepositoryRoot,
  findMissingSources,
  formatMissingSources,
  resolveCopyEntries,
} from "./portal-manifest";

const MANIFEST_PATH = "docs/portal/manifest.yaml";
const OUTPUT_ROOT = "docs/portal/guides";

function main(): void {
  if (!existsSync(MANIFEST_PATH)) {
    console.error(`❌ manifest がありません: ${MANIFEST_PATH}`);
    process.exit(1);
  }

  const entries = resolveCopyEntries(yaml.load(readFileSync(MANIFEST_PATH, "utf8")));

  assertUniqueDestinations(entries);
  assertWithinOutputRoot(entries, OUTPUT_ROOT, resolve);
  assertWithinRepositoryRoot(entries, ".", resolve);

  const missing = findMissingSources(entries, existsSync);

  if (missing.length) {
    console.error("❌ manifest が参照する src が見つかりません（manifest の陳腐化を解消してください）:");
    console.error(formatMissingSources(missing));
    process.exit(1);
  }

  console.log("🧹 Cleaning output directory...");
  rmSync(OUTPUT_ROOT, { force: true, recursive: true });
  mkdirSync(OUTPUT_ROOT, { recursive: true });

  console.log("🚀 Generating docs from manifest...");

  for (const entry of entries) {
    mkdirSync(dirname(entry.dst), { recursive: true });
    copyFileSync(entry.src, entry.dst);
    console.log(`✔ [${entry.section}] ${entry.src} -> ${entry.dst}`);
  }

  console.log(`✅ ${entries.length} 件を ${OUTPUT_ROOT} へ複製しました`);
}

try {
  main();
} catch (e) {
  console.error(`❌ ${e instanceof Error ? e.message : String(e)}`);
  process.exit(1);
}
