#!/usr/bin/env -S tsx
// bundle 済み OpenAPI（openapi/openapi.gen.yaml）から frontend generator（orval）で client を生成し、
// Realtime Delivery の contract として公開している component 型が生成物に現れることを確認する。
//
// 判定は generated-types.ts が持ち、ここは orval の起動・生成物の読み取り・終了コードだけを担う。
// 生成物は tmp/openapi-client/ に出し、コミットしない。
//
// 使い方:
//   tsx scripts/openapi-client-check

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { generate } from "orval";

import { DegenerateOutputError, missingTypes } from "./generated-types";

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const INPUT = path.join(REPO_ROOT, "openapi/openapi.gen.yaml");
const OUTPUT_DIR = path.join(REPO_ROOT, "tmp/openapi-client");

/** 生成物に現れなければならない component 型。SSE の契約（DeliveryEvent / ControlEvent / StreamCursor）。 */
const EXPECTED_TYPES = ["DeliveryEvent", "ControlEvent", "StreamCursor"] as const;

fs.rmSync(OUTPUT_DIR, { recursive: true, force: true });
fs.mkdirSync(OUTPUT_DIR, { recursive: true });

await generate(
  {
    client: {
      input: { target: INPUT },
      output: { target: path.join(OUTPUT_DIR, "api.ts"), client: "fetch", mode: "single", clean: true },
    },
  },
  REPO_ROOT,
);

const sources = fs
  .readdirSync(OUTPUT_DIR, { recursive: true, withFileTypes: true })
  .filter((entry) => entry.isFile() && entry.name.endsWith(".ts"))
  .map((entry) => fs.readFileSync(path.join(entry.parentPath, entry.name), "utf8"));

let findings;
try {
  findings = missingTypes(sources, EXPECTED_TYPES);
} catch (error) {
  if (error instanceof DegenerateOutputError) {
    console.error(`✗ openapi-client-check: ${error.message}`);
    process.exit(2);
  }
  throw error;
}

if (findings.length === 0) {
  console.log(`✓ openapi-client-check: ${EXPECTED_TYPES.length} 件の component 型を orval が生成できました`);
  process.exit(0);
}

console.error(`✗ openapi-client-check: ${findings.length} 件\n`);
for (const finding of findings) {
  console.error(`  ${finding.expected}: ${finding.reason}`);
}
process.exit(1);
