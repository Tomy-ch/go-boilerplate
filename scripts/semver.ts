#!/usr/bin/env -S tsx
// タグ名から次のリリースバージョン候補を出力する。
//
// 使い方:
//   tsx scripts/semver.ts <version> <patch|minor|major>

import { bumpVersion, isBumpType } from "./lib/semver";

function main(): void {
  const [version, type] = process.argv.slice(2);

  if (version === undefined || version === "") {
    console.error("version is required");
    process.exit(1);
  }

  if (type === undefined || !isBumpType(type)) {
    console.error("type must be patch | minor | major");
    process.exit(1);
  }

  try {
    console.log(bumpVersion(version, type));
  } catch (e) {
    console.error(e instanceof Error ? e.message : String(e));
    process.exit(1);
  }
}

main();
