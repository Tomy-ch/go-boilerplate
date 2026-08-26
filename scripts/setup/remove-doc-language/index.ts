#!/usr/bin/env -S tsx
// ドキュメントの言語をどちらか一方へ畳む。計画は plan.ts、規則は doc-language.ts、
// 撤去対象の宣言は language-manifest.ts が持つ。
//
// 他の setup ツールと違い git へコミットまで積むのは、400 件超のファイルが一度に動くためである。
// 未コミットの変更と混ざると、やめたくなったときに戻す手立てが無い。

import fs from "node:fs";

import { BASELINE_PATH, scanRepository } from "../../marker-baseline/scan";
import { toAbsolutePath } from "../lib/file-utils";
import { assertCleanWorktree, commitPaths, trackedFiles } from "../lib/git-commit";
import { newSetupCommand } from "../lib/runtime";
import { type Mode } from "./doc-language";
import {
  ALLOWED_MENTIONS,
  COMMIT_SUBJECTS,
  DECLARED_LINES,
  DOC_REPLACEMENTS,
  MODES,
  REMOVED_PATHS,
  SELF_DESTRUCT_PATHS,
} from "./language-manifest";
import { type Operation, planKeepBoth, planRemoval } from "./plan";

/** `--lang` に渡せる値。`both` は両方残す（何もしない）。 */
type LanguageChoice = Mode | "both";

type Options = {
  dryRun: boolean;
  lang?: string;
};

/** 宣言の無い散文に当たって撤去を止めたことを表す。 */
class UndeclaredProseError extends Error {
  constructor(lines: readonly { file: string; line: number; text: string }[]) {
    super(
      [
        `対訳規約を説明している散文が ${lines.length} 件あり、機械的には畳めません。`,
        "撤去を中止しました（1 ファイルも書き換えていません）。",
        "",
        "それぞれの箇所へ doc-pair マーカーを入れるか、language-manifest.ts の",
        "DECLARED_LINES へ完全一致で宣言してから再実行してください。",
        "",
        ...lines.map(({ file, line, text }) => `  ${file}:${line}  ${text.trim()}`),
      ].join("\n"),
    );
    this.name = "UndeclaredProseError";
  }
}

/** 差し替え宣言が本文に当たらなくなったことを表す。 */
class StaleReplacementError extends Error {
  constructor(replacements: readonly { file: string; from: string }[]) {
    super(
      [
        `差し替えの宣言 ${replacements.length} 件が本文に当たりません。`,
        "撤去を中止しました（1 ファイルも書き換えていません）。",
        "",
        "language-manifest.ts の DOC_REPLACEMENTS を本文に合わせて直してください。",
        "空振りのまま進めると、差し替えたつもりの記述がそのまま作成先へ残ります。",
        "",
        ...replacements.map(({ file, from }) => `  ${file}: ${from}`),
      ].join("\n"),
    );
    this.name = "StaleReplacementError";
  }
}

/**
 * 追跡ファイルのうち、実体を持つものだけを返す。
 *
 * @remarks
 * シンボリックリンクの中身は指し先のものです。除かないと同じ実体を 2 回書き換えることになり、
 * `ja` では指し先を消して作り直した後にリンク経由でもう一度処理します。このリポジトリでは
 * `CLAUDE.md` と `CODEX.md` が `AGENTS.md` を指しています。
 */
function realFiles(): string[] {
  return trackedFiles().filter((relativePath) => {
    try {
      return !fs.lstatSync(toAbsolutePath(relativePath)).isSymbolicLink();
    } catch {
      return false;
    }
  });
}

function readRepoFile(relativePath: string): string | null {
  try {
    return fs.readFileSync(toAbsolutePath(relativePath), "utf8");
  } catch {
    return null;
  }
}

function applyOperation(operation: Operation, dryRun: boolean): string[] {
  if (dryRun) {
    return operation.kind === "rename" ? [operation.from, operation.to] : [operation.path];
  }

  switch (operation.kind) {
    case "delete":
      fs.rmSync(toAbsolutePath(operation.path), { force: true });

      return [operation.path];

    case "write":
      fs.writeFileSync(toAbsolutePath(operation.path), operation.content);

      return [operation.path];

    case "rename":
      fs.rmSync(toAbsolutePath(operation.from), { force: true });
      fs.writeFileSync(toAbsolutePath(operation.to), operation.content);

      return [operation.from, operation.to];
  }
}

function removeDeclaredPaths(dryRun: boolean): string[] {
  return REMOVED_PATHS.filter((relativePath) => fs.existsSync(toAbsolutePath(relativePath))).map(
    (relativePath) => {
      if (!dryRun) {
        fs.rmSync(toAbsolutePath(relativePath), { recursive: true, force: true });
      }

      return relativePath;
    },
  );
}

