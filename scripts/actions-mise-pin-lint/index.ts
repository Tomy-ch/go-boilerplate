import { readFileSync } from "node:fs";
import path from "node:path";

import { findViolations, readPin } from "./pin-consistency.js";

const ACTION_FILE = ".github/actions/setup-mise/action.yaml";

function main(): void {
  const file = path.join(process.cwd(), ACTION_FILE);

  let source: string;
  try {
    source = readFileSync(file, "utf8");
  } catch {
    abort(`${ACTION_FILE} を読めません（リポジトリルートで実行してください）`);
  }

  const violations = findViolations(readPin(source));
  if (violations.length > 0) {
    console.error(`❌ ${ACTION_FILE} の版 / digest / キャッシュキーが揃っていません`);
    for (const violation of violations) console.error(`   ${violation}`);
    process.exit(1);
  }

  console.log("✅ mise の版 / digest / キャッシュキーが揃っています");
}

function abort(message: string): never {
  console.error(`❌ ${message}`);
  process.exit(2);
}

main();
