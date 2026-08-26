#!/usr/bin/env -S tsx
// docs/ の走査結果と manifest の meta からビューアーが読む docs/portal/docs.json を生成する。

import { existsSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import yaml from "js-yaml";

import {
  buildDocsJson,
  type DiscoveredDirectory,
  type DiscoveredDocs,
  isMarkdownFile,
  isSectionDirectory,
  sortSectionNames,
} from "./docs-json";

const DOCS_DIR = "docs";
const MANIFEST_PATH = "docs/portal/manifest.yaml";
const OUTPUT_PATH = "docs/portal/docs.json";

function markdownIn(directory: string): string[] {
  return existsSync(directory) ? readdirSync(directory).filter(isMarkdownFile).sort() : [];
}

/** 対訳は正本と同じディレクトリに居る。分けるのは接尾辞であって置き場所ではない。 */
function markdownByLanguage(directory: string): { en: string[]; ja: string[] } {
  const files = markdownIn(directory);

  return { en: files.filter((file) => !file.endsWith(".ja.md")), ja: files.filter((file) => file.endsWith(".ja.md")) };
}

function discover(): DiscoveredDocs {
  const sectionNames = readdirSync(DOCS_DIR, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && isSectionDirectory(entry.name))
    .map((entry) => entry.name);

  const directories: DiscoveredDirectory[] = sortSectionNames(sectionNames).map((name) => {
    const byLanguage = markdownByLanguage(join(DOCS_DIR, name));

    return {
      name,
      hasIndexHtml: existsSync(join(DOCS_DIR, name, "index.html")),
      enFiles: byLanguage.en,
      jaFiles: byLanguage.ja,
    };
  });
  const root = markdownByLanguage(DOCS_DIR);

  return { directories, rootEnFiles: root.en, rootJaFiles: root.ja };
}

function main(): void {
  const manifest = existsSync(MANIFEST_PATH) ? yaml.load(readFileSync(MANIFEST_PATH, "utf8")) : {};
  const { docs, warnings } = buildDocsJson(manifest, discover());

  for (const warning of warnings) {
    console.warn(`⚠ ${warning}`);
  }

  writeFileSync(OUTPUT_PATH, `${JSON.stringify(docs, null, 2)}\n`);

  console.log(`✅ ${OUTPUT_PATH} を生成しました（group ${docs.groups.length} 件）`);
}

try {
  main();
} catch (e) {
  console.error(`❌ ${e instanceof Error ? e.message : String(e)}`);
  process.exit(1);
}
