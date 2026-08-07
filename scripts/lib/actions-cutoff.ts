import { type WorkflowJob, type WorkflowStep, usesActionPattern } from "./workflow";

export const COMMENT_ACTION = "./.github/actions/upsert-pr-comment";

// ジョブ直下のキーだけを見る。ステップの `uses:` を拾わないよう桁で絞る。
const JOB_LEVEL_USES = /^ {4}uses:\s*\S/;
const JOB_LEVEL_TIMEOUT = /^ {4}timeout-minutes:\s*\S/;
// ステップのキーは列 8 に並ぶ。先頭キーだけは `      - ` に続けて同じ列から始まる。
const STEP_KEY_IF = /^(?: {6}- | {8})if:\s*(.*)$/;
const STEP_KEY_TITLE = /^ {10}title:\s*(.*)$/;
const COMMENT_ACTION_USE = usesActionPattern(COMMENT_ACTION, true);
const REACHES_CANCELLED = /\b(?:always|cancelled)\s*\(\s*\)/;
const CUT_OFF_HEADING = /CUT OFF/;
const BLOCK_SCALAR_HEAD = new Set(["", ">", ">-", "|", "|-"]);

/** ステップから読み取った値と、その値が書かれている行番号。 */
export type LocatedValue = {
  line: number;
  value: string;
};

/**
 * reusable workflow を呼ぶジョブか。
 *
 * @remarks
 * 呼び出しジョブには `timeout-minutes` を書けない（invalid key）ため、その検査から外します。
 */
export function callsReusableWorkflow(job: WorkflowJob): boolean {
  return job.lines.some(({ text }) => JOB_LEVEL_USES.test(text));
}

/** ジョブ直下に `timeout-minutes:` があるか。 */
export function hasJobTimeout(job: WorkflowJob): boolean {
  return job.lines.some(({ text }) => JOB_LEVEL_TIMEOUT.test(text));
}

/** ステップが PR コメント投稿アクションを呼んでいるか。 */
export function callsCommentAction(step: WorkflowStep): boolean {
  return step.lines.some(({ text }) => COMMENT_ACTION_USE.test(text));
}

/**
 * ステップの `if:` を読む。
 *
 * @remarks
 * `if: >-` のような折り畳みスカラーでは値が次行以降にあります。ステップ内の後続の深い行を
 * 続きとみなして 1 つの式へ連結します。
 */
export function conditionOf(step: WorkflowStep): LocatedValue | null {
  for (let index = 0; index < step.lines.length; index++) {
    const matched = STEP_KEY_IF.exec(step.lines[index].text);
    if (matched === null) continue;

    const head = matched[1].trim();
    const parts = BLOCK_SCALAR_HEAD.has(head) ? [] : [head];

    for (const { text } of step.lines.slice(index + 1)) {
      if (!/^ {9,}\S/.test(text)) break;
      parts.push(text.trim());
    }

    return { line: step.lines[index].number, value: parts.join(" ") };
  }

  return null;
}

/** ステップの `title:` 入力を読む。 */
export function titleOf(step: WorkflowStep): LocatedValue | null {
  for (const { number, text } of step.lines) {
    const matched = STEP_KEY_TITLE.exec(text);
    if (matched === null) continue;

    return { line: number, value: matched[1].trim() };
  }

  return null;
}

/**
 * `if:` が打ち切られたジョブに到達するか。
 *
 * @remarks
 * 合格条件から `failure()` を外しているのは、それが cancelled で false になるためです。
 * 「ステータス関数を持つか」で書くと、関数はあるのに打ち切り時は沈黙するステップを取り逃がします。
 */
export function reachesCancelled(condition: string): boolean {
  return REACHES_CANCELLED.test(condition);
}

/** `title:` が打ち切り時の見出しを持つか。 */
export function hasCutOffHeading(title: string): boolean {
  return CUT_OFF_HEADING.test(title);
}
