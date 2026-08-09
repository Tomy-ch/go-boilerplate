import type { Finding } from "../lib/lint-report";
import { splitJobs } from "../lib/workflow";

export type WorkflowSource = { file: string; source: string };

/** Ruleset 宣言と本体・skip guard の三者対応を検査する。 */
export function checkRequiredChecks(
  requiredContexts: readonly string[],
  workflows: readonly WorkflowSource[],
): Finding[] {
  const findings: Finding[] = [];
  const required = new Set(requiredContexts);
  const jobsByContext = new Map<string, { main: string[]; guard: string[] }>();

  for (const context of required) jobsByContext.set(context, { main: [], guard: [] });

  collectWorkflowJobs(required, workflows, jobsByContext, findings);
  appendRequiredContextFindings(requiredContexts, jobsByContext, findings);

  return findings;
}

function collectWorkflowJobs(
  required: ReadonlySet<string>,
  workflows: readonly WorkflowSource[],
  jobsByContext: Map<string, { main: string[]; guard: string[] }>,
  findings: Finding[],
): void {
  for (const workflow of workflows) {
    const { found, jobs } = splitJobs(workflow.source);
    if (!found) {
      findings.push({ file: workflow.file, line: 1, message: "jobs: が見つかりません" });
      continue;
    }

    const isGuard = workflow.file.endsWith("-guard.yaml") || workflow.file.endsWith("-guard.yml");

    for (const job of jobs) {
      const entry = jobsByContext.get(job.id);
      if (entry !== undefined) (isGuard ? entry.guard : entry.main).push(workflow.file);

      if (isGuard && !required.has(job.id)) {
        findings.push({
          file: workflow.file,
          line: job.number,
          message: `guard job \`${job.id}\` は branch-protection.json の required context ではありません`,
        });
      }
    }
  }
}

function appendRequiredContextFindings(
  requiredContexts: readonly string[],
  jobsByContext: ReadonlyMap<string, { main: string[]; guard: string[] }>,
  findings: Finding[],
): void {
  for (const context of requiredContexts) {
    const entry = jobsByContext.get(context)!;
    appendJobCountFinding(context, "本体", entry.main.length, findings);
    appendJobCountFinding(context, "skip guard", entry.guard.length, findings);
  }
}

function appendJobCountFinding(
  context: string,
  kind: "本体" | "skip guard",
  actual: number,
  findings: Finding[],
): void {
  if (actual === 1) return;

  findings.push({
    file: ".github/settings/branch-protection.json",
    line: 1,
    message: `required context \`${context}\` の${kind} job は 1 件必要です（実際: ${actual}）`,
  });
}
