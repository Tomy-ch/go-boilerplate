#!/usr/bin/env -S tsx
// Mermaid のコードフェンス（```mermaid）を実際の mermaid パーサで構文検証する lint スクリプト
//（塞いでいる穴は scripts/README.md の mermaid-lint の行）。
//
// mermaid.parse は DOMPurify サニタイズで DOM を要求するため、mermaid のロードには DOM 環境が要る。
// 1 つでも壊れた図があれば非 0 で終了する。

import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

import { extractMermaidBlocks, isTargetMarkdown, shouldDescend } from "./blocks";
import {
  type Failure,
  errorMessage,
  formatFailures,
  isDependencyMissing,
  summarize,
} from "./diagnostics";

// 本スクリプトが使う mermaid の最小面。mermaid の公開型は DOM 前提で重く、
// ここで必要なのは initialize / parse の 2 つだけなので構造的に絞る。
type MermaidApi = {
  initialize: (config: Record<string, unknown>) => void;
  parse: (text: string) => Promise<unknown>;
};

// 依存ロード（環境セットアップ）。mermaid / linkedom は node_tool_runner コンテナ内の
// scripts/node_modules に入る前提。ここで失敗した場合は mermaid 図の文法問題ではなく
// 「環境未整備」なので、生の ERR_MODULE_NOT_FOUND スタックトレースではなく、原因と対処を
// 明示して exit 2（lint 失敗の exit 1 と区別）で落とす。
async function loadMermaid(): Promise<MermaidApi> {
  try {
    // import 順の都合で mermaid より先に DOM を globalThis へ載せる必要があるため require で先行ロードする。
    const require = createRequire(import.meta.url);
    const { parseHTML } = require("linkedom");

    const { window, document } = parseHTML("<!doctype html><html><head></head><body></body></html>");
    // globalThis の window / document 等は DOM lib で読み取り専用のため、
    // 代入するには構造を持たない袋として扱う必要がある。
    const g = globalThis as unknown as Record<string, unknown>;
    g.window = window;
    g.document = document;
    Object.defineProperty(globalThis, "navigator", { value: window.navigator, configurable: true });
    g.location = window.location;
    g.requestAnimationFrame = (fn: () => void) => setTimeout(fn, 0);
    g.MutationObserver = window.MutationObserver;

    const mermaidModule = await import("mermaid");
    const mermaid = (mermaidModule.default ?? mermaidModule) as unknown as MermaidApi;
    // logLevel:5(fatal) でパース失敗時の冗長な内部ログを抑止し、本スクリプトの整形済み出力に一本化する。
    mermaid.initialize({ startOnLoad: false, securityLevel: "loose", logLevel: 5 });
    return mermaid;
  } catch (e) {
    console.error("✘ mermaid-lint: セットアップエラー（mermaid 図の文法問題ではありません）");
    if (isDependencyMissing(e)) {
      console.error("    原因: mermaid / linkedom を解決できません（scripts/node_modules が未整備）。");
      console.error("    対処: この lint は node_tool_runner コンテナ内で動く前提です。");
      console.error("          - 通常は `make md-lint`（コンテナ経由）で実行する（host での直接実行は不可）。");
      console.error("          - イメージが古い場合は `make tool-runners-build`（または `-clean`）で再ビルドする。");
    } else {
      console.error("    原因: 依存ロード中に想定外のエラーが発生しました。");
    }
    console.error(`    詳細: ${errorMessage(e)}`);
    process.exit(2);
  }
}

// repoRoot 配下の *.md を再帰収集する（除外ディレクトリは降りない）。
function collectMarkdown(repoRoot: string): string[] {
  const out: string[] = [];

  const walk = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name);
      const rel = path.relative(repoRoot, abs);

      if (entry.isDirectory()) {
        if (shouldDescend(entry.name, rel)) walk(abs);
        continue;
      }
      if (isTargetMarkdown(rel)) out.push(rel);
    }
  };

  walk(repoRoot);
  return out.sort();
}

async function main(): Promise<void> {
  const mermaid = await loadMermaid();

  const repoRoot = process.cwd();
  const files = collectMarkdown(repoRoot);

  let blockCount = 0;
  let fileWithBlocks = 0;
  const failures: Failure[] = [];
  const skipped: string[] = [];

  for (const rel of files) {
    let content: string;
    try {
      content = fs.readFileSync(path.join(repoRoot, rel), "utf8");
    } catch {
      // 読めないファイル（壊れた symlink や権限エラー）は検証対象外としてスキップする。
      // markdownlint も壊れた symlink を黙って飛ばすため挙動を揃える。
      skipped.push(rel);
      continue;
    }

    const blocks = extractMermaidBlocks(content);
    if (blocks.length > 0) fileWithBlocks++;

    for (let b = 0; b < blocks.length; b++) {
      blockCount++;
      try {
        // mermaid はグローバルな単一インスタンス + 共有 DOM を使うため、parse は並列化せず逐次実行する。
        await mermaid.parse(blocks[b].code);
      } catch (e) {
        failures.push({ rel, startLine: blocks[b].startLine, index: b + 1, msg: errorMessage(e) });
      }
    }
  }

  if (failures.length > 0) {
    console.error(`✘ mermaid-lint: ${failures.length} 件の壊れた mermaid ブロック\n`);
    console.error(formatFailures(failures));
    console.error(
      `検証 ${summarize(blockCount, fileWithBlocks, skipped.length)}中 ${failures.length} 件 NG`,
    );
    process.exit(1);
  }

  console.log(`✓ mermaid-lint: ${summarize(blockCount, fileWithBlocks, skipped.length)} すべて OK`);
}

try {
  await main();
} catch (e: unknown) {
  console.error(`✘ mermaid-lint: 想定外のエラー\n    ${errorMessage(e)}`);
  process.exit(2);
}
