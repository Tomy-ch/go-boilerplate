#!/usr/bin/env -S tsx
// mock-auth-server の固定 User Fixture（mock-auth-server/fixtures/users.json）を
// 中立な既定内容へリセット（上書き）する。users.json 自体は削除しないため mock は常に起動可能。
//
// 用途: `make setup-remove-sample-api` がこのスクリプトを呼び、デモの固定ユーザー（John Doe 等）を中立な既定ユーザー 1 件へ上書きする。
//
// 実行例:
//   make reset-mock-auth-users
//   make reset-mock-auth-users-ci

import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { DEFAULT_USERS, renderFixture } from "./lib/mock-auth-fixture";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const target = resolve(repoRoot, "mock-auth-server/fixtures/users.json");

mkdirSync(dirname(target), { recursive: true });
writeFileSync(target, renderFixture(DEFAULT_USERS));

console.log(`reset mock-auth-server fixture: ${target} (${DEFAULT_USERS.length} user)`);
