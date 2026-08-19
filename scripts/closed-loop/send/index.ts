#!/usr/bin/env -S tsx
// 閉じた窓を Feedback Issue へ送出する。
//
// 判定は issue.ts / index-store.ts / windows.ts が持ち、ここはファイル入出力・gh の呼び出し・
// 終了コードだけを担う。
//
// 窓を閉じる処理そのものは送らない。閉じるのは hook（marks.sh --hook session-end）で、
// ネットワークに触れず即座に終わる。送出はここで別に行い、届かなければ索引に
// feedbackIssue を持たないまま残して次回に持ち越す。「ローカル開発をブロックしない」は
// この分離で満たす。
//
// 使い方:
//   tsx scripts/closed-loop/send [--dry-run] [--transcripts <dir>]

import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { renderCandidateComment, selectCandidates } from "../candidates";
import { parseClaudeLine, summarizeSession, type Event } from "../events";
import { findByWindow, needsSend, parseIndex, upsert, type IndexEntry } from "../index-store";
import { issueTitle, renderBody, type Observation } from "../issue";
import { withinPeriod } from "../report";
import { isSubstantive, phasesOf, representativeAt, toWindow, type Window } from "../windows";

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const MARKS_DIR = path.join(REPO_ROOT, "tmp", "closed-loop", "marks");
const INDEX_FILE = path.join(REPO_ROOT, ".agents", "private", "closed_loop_index.json");
const DRY_RUN = process.argv.includes("--dry-run");

function flag(name: string): string | undefined {
  const i = process.argv.indexOf(`--${name}`);
  return i >= 0 ? process.argv[i + 1] : undefined;
}

function readWindows(): Window[] {
  if (!fs.existsSync(MARKS_DIR)) return [];
  return fs
    .readdirSync(MARKS_DIR, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => {
      const dir = path.join(MARKS_DIR, e.name);
      const files: Record<string, string> = {};
      for (const f of fs.readdirSync(dir)) files[f] = fs.readFileSync(path.join(dir, f), "utf8");
      return toWindow(e.name, files);
    });
}

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
function sessionFactsFor(dir: string | undefined, from: number | undefined, to: number | undefined) {
  if (dir === undefined || !fs.existsSync(dir) || from === undefined || to === undefined) return undefined;
  const events: Event[] = [];
  const sessions = new Set<string>();
  for (const f of fs.readdirSync(dir).filter((n) => n.endsWith(".jsonl"))) {
    try {
      for (const line of fs.readFileSync(path.join(dir, f), "utf8").split("\n")) {
        for (const e of parseClaudeLine(line)) {
          if (!withinPeriod(e.at, { from, to })) continue;
          events.push(e);
          if (e.sessionId !== undefined) sessions.add(e.sessionId);
        }
      }
    } catch {
      // 読めないファイルは無いものとして扱う
    }
  }
  return { facts: summarizeSession("claude", events), sessions: sessions.size, events };
}

const branch = currentBranch();
const transcriptsDir = flag("transcripts");
let store = readIndex();
let sent = 0;
let pending = 0;
let skipped = 0;

for (const window of readWindows()) {
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
  const body = renderBody(observation);

  if (DRY_RUN) {
    const comment = transcripts === undefined ? "(トランスクリプト未指定)" : renderCandidateComment(selectCandidates(transcripts.events));
    console.log(`--- ${title}\n${body}\n--- コメント\n${comment}`);
    sent += 1;
    continue;
  }

  // issue の作成とコメントの投稿は別の try に分ける。1 つに束ねると、issue は作れたが
  // コメントだけ落ちた部分成功が「送出済み」に見え、読解候補が永久に付かないまま
  // 二度と再試行されない。
  let issueNumber = existing?.feedbackIssue;
  if (issueNumber === undefined) {
    try {
      const url = gh(["issue", "create", "--title", title, "--body", body, "--label", "feedback"]);
      const m = /\/(\d+)$/.exec(url);
      issueNumber = m ? Number(m[1]) : undefined;
    } catch (e) {
      // 届かなければ索引に残して次回へ持ち越す。窓を閉じる処理は既に終わっており、ここで
      // 落としてもローカルの作業は止まらない。
      console.error(`issue を作成できませんでした（次回に持ち越します）: ${window.id}: ${String(e).split("\n")[0]}`);
    }
  }

  let commentPending: boolean | undefined;
  if (issueNumber !== undefined && transcripts !== undefined) {
    try {
      // 読解候補は本文ではなくコメントへ。本文は AI の出力面であり、入力と出力を同じ
      // テキストに混ぜると AI が自分の入力を上書きしうる。
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
  if (issueNumber === undefined || commentPending === true) pending += 1;
  else sent += 1;
}

if (!DRY_RUN) writeIndex(store);
console.log(
  `送出 ${sent} 件` +
    `${pending > 0 ? ` / 未送出 ${pending} 件（次回に持ち越し）` : ""}` +
    `${skipped > 0 ? ` / 中身が無く送らなかった窓 ${skipped} 件` : ""}`,
);
