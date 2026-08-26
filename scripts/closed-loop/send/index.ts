#!/usr/bin/env -S tsx
// 閉じた窓を Feedback Issue へ送出する。
//
// 窓を閉じるのはここではなく .agents/closed-loop/marks.sh。
//
// 届かなかった窓は索引に feedbackIssue を持たないまま残り、次回に持ち越す。逐語のターンが
// public に出るのは、手元の読解が走らなかったときの縮退経路だけ（docs/design/security.md）。
//
// 使い方:
//   tsx scripts/closed-loop/send [--dry-run] [--transcripts <dir>]

import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { renderCandidateComment, selectCandidates } from "../candidates";
import { findByWindow, needsSend, parseIndex, upsert, type IndexEntry } from "../index-store";
import { issueTitle, renderBody, type Observation } from "../issue";
import { foldWindowEvents } from "../report";
import {
  buildPrompt,
  issueLabels,
  needsCandidateComment,
  parseSummary,
  readingGap,
  LOCAL_CANDIDATE_LIMIT,
  LOCAL_EXCERPT_CHARS,
  type Summary,
} from "../summarize";
import { collectWindows, isSubstantive, phasesOf, representativeAt, type MarksReader } from "../windows";

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const MARKS_DIR = path.join(REPO_ROOT, "tmp", "closed-loop", "marks");
const INDEX_FILE = path.join(REPO_ROOT, ".agents", "private", "closed_loop_index.json");
const DRY_RUN = process.argv.includes("--dry-run");
const NO_SUMMARY = process.argv.includes("--no-summary");
const SUMMARY_TIMEOUT_MS = 180_000;
// 読解に必要なのは渡したプロンプトだけ。外界へ出る手段は落とす。
// `--allowed-tools ''` は無視される（実測）ので、許可側ではなく拒否側で列挙する。
const SUMMARY_DENIED_TOOLS = "Read Bash Glob Grep Edit Write NotebookEdit WebFetch WebSearch Task";

function flag(name: string): string | undefined {
  const i = process.argv.indexOf(`--${name}`);
  return i >= 0 ? process.argv[i + 1] : undefined;
}

const marksReader: MarksReader = {
  listWindowIds: () =>
    fs.existsSync(MARKS_DIR)
      ? fs
          .readdirSync(MARKS_DIR, { withFileTypes: true })
          .filter((e) => e.isDirectory())
          .map((e) => e.name)
      : [],
  listFiles: (id) => fs.readdirSync(path.join(MARKS_DIR, id)),
  readFile: (id, file) => fs.readFileSync(path.join(MARKS_DIR, id, file), "utf8"),
};

function readIndex() {
  try {
    return parseIndex(JSON.parse(fs.readFileSync(INDEX_FILE, "utf8")));
  } catch {
    return parseIndex(undefined);
  }
}

function writeIndex(store: { entries: readonly IndexEntry[] }): void {
  fs.mkdirSync(path.dirname(INDEX_FILE), { recursive: true });
  fs.writeFileSync(INDEX_FILE, `${JSON.stringify(store, null, 2)}\n`);
}

function gh(args: string[]): string {
  return execFileSync("gh", args, { encoding: "utf8" }).trim();
}

/**
 * 手元のモデルに読解させる。呼べない／失敗した場合は `undefined`。
 *
 * @remarks
 * リポジトリの外で走らせる。中で走らせると 2 つ壊れる。`.claude/settings.json` の
 * `SessionEnd` フックが発火して**いま観測中の窓が閉じられ**、1 つの作業が 2 窓に割れる
 * （実測: 開いた窓に `closedAt` が打たれる）。加えて同じ設定の広い `allow` 規則を継承し、
 * リポジトリ内のファイルへ到達できてしまう。
 *
 * ツールは `SUMMARY_DENIED_TOOLS` で落とす。
 *
 * stderr は捨てる。claude は設定の警告などを stderr に出すが、それは読解の成否と関係が無く、
 * 混ぜると本文の解析が壊れる。
 */
function localSummary(prompt: string): Summary | undefined {
  try {
    const out = execFileSync("claude", ["-p", "--model", "sonnet", "--disallowed-tools", SUMMARY_DENIED_TOOLS], {
      cwd: os.tmpdir(),
      encoding: "utf8",
      input: prompt,
      stdio: ["pipe", "pipe", "ignore"],
      timeout: SUMMARY_TIMEOUT_MS,
    });
    return parseSummary(out);
  } catch {
    return undefined;
  }
}

/** 窓のブランチ。打刻には無いので git から引く。窓ごとに違う値は取れないため現在値を使う。 */
function currentBranch(): string | undefined {
  try {
    const b = execFileSync("git", ["rev-parse", "--abbrev-ref", "HEAD"], { encoding: "utf8" }).trim();
    return b === "HEAD" ? undefined : b;
  } catch {
    return undefined;
  }
}

/** ブランチに紐づく PR 番号。無ければ undefined。 */
function prFor(branch: string): number | undefined {
  try {
    const out = gh(["pr", "list", "--head", branch, "--state", "all", "--limit", "1", "--json", "number"]);
    const parsed: unknown = JSON.parse(out);
    const first = Array.isArray(parsed) ? parsed[0] : undefined;
    const n = typeof first === "object" && first !== null ? (first as { number?: unknown }).number : undefined;
    return typeof n === "number" ? n : undefined;
  } catch {
    return undefined;
  }
}

