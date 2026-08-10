// ワークフロー定義を「桁」で読むための共通の切り出し。
//
// YAML パーサを持ち込まないのは、検査対象が値ではなく記述そのものだから。ブロックスカラーの
// 中身は必ず親より深い桁に来るという YAML の規約に乗っており、その前提は同じ `actions-lint` の
// 中で先に走る actionlint が担保する。

/** 行番号（1 始まり）付きの 1 行。 */
export type WorkflowLine = {
  number: number;
  text: string;
};

/** `jobs:` 直下の 1 ジョブ。`number` はジョブ見出し行の行番号。 */
export type WorkflowJob = {
  id: string;
  number: number;
  lines: WorkflowLine[];
};

/** `steps:` 配下の 1 ステップ。`number` は `- ` で始まる先頭行の行番号。 */
export type WorkflowStep = {
  number: number;
  lines: WorkflowLine[];
};

export type SplitWorkflow = {
  /** `jobs:` が見つかったか。見つからないのは検査対象の取り違えなので、空とは区別する。 */
  found: boolean;
  jobs: WorkflowJob[];
  /** `jobs:` の外側（トップレベルの `env:` など）。ジョブ本文には現れないが、ジョブへ届く。 */
  preamble: WorkflowLine[];
};

const JOBS_KEY = /^jobs:\s*(#.*)?$/;
// 行末コメントや引用符といった書式差で検出が外れると、規約が破られた瞬間に検査が沈黙する。
const JOB_HEADER = /^ {2}(?:"([A-Za-z0-9_-]+)"|'([A-Za-z0-9_-]+)'|([A-Za-z0-9_-]+)):\s*(#.*)?$/;
const STEPS_KEY = /^ {4}steps:\s*(#.*)?$/;
const STEP_ITEM = /^ {6}- /;

/**
 * ディレクトリの読み取り結果から、検査対象のワークフローを選び出してパスへ組み立てる。
 *
 * @remarks
 * 拡張子は `.yaml` / `.yml` の両方を採ります。GitHub がどちらも読むため、片方だけを見ると
 * 拡張子を替えただけのワークフローが検査から静かに外れます。並びを固定するのは、違反の出力順が
 * 実行ごとに揺れると CI の失敗差分が読めなくなるためです。
 */
export function selectWorkflowFiles(names: readonly string[], dir: string): string[] {
  return names
    .filter((name) => name.endsWith(".yaml") || name.endsWith(".yml"))
    .sort()
    .map((name) => `${dir}/${name}`);
}

/** `uses:` にこのローカルアクションを指定している行へ当たる正規表現を組み立てる。 */
export function usesActionPattern(actionPath: string, anchored: boolean): RegExp {
  const escaped = actionPath.replace(/[.*+?^${}()|[\]\\\/]/g, "\\$&");

  return anchored
    ? new RegExp(`^(?: {6}- | {8})uses:\\s*["']?${escaped}["']?\\s*(#.*)?$`)
    : new RegExp(`uses:\\s*["']?${escaped}["']?\\s*(#.*)?$`);
}

/**
 * ワークフロー本文をジョブ単位に切り出す。
 *
 * @remarks
 * 桁 0 のコメント行では `jobs:` を打ち切りません。トップレベルキーではないため、ここで
 * 打ち切ると以降のジョブが丸ごと検査対象から外れます。
 */
export function splitJobs(source: string): SplitWorkflow {
  const lines = source.split("\n");
  const jobsIndex = lines.findIndex((line) => JOBS_KEY.test(line));

  if (jobsIndex === -1) {
    return { found: false, jobs: [], preamble: lines.map(toLine(0)) };
  }

  const jobs: WorkflowJob[] = [];
  let current: WorkflowJob | null = null;
  let end = lines.length;

  for (let i = jobsIndex + 1; i < lines.length; i++) {
    const text = lines[i];

    if (/^\S/.test(text) && !text.startsWith("#")) {
      end = i;
      break;
    }

    const header = JOB_HEADER.exec(text);
    if (header) {
      current = { id: header[1] ?? header[2] ?? header[3], number: i + 1, lines: [] };
      jobs.push(current);
      continue;
    }

    if (current) current.lines.push({ number: i + 1, text });
  }

  const preamble = [
    ...lines.slice(0, jobsIndex).map(toLine(0)),
    ...lines.slice(end).map(toLine(end)),
  ];

  return { found: true, jobs, preamble };
}

/**
 * ジョブ本文をステップ単位に切り出す。
 *
 * @remarks
 * `if:` / `title:` は「そのステップのもの」だけを見る必要があるため、`- ` で区切ります。
 */
export function splitSteps(job: WorkflowJob): WorkflowStep[] {
  const start = job.lines.findIndex(({ text }) => STEPS_KEY.test(text));
  if (start === -1) return [];

  const steps: WorkflowStep[] = [];
  let current: WorkflowStep | null = null;

  for (const line of job.lines.slice(start + 1)) {
    if (STEP_ITEM.test(line.text)) {
      current = { number: line.number, lines: [] };
      steps.push(current);
    } else if (/^ {0,4}\S/.test(line.text) && !/^\s*#/.test(line.text)) {
      // steps: と同じかそれより浅い桁のキーが来たらステップ列の終わり。
      break;
    }

    if (current) current.lines.push(line);
  }

  return steps;
}

function toLine(offset: number): (text: string, index: number) => WorkflowLine {
  return (text, index) => ({ number: offset + index + 1, text });
}
