#!/usr/bin/env -S tsx
import fs from "node:fs";
import path from "node:path";

import { formatFindings } from "../lib/lint-report";
import { selectWorkflowFiles } from "../lib/workflow";
import { checkRequiredChecks } from "./required-checks";

const rulesetPath = ".github/settings/branch-protection.json";
const workflowsDir = ".github/workflows";
const ruleset = JSON.parse(fs.readFileSync(rulesetPath, "utf8")) as {
  rules: { type: string; parameters?: { required_status_checks?: { context: string }[] } }[];
};
const statusRule = ruleset.rules.find((rule) => rule.type === "required_status_checks");
const contexts = statusRule?.parameters?.required_status_checks?.map(({ context }) => context) ?? [];
const workflows = selectWorkflowFiles(fs.readdirSync(workflowsDir), workflowsDir).map((file) => ({
  file,
  source: fs.readFileSync(path.join(process.cwd(), file), "utf8"),
}));

if (contexts.length === 0) {
  console.error(`✘ required-check-lint: ${rulesetPath} に required status check がありません`);
  process.exit(2);
}

const findings = checkRequiredChecks(contexts, workflows);
if (findings.length > 0) {
  console.error(`✘ required-check-lint: ${findings.length} 件の不整合\n`);
  console.error(formatFindings(findings));
  process.exit(1);
}

console.log(`✓ required-check-lint: ${contexts.length} required contexts と本体 / guard job の対応すべて OK`);
