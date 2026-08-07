import type { Finding } from "../lib/lint-report";
import {
  type WorkflowJob,
  type WorkflowStep,
  splitJobs,
  splitSteps,
  usesActionPattern,
} from "../lib/workflow";

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

/** 1 ワークフロー分の走査結果。`found` が false なら `jobs:` を読めていない（検査対象の取り違え）。 */
export type CutOffScan = {
  found: boolean;
  findings: Finding[];
  /** timeout を検査したジョブ数。再利用ワークフロー呼び出しは `timeout-minutes` を書けないので数えない。 */
  checkedJobs: number;
  /** 検査したコメント投稿ステップ数。0 件で終わる実行を「規約が守られている」と読ませないために数える。 */
  checkedSteps: number;
};

/**
 * ワークフロー 1 本を走査し、打ち切り時の振る舞いが定義されていない箇所を違反として返す。
 *
 * @remarks
 * ジョブの `timeout-minutes` と、コメント投稿ステップの `if:` / `title:` をまとめて見ます。
 * 3 つは別々の欠落ですが、打ち切りを不可視にするという 1 つの結果へ集まるため 1 本の走査に置きます。
 */
export function scanWorkflow(file: string, source: string): CutOffScan {
  const { jobs, found } = splitJobs(source);

  if (!found) {
    return { found: false, findings: [], checkedJobs: 0, checkedSteps: 0 };
  }

  const findings: Finding[] = [];
  let checkedJobs = 0;
  let checkedSteps = 0;

  for (const job of jobs) {
    if (!callsReusableWorkflow(job)) {
      checkedJobs += 1;

      if (!hasJobTimeout(job)) {
        findings.push({
          file,
          line: job.number,
          message: `ジョブ \`${job.id}\` に timeout-minutes がありません（GitHub 既定の 360 分まで走ります）`,
        });
      }
    }

    for (const step of splitSteps(job)) {
      if (!callsCommentAction(step)) continue;

      checkedSteps += 1;
      findings.push(...checkCommentStep(file, job.id, step));
    }
  }

  return { found: true, findings, checkedJobs, checkedSteps };
}

/** コメント投稿ステップ 1 つ分の `if:` / `title:` を検査する。 */
function checkCommentStep(file: string, jobId: string, step: WorkflowStep): Finding[] {
  const findings: Finding[] = [];
  const condition = conditionOf(step);

  if (condition === null) {
    findings.push({
      file,
      line: step.number,
      message: `ジョブ \`${jobId}\` の ${COMMENT_ACTION} ステップに if: がありません（暗黙の success() で打ち切り時にスキップされます）`,
    });
  } else if (!reachesCancelled(condition.value)) {
    findings.push({
      file,
      line: condition.line,
      message: `ジョブ \`${jobId}\` の ${COMMENT_ACTION} ステップの if: が打ち切りに到達しません（always() / cancelled() が要ります。failure() は cancelled では false です）`,
    });
  }

  const title = titleOf(step);

  if (title === null) {
    findings.push({
      file,
      line: step.number,
      message: `ジョブ \`${jobId}\` の ${COMMENT_ACTION} ステップに title: がありません（本文だけ書きかけで残った打ち切りを見出しで区別できません）`,
    });
  } else if (!hasCutOffHeading(title.value)) {
    findings.push({
      file,
      line: title.line,
      message: `ジョブ \`${jobId}\` の ${COMMENT_ACTION} ステップの title: に打ち切り時の見出しがありません（\`\${{ steps.X.outputs.title || '## ⚠️ …: CUT OFF (no result produced)' }}\` の形にしてください）`,
    });
  }

  return findings;
}
