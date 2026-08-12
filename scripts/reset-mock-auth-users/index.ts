#!/usr/bin/env -S tsx
// mock-auth-server の User Fixture（mock-auth-server/fixtures/users.json）を中立な既定内容へ
// 上書きする。詳細は scripts/README.md の reset-mock-auth-users の行を参照。
//
//
// 実行例:
//   make reset-mock-auth-users
//   make reset-mock-auth-users-ci

import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { DEFAULT_USERS, renderFixture } from "./fixture";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const target = resolve(repoRoot, "mock-auth-server/fixtures/users.json");

mkdirSync(dirname(target), { recursive: true });
writeFileSync(target, renderFixture(DEFAULT_USERS));

console.log(`reset mock-auth-server fixture: ${target} (${DEFAULT_USERS.length} user)`);
