#!/usr/bin/env -S tsx
// タグ名から次のリリースバージョン候補を出力する。
//
// 使い方:
//   tsx scripts/semver <version> <patch|minor|major>

import { bumpVersion, parseArgs } from "./semver";

function main(): void {
  const parsed = parseArgs(process.argv.slice(2));

  if (!parsed.ok) {
    console.error(parsed.error);
    process.exit(1);
  }

  try {
    console.log(bumpVersion(parsed.version, parsed.type));
  } catch (e) {
    console.error(e instanceof Error ? e.message : String(e));
    process.exit(1);
  }
}

main();