/**
 * 窓の期間に落ちるイベントだけを畳む。
 *
 * ディレクトリ全体を畳むと、窓 1 件の本文に全期間の合計が載る。窓ごとの数字であることが
 * 前提の集計なので、期間で絞るのはここの責務になる。
 */
function readTranscripts(dir: string): string[] {
  const contents: string[] = [];
  for (const f of fs.readdirSync(dir).filter((n) => n.endsWith(".jsonl"))) {
    try {
      contents.push(fs.readFileSync(path.join(dir, f), "utf8"));
    } catch {
      // 読めないファイルは無いものとして扱う
    }
  }
  return contents;
}

function sessionFactsFor(dir: string | undefined, from: number | undefined, to: number | undefined) {
  if (dir === undefined || !fs.existsSync(dir) || from === undefined || to === undefined) return undefined;
  return foldWindowEvents(readTranscripts(dir), { from, to });
}

const branch = currentBranch();
const transcriptsDir = flag("transcripts");
let store = readIndex();
let sent = 0;
let pending = 0;
let skipped = 0;

for (const window of collectWindows(marksReader)) {
  if (representativeAt(window, "closedAt") === undefined) continue; // まだ開いている窓は送らない
  const existing = findByWindow(store, window.id);
  if (!needsSend(existing)) continue;

  const openedAt = representativeAt(window, "openedAt");
  const closedAt = representativeAt(window, "closedAt");
  const transcripts = sessionFactsFor(transcriptsDir, openedAt, closedAt);

  // 開いて閉じただけの窓は送らない。公開 issue にしても読む人に何も伝えないうえ、
  // 消すには手作業が要る。打刻は tmp/ に残るので closed-loop-report からは見える。
  if (!isSubstantive(window, transcripts?.facts)) {
    skipped += 1;
    continue;
  }

  const observation: Observation = {
    windowId: window.id,
    client: "claude",
    branch,
    pr: branch === undefined ? undefined : prFor(branch),
    openedAt,
    closedAt,
    phases: phasesOf(window).map((p) => ({ from: p.from, to: p.to, sec: p.durationSec })),
    sessions: transcripts?.sessions,
    prompts: transcripts?.facts.prompts,
    toolCalls: transcripts?.facts.toolCalls,
    toolFailures: transcripts?.facts.toolFailures,
    interrupts: transcripts?.facts.interrupts,
    skills: transcripts?.facts.skillCalls,
  };

  const title = issueTitle(observation);

  // 手元の読解に渡す候補は、公開する既定より多く長く取る（docs/design/closed-loop.md）。
  // dry-run は読解を呼ばない。何も送らない実行がモデルを窓の数だけ叩かないようにする。
  const summary =
    transcripts === undefined || NO_SUMMARY || DRY_RUN
      ? undefined
      : localSummary(
          buildPrompt(observation, selectCandidates(transcripts.events, LOCAL_CANDIDATE_LIMIT, LOCAL_EXCERPT_CHARS)),
        );
  const gap = readingGap(summary, transcripts !== undefined);
  const body = renderBody(observation, summary?.sections);
  const labels = issueLabels(gap, summary);

  if (DRY_RUN) {
    const comment =
      transcripts === undefined
        ? "(トランスクリプト未指定)"
        : "(dry-run では読解を呼ばないため、実際の送出でコメントが付くかは確定しない)";
    console.log(`--- ${title}\nラベル: ${labels.join(", ")}\n${body}\n--- コメント\n${comment}`);
    sent += 1;
    continue;
  }

  // issue の作成とコメントの投稿は別の try に分ける。束ねると部分成功が「送出済み」になり、
  // コメントだけが二度と再試行されない（scripts/README.md）。
  let issueNumber = existing?.feedbackIssue;
  if (issueNumber === undefined) {
    try {
      const url = gh(["issue", "create", "--title", title, "--body", body, ...labels.flatMap((l) => ["--label", l])]);
      const m = /\/(\d+)$/.exec(url);
      issueNumber = m ? Number(m[1]) : undefined;
    } catch (e) {
      console.error(`issue を作成できませんでした（次回に持ち越します）: ${window.id}: ${String(e).split("\n")[0]}`);
    }
  }

  let commentPending: boolean | undefined;
  if (issueNumber !== undefined && transcripts !== undefined && needsCandidateComment(gap)) {
    try {
      const comment = renderCandidateComment(selectCandidates(transcripts.events));
      gh(["issue", "comment", String(issueNumber), "--body", comment]);
    } catch (e) {
      commentPending = true;
      console.error(`コメントを投稿できませんでした（次回に再送します）: #${issueNumber}: ${String(e).split("\n")[0]}`);
    }
  }

  store = upsert(store, {
    windowId: window.id,
    branch,
    feedbackIssue: issueNumber,
    commentPending,
    updatedAt: Math.floor(Date.now() / 1000),
  });
  // 窓ごとに書き出す。最後にまとめて書くと、途中で落ちた実行が作った issue が索引に残らず、
  // 次回そのぶんを作り直す。
  writeIndex(store);
  if (issueNumber === undefined || commentPending === true) pending += 1;
  else sent += 1;
}

console.log(
  `送出 ${sent} 件` +
    `${pending > 0 ? ` / 未送出 ${pending} 件（次回に持ち越し）` : ""}` +
    `${skipped > 0 ? ` / 中身が無く送らなかった窓 ${skipped} 件` : ""}`,
);
