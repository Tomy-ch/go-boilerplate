#!/usr/bin/env -S tsx
// `.makefiles/**/*.mk` の宣言から `make help` の一覧を組み立てて出力する。

import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

import { type MakefileSource, renderHelp } from "./help";

const MAKEFILES_DIR = ".makefiles";

// `.makefiles` 配下の *.mk をフルパス昇順で列挙する。
function collectMakefiles(dir: string): string[] {
  const files: string[] = [];

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const entryPath = join(dir, entry.name);

    if (entry.isDirectory()) {
      files.push(...collectMakefiles(entryPath));
    } else if (entry.isFile() && entry.name.endsWith(".mk")) {
      files.push(entryPath);
    }
  }

  return files;
}

function main(): void {
  const sources: MakefileSource[] = collectMakefiles(MAKEFILES_DIR)
    .sort()
    .map((path) => ({ path, content: readFileSync(path, "utf8") }));

  const { lines, undocumented } = renderHelp(sources);

  console.log(lines.join("\n"));

  for (const entry of undocumented) {
    console.error(`⚠️  ${entry} — 説明コメント(## ...)が無いため一覧から除外しました`);
  }
}

main();
