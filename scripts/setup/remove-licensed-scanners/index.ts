#!/usr/bin/env -S tsx
import fs from "node:fs";

import { listFilesRecursive, toAbsolutePath, toRelativePath, updateFile } from "../lib/file-utils";
import { type SetupOptions, newSetupCommand } from "../lib/runtime";
import { assertCleanWorktree, commitPaths } from "./git-commit";
import { SCANNER_DOMAINS, type ScannerDomain } from "./scanner-manifest";
import {
  isPinKeyReferenced,
  removeEgressSections,
  removeExact,
  removePinEntries,
  removeSection,
} from "./scanner-removal";

const PIN_FILE = ".github/actions-pin.toml";
const EGRESS_FILE = ".github/egress.toml";
const SCANNED_DIRS = [".github/workflows", ".github/actions"];
const DOCS_COMMIT_SUBJECT = "Docs: 撤去したスキャナの記述を workflows の README から落とす";

function survivingContents(removedPaths: ReadonlySet<string>): string[] {
  return SCANNED_DIRS.map(toAbsolutePath)
    .filter((dir) => fs.existsSync(dir))
    .flatMap((dir) => listFilesRecursive(dir))
    .filter((file) => !removedPaths.has(toRelativePath(file)))
    .map((file) => fs.readFileSync(file, "utf8"));
}

function deletePaths(domain: ScannerDomain, dryRun: boolean): string[] {
  const deleted: string[] = [];

  for (const relativePath of domain.paths) {
    const absolutePath = toAbsolutePath(relativePath);

    if (!fs.existsSync(absolutePath)) {
      continue;
    }

    if (!dryRun) {
      fs.rmSync(absolutePath, { recursive: true, force: true });
    }

    deleted.push(relativePath);
  }

  return deleted;
}

function editSsot(
  domain: ScannerDomain,
  removedPaths: ReadonlySet<string>,
  dryRun: boolean,
): string[] {
  const changed: string[] = [];
  const egressResult = updateFile(
    EGRESS_FILE,
    (content) => removeEgressSections(content, domain.egressJobs),
    dryRun,
  );

  if (egressResult) {
    changed.push(egressResult);
  }

  const remaining = survivingContents(removedPaths);
  const orphans = domain.pinKeys.filter(
    (key) => !isPinKeyReferenced({ key, survivingContents: remaining }),
  );
  const pinResult = updateFile(PIN_FILE, (content) => removePinEntries(content, orphans), dryRun);

  if (pinResult) {
    changed.push(pinResult);
  }

  for (const key of domain.pinKeys.filter((key) => !orphans.includes(key))) {
    console.log(`  - lockfile の ${key} は他から参照が残っているため保持しました`);
  }

  return changed;
}

function removeDomain(
  domain: ScannerDomain,
  removedPaths: Set<string>,
  dryRun: boolean,
): boolean {
  console.log(`\n▶ ${domain.label}`);

  if (!fs.existsSync(toAbsolutePath(domain.presenceMarker))) {
    console.log("  - 撤去済みのため変更はありません");
    return false;
  }

  const deleted = deletePaths(domain, dryRun);

  for (const relativePath of domain.paths) {
    removedPaths.add(relativePath);
  }

  const touched = [...deleted, ...editSsot(domain, removedPaths, dryRun)];

  for (const file of touched) {
    console.log(`  - ${file}`);
  }

  if (dryRun) {
    console.log(`  → コミット予定: ${domain.commitSubject}`);
    return true;
  }

  if (commitPaths(touched, domain.commitSubject)) {
    console.log(`  → コミットしました: ${domain.commitSubject}`);
  }

  return true;
}

function editDocs(domains: readonly ScannerDomain[], dryRun: boolean): string[] {
  const files = new Set(
    domains.flatMap((domain) =>
      [...domain.docBlocks, ...domain.docFragments, ...domain.docSections].map(
        (entry) => entry.file,
      ),
    ),
  );
  const changed: string[] = [];

  for (const file of files) {
    const result = updateFile(
      file,
      (content) => {
        let next = content;

        for (const domain of domains) {
          for (const entry of domain.docSections.filter((s) => s.file === file)) {
            next = removeSection(next, entry.heading, file);
          }

          for (const entry of domain.docBlocks.filter((b) => b.file === file)) {
            next = removeExact(next, entry.block, file, "行");
          }

          for (const entry of domain.docFragments.filter((f) => f.file === file)) {
            next = removeExact(next, entry.fragment, file, "語句");
          }
        }

        return next;
      },
      dryRun,
    );

    if (result) {
      changed.push(result);
    }
  }

  return changed;
}

function run(dryRun: boolean): void {
  assertCleanWorktree();

  const removedPaths = new Set<string>();
  const removed = SCANNER_DOMAINS.filter((domain) => removeDomain(domain, removedPaths, dryRun));

  if (removed.length > 0) {
    console.log("\n▶ ドキュメント");

    const changed = editDocs(removed, dryRun);

    for (const file of changed) {
      console.log(`  - ${file}`);
    }

    if (dryRun) {
      console.log(`  → コミット予定: ${DOCS_COMMIT_SUBJECT}`);
    } else if (commitPaths(changed, DOCS_COMMIT_SUBJECT)) {
      console.log(`  → コミットしました: ${DOCS_COMMIT_SUBJECT}`);
    }
  }

  console.log("");
  console.log(
    dryRun
      ? "ドライラン: 書き込みもコミットもしていません。"
      : "撤去完了: 製品ごとにコミットを積みました。1 つだけ戻すには該当コミットを git revert してください（README の記述は戻りません）。",
  );
  console.log(
    "SONAR_TOKEN を登録済みの場合は、併せて削除してください（スクリプトからは操作できません）。",
  );
}

const program = newSetupCommand("remove-licensed-scanners");
program
  .description("資格情報 / ライセンス費用を要するスキャナ 2 件を撤去し、製品ごとにコミットする")
  .action((options: SetupOptions) => {
    try {
      run(options.dryRun);
    } catch (error) {
      console.error(error instanceof Error ? error.message : String(error));
      process.exitCode = 1;
    }
  });

program.parse();
