#!/usr/bin/env -S tsx
// 資格情報 / ライセンス費用を要するスキャナ 3 件を一括撤去する。撤去対象の宣言は
// scanner-manifest.ts、書き換え規則は scanner-removal.ts、git 操作は git-commit.ts が持ち、
// ここは削除・書き込み・出力とコミットの順序だけを担う。
//
// コミットは「製品ごとに 1 つ + README をまとめた 1 つ」に分ける。README の編集を製品側へ
// 混ぜないのは、3 製品の行が同じ表に隣接して並んでいるためで、混ぜると 2 つ目以降の
// `git revert` が必ず README で衝突し、1 製品だけ戻すという撤去設計の目的が果たせなくなる。
// 代わりに、復活させた製品の記述は README に戻らない。

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

/** 撤去後に残るワークフロー / composite action の本文。lockfile の孤児判定に使う。 */
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

/** 撤去したドメインだけ返す。既に消えているものは触らない。 */
function removeDomain(
  domain: ScannerDomain,
  removedPaths: Set<string>,
  dryRun: boolean,
): boolean {
  console.log(`\n▶ ${domain.label}`);

  // 撤去済みなら手を付けない。README の書き換えは完全一致に一致しないと投げる設計なので、
  // ここで抜けないと 2 回目の実行が「既に消えている」を「README が動いた」と報告してしまう。
  if (!fs.existsSync(toAbsolutePath(domain.presenceMarker))) {
    console.log("  - 撤去済みのため変更はありません");
    return false;
  }

  const deleted = deletePaths(domain, dryRun);

  // dry run では実ファイルが残るため、判定用の集合には製品をまたいで積み上げる。積まないと
  // dry run だけが「まだ参照が残っている」と答え、本番と違う結論を出す。
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
  // dry run でも確認する。汚れたツリーでは結局実行できないので、プレビューだけ通しても
  // 「プレビューは通ったのに本番で止まる」を作るだけになる。
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
  .description("資格情報 / ライセンス費用を要するスキャナ 3 件を撤去し、製品ごとにコミットする")
  .action((options: SetupOptions) => {
    try {
      run(options.dryRun);
    } catch (error) {
      console.error(error instanceof Error ? error.message : String(error));
      process.exitCode = 1;
    }
  });

program.parse();