/**
 * マーカー行の分布のベースラインを、撤去後のツリーで引き直す。
 *
 * @remarks
 * ベースラインはマーカー行が増えていないかを見張る固定値なので、正当に減るこの撤去の後は
 * 引き直さない限り `scripts/marker-baseline/scan.test.ts` が鳴り続けます。
 *
 * 呼ぶ位置は自消滅の後です。このツール自身のテストはマーカーの形を入力として持つので、
 * 先に引くと消える予定のファイルを数え、消えた後に食い違います。
 */
function rewriteMarkerBaseline(dryRun: boolean): string[] {
  if (dryRun) {
    return [];
  }

  fs.writeFileSync(BASELINE_PATH, `${JSON.stringify(scanRepository(), null, 2)}\n`);

  return ["scripts/marker-baseline/baseline.json"];
}

/** 畳む 2 モードの計画。宣言と食い違ったら、1 バイトも書かずに投げる。 */
function planFold(choice: Mode): Operation[] {
  const plan = planRemoval(
    choice,
    realFiles(),
    readRepoFile,
    new Set(DECLARED_LINES),
    // 自消滅する対象も畳む対象から外す。このツール自身のソースとテストは `.ja.md` を語るが、
    // 撤去し終えた後には 1 行も残らない。
    [...REMOVED_PATHS, ...SELF_DESTRUCT_PATHS],
    DOC_REPLACEMENTS,
    new Set(ALLOWED_MENTIONS),
  );

  if (plan.staleReplacements.length > 0) {
    throw new StaleReplacementError(plan.staleReplacements);
  }

  if (plan.undeclared.length > 0) {
    throw new UndeclaredProseError(plan.undeclared);
  }

  return plan.operations;
}

function run(choice: LanguageChoice, dryRun: boolean): void {
  // ドライランは何も書かないので、作業ツリーの状態を問わない。プレビューを見るために
  // 手元の変更をコミットさせるのは、確かめてから決めるという手順そのものを壊す。
  if (!dryRun) {
    assertCleanWorktree();
  }

  const operations =
    choice === "both"
      ? planKeepBoth(realFiles(), readRepoFile, SELF_DESTRUCT_PATHS)
      : planFold(choice);

  const touched = operations.flatMap((operation) => applyOperation(operation, dryRun));
  // 対訳を運ぶ仕組みは `both` では残る。使い続けると決めた選択だからである。
  const declared = choice === "both" ? [] : removeDeclaredPaths(dryRun);
  const subject = COMMIT_SUBJECTS[choice];

  console.log(
    choice === "both"
      ? `▶ ${operations.length} 件のファイルから言語選択のマーカーを解決しました（both）`
      : `▶ ${operations.length} 件のファイルを畳みました（${choice}）`,
  );
  console.log(`  - 削除 / 改名 / 書き換え: ${touched.length} パス`);

  for (const relativePath of declared) {
    console.log(`  - 撤去: ${relativePath}`);
  }

  if (dryRun) {
    console.log(`  → コミット予定: ${subject}`);
    console.log("\nドライラン: 書き込みもコミットもしていません。");

    return;
  }

  const selfDestruct = SELF_DESTRUCT_PATHS.filter((relativePath) =>
    fs.existsSync(toAbsolutePath(relativePath)),
  );

  for (const relativePath of selfDestruct) {
    fs.rmSync(toAbsolutePath(relativePath), { recursive: true, force: true });
  }

  // ベースラインは最後に引く。このツール自身のテストがマーカーの形を入力として持つため、
  // 自消滅より先に引くと「在るはずのマーカーが無くなった」と鳴り続ける。
  const baseline = rewriteMarkerBaseline(dryRun);

  if (commitPaths([...touched, ...declared, ...baseline, ...selfDestruct], subject)) {
    console.log(`  → コミットしました: ${subject}`);
  }

  console.log("\n撤去完了。戻すにはこのコミットを git revert してください。");
}

function parseChoice(value: string | undefined): LanguageChoice {
  if (value === undefined || !MODES.includes(value as LanguageChoice)) {
    throw new Error(`--lang に ${MODES.join(" / ")} のいずれかを指定してください。`);
  }

  return value as LanguageChoice;
}

const program = newSetupCommand("remove-doc-language");
program
  .description("ドキュメント / スキルの対訳ペアを、選んだ言語で解決する")
  .option("--lang <lang>", `残す言語（${MODES.join(" / ")}）。both は畳まずマーカーだけ解決する`)
  .action((options: Options) => {
    try {
      run(parseChoice(options.lang), options.dryRun);
    } catch (error) {
      console.error(error instanceof Error ? error.message : String(error));
      process.exitCode = 1;
    }
  });

program.parse();
