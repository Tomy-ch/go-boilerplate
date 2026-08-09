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

  for (const context of requiredContexts) {
    const entry = jobsByContext.get(context)!;
    if (entry.main.length !== 1) {
      findings.push({
        file: ".github/settings/branch-protection.json",
        line: 1,
        message: `required context \`${context}\` の本体 job は 1 件必要です（実際: ${entry.main.length}）`,
      });
    }
    if (entry.guard.length !== 1) {
      findings.push({
        file: ".github/settings/branch-protection.json",
        line: 1,
        message: `required context \`${context}\` の skip guard job は 1 件必要です（実際: ${entry.guard.length}）`,
      });
    }
  }

  return findings;
}
