// mock-auth-server の固定 User Fixture（docker/mock-auth-server/fixtures/users.json）を
// 中立な既定内容へリセット（上書き）する。users.json 自体は削除しないため mock は常に起動可能。
//
// 用途: `make setup-remove-sample-api` がこのスクリプトを呼び、デモの固定ユーザー（John Doe 等）を
// 中立な既定ユーザー 1 件へ上書きする（サンプル identity をデータから除く）。サンプル削除処理
// （remove-sample-api.mjs）とは独立した「リセット」ユーティリティで、任意のタイミングで単独実行
// しても users.json を既定へ戻せる。
//
// 実行例:
//   node scripts/setup/reset-mock-auth-users.mjs
//   docker compose run --rm node_tool_runner node scripts/setup/reset-mock-auth-users.mjs
import { writeFileSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const target = resolve(repoRoot, "docker/mock-auth-server/fixtures/users.json");

// 中立な既定ユーザー（サンプル固有の John Doe 等は含めない）。
const defaultUsers = [
  {
    subject: "user-example",
    email: "user@example.com",
    given_name: "Example",
    family_name: "User",
    name: "Example User",
    status: "active",
  },
];

mkdirSync(dirname(target), { recursive: true });
writeFileSync(target, `${JSON.stringify(defaultUsers, null, 2)}\n`);

console.log(
  `reset mock-auth-server fixture: ${target} (${defaultUsers.length} user)`,
);
