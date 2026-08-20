import type { Finding } from "../lib/lint-report";
import { splitJobs } from "../lib/workflow";

export type WorkflowSource = { file: string; source: string };

/** required context を報告する job と、その workflow の起動条件を検査する。 */
export function checkRequiredChecks(
  requiredContexts: readonly string[],
  workflows: readonly WorkflowSource[],
): Finding[] {
  const findings: Finding[] = [];
  const required = new Set(requiredContexts);
  const declarations = new Map<string, WorkflowSource[]>();

  for (const context of required) declarations.set(context, []);

  for (const workflow of workflows) {
    const { found, jobs } = splitJobs(workflow.source);
    if (!found) {
      findings.push({ file: workflow.file, line: 1, message: "jobs: が見つかりません" });
      continue;
    }

    for (const job of jobs) {
      if (!required.has(job.id)) continue;
      declarations.get(job.id)!.push(workflow);
      appendTriggerFindings(job.id, workflow, findings);
    }
  }

  for (const context of requiredContexts) {
    const count = declarations.get(context)!.length;
    if (count === 1) continue;

    findings.push({
      file: ".github/settings/branch-protection.json",
      line: 1,
      message: `required context \`${context}\` を報告する job は 1 件必要です（実際: ${count}）`,
    });
  }

  return findings;
}

/**
 * required context を報告する workflow の `pull_request` 起動条件を検査する。
 *
 * @remarks
 * フィルタで起動しなかった workflow は check を 1 件も作らず、GitHub はその不在を
 * 「該当しない」ではなく「未報告」と読んで待ち続けます。起動条件は `on:` から外し、
 * job の `if:` で表現しなければなりません（skip された job は成功として数えられます）。
 */
function appendTriggerFindings(context: string, workflow: WorkflowSource, findings: Finding[]): void {
  const trigger = findPullRequestTrigger(workflow.source);

  if (trigger === null) {
    findings.push({
      file: workflow.file,
      line: 1,
      message: `required context \`${context}\` の workflow に pull_request トリガーがありません`,
    });
    return;
  }

  for (const filter of trigger.filters) {
    findings.push({
      file: workflow.file,
      line: filter.number,
      message:
        `required context \`${context}\` の pull_request に \`${filter.key}\` が残っています` +
        "（起動しなかった check は未報告のままマージを止めるため、job の if: へ移してください）",
    });
  }
}

const ON_KEY = /^on:[ \t]*(?:#.*)?$/;
const PULL_REQUEST_KEY = /^ {2}pull_request:[ \t]*(?:#.*)?$/;
const TOP_LEVEL_KEY = /^\S/;
const EVENT_KEY = /^ {2}[A-Za-z_]+:/;
const FILTER_KEY = /^ {4}(paths|paths-ignore|branches|branches-ignore):[ \t]*(?:#.*)?$/;

type PullRequestTrigger = { filters: { key: string; number: number }[] };

function findPullRequestTrigger(source: string): PullRequestTrigger | null {
  const lines = source.split("\n");
  const onIndex = lines.findIndex((line) => ON_KEY.test(line));
  if (onIndex === -1) return null;

  const start = lines.findIndex((line, index) => index > onIndex && PULL_REQUEST_KEY.test(line));
  if (start === -1) return null;

  const filters: { key: string; number: number }[] = [];

  for (let i = start + 1; i < lines.length; i++) {
    const text = lines[i];
    if (TOP_LEVEL_KEY.test(text) || EVENT_KEY.test(text)) break;

    const filter = FILTER_KEY.exec(text);
    if (filter) filters.push({ key: filter[1], number: i + 1 });
  }

  return { filters };
}
