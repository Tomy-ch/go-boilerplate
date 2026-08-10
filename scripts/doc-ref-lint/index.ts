#!/usr/bin/env -S tsx

import fs from "node:fs";
import path from "node:path";
import { ADR_FILE, adrIndex, checkReferences, checkTranslationExclusions, checkTranslations, isEligible, normalizeReferences } from "./rules";

const root = process.cwd();
const write = process.argv.includes("--write");
const skip = new Set([".git", "node_modules", "vendor", "tmp"]);
const walk = (dir: string): string[] => fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
  const absolute = path.join(dir, entry.name);
  const relative = path.relative(root, absolute);
  if (entry.isDirectory()) return skip.has(entry.name) ? [] : walk(absolute);
  return [relative];
});
const files = walk(root);
const adr = adrIndex(files);
const findings: string[] = [];
for (const file of files.filter(isEligible).filter((file) => !file.startsWith("docs/adr/"))) {
  const absolute = path.join(root, file);
  const source = fs.readFileSync(absolute, "utf8");
  const normalized = normalizeReferences(source, adr);
  if (write && normalized !== source) fs.writeFileSync(absolute, normalized);
  findings.push(...checkReferences(file, normalized, adr).map((finding) => `${finding.file}: ${finding.message}`));
}
for (const file of files.filter((file) => ADR_FILE.test(file))) {
  const number = path.basename(file).slice(0, 4);
  if (!new RegExp(`^# ADR-${number}\\b`, "m").test(fs.readFileSync(path.join(root, file), "utf8"))) findings.push(`${file}: ADR filename and H1 disagree`);
}
for (const file of files.filter((file) => /^(internal|pkg)\/.*\/README\.md$/.test(file))) if (!fs.existsSync(path.join(root, path.dirname(file), "README.ja.md"))) findings.push(`${file}: missing README.ja.md`);
findings.push(...checkTranslations(files).map((finding) => `${finding.file}: ${finding.message}`));
findings.push(...checkTranslationExclusions(files).map((finding) => `${finding.file}: ${finding.message}`));
if (findings.length) { console.error(findings.map((item) => `✘ doc-ref-lint: ${item}`).join("\n")); process.exit(1); }
