/**
 * 窓の観測を Feedback Issue の本文へ書き、書いたものを読み返す。
 *
 * @remarks
 * 書く側と読む側をここに対で置きます。週次の統合機構はこの本文を解析して集計するので、
 * 形式が片方だけ変わると集計が黙って空になります。往復が同じモジュールにあれば、
 * 変更したときにテストが両側で落ちます。
 *
 * 本文は 2 層です。人が読む H2 節と、機械が読む YAML ブロック。決定論的に取れた事実だけを
 * YAML に置き、意味の分類は AI が後から H2 節へ書きます（`docs/design/closed-loop.md`
 * 「Deterministic first, model second」）。
 */

/** 機械が読む観測ブロック。決定論的に取れたものだけを置く。 */
export type Observation = {
  readonly windowId: string;
  readonly client: string;
  readonly branch?: string;
  readonly pr?: number;
  readonly parentIssue?: number;
  readonly kind?: string;
  readonly openedAt?: number;
  readonly closedAt?: number;
  readonly phases: readonly { readonly from: string; readonly to: string; readonly sec: number }[];
  readonly sessions?: number;
  readonly prompts?: number;
  readonly toolCalls?: number;
  readonly toolFailures?: number;
  readonly interrupts?: number;
  readonly skills?: Readonly<Record<string, number>>;
};

/** 本文に置く H2 節。Outcome から Evidence へ、着地から根拠へと降りる順。 */
export const BODY_SECTIONS: readonly string[] = [
  "Outcome",
  "Friction",
  "AI Misunderstanding",
  "Skill / Rule Gap",
  "Tooling Gap",
  "Human Intervention",
  "Suggested Improvement",
  "Evidence",
] as const;

/**
 * 改善案が書かれる節。
 *
 * @remarks
 * 週次がこの節だけを名指しで拾います。8 つのうち唯一、そのままレトロの議題になる形で書かれる
 * 節だからです。名前を定数にしておくのは、`BODY_SECTIONS` の綴りを変えたときに拾えなくなる側が
 * 黙って空になるためです。
 */
export const IMPROVEMENT_SECTION = "Suggested Improvement";

const FENCE = "```";
const BLOCK_START = `${FENCE}yaml closed-loop`;

/** Issue のタイトル。窓 ID を含めるのは、本文を読まずに突き合わせられるようにするため。 */
export function issueTitle(observation: Observation): string {
  const where = observation.branch ?? observation.client;
  return `[feedback] ${where} (${observation.windowId})`;
}

/**
 * 観測を YAML ブロックへ書き出す。
 *
 * @remarks
 * `undefined` の項目は書きません。「観測できなかった」を空欄で表し、`0` と区別するためです。
 * 依存を増やさないよう、値が単純な型に限られることを前提に手で組み立てます。
 */
export function renderObservation(observation: Observation): string {
  const lines: string[] = [BLOCK_START];
  const put = (key: string, value: string | number | undefined) => {
    if (value !== undefined) lines.push(`${key}: ${value}`);
  };
  put("window_id", observation.windowId);
  put("client", observation.client);
  put("branch", observation.branch);
  put("pr", observation.pr);
  put("parent_issue", observation.parentIssue);
  put("kind", observation.kind);
  put("opened_at", observation.openedAt);
  put("closed_at", observation.closedAt);
  put("sessions", observation.sessions);
  put("prompts", observation.prompts);
  put("tool_calls", observation.toolCalls);
  put("tool_failures", observation.toolFailures);
  put("interrupts", observation.interrupts);
  if (observation.phases.length > 0) {
    lines.push("phases:");
    for (const p of observation.phases) lines.push(`  - {from: ${p.from}, to: ${p.to}, sec: ${p.sec}}`);
  }
  if (observation.skills !== undefined) {
    const entries = Object.entries(observation.skills);
    lines.push("skills:");
    for (const [name, count] of entries) lines.push(`  ${name}: ${count}`);
  }
  lines.push(FENCE);
  return lines.join("\n");
}

/**
 * Issue 本文を組み立てる。
 *
 * @param sections 読解が得られていれば節名 → 本文。渡さなければ節は空のまま置く。
 *
 * @remarks
 * 空の節を残す形を捨てないのは、読解が得られなかった窓も観測としては成立するからです。
 * 節が空であることが「読解がまだ無い」を表し、後から埋められます。
 */
export function renderBody(observation: Observation, sections?: Readonly<Record<string, string>>): string {
  const parts = [
    "<!-- このブロックは機械が書き換えます。手で編集すると集計から外れます -->",
    renderObservation(observation),
    "",
    ...BODY_SECTIONS.map((s) => `## ${s}\n${sections?.[s] ?? ""}\n`),
  ];
  return parts.join("\n");
}

/**
 * 本文から観測ブロックを読み返す。
 *
 * @remarks
 * 解析できない本文は `undefined` を返します。人が手で壊した Issue 1 件で
 * 週次集計全体を落とさないためです。落ちた件数は呼び出し側が報告します。
 */
export function parseObservation(body: string): Observation | undefined {
  const start = body.indexOf(BLOCK_START);
  if (start < 0) return undefined;
  const rest = body.slice(start + BLOCK_START.length);
  const end = rest.indexOf(FENCE);
  if (end < 0) return undefined;
  const block = rest.slice(0, end);

  const scalars: Record<string, string> = {};
  const phases: { from: string; to: string; sec: number }[] = [];
  const skills: Record<string, number> = {};
  let mode: "top" | "phases" | "skills" = "top";

  for (const raw of block.split("\n")) {
    const line = raw.replace(/\s+$/, "");
    if (line.trim() === "") continue;
    if (line === "phases:") {
      mode = "phases";
      continue;
    }
    if (line === "skills:") {
      mode = "skills";
      continue;
    }
    if (mode === "phases" && line.startsWith("  - ")) {
      const m = /from:\s*(\S+?),\s*to:\s*(\S+?),\s*sec:\s*(-?\d+)/.exec(line);
      if (m) phases.push({ from: m[1] as string, to: m[2] as string, sec: Number(m[3]) });
      continue;
    }
    if (mode === "skills" && line.startsWith("  ")) {
      const m = /^\s+(\S+):\s*(\d+)$/.exec(line);
      if (m) skills[m[1] as string] = Number(m[2]);
      continue;
    }
    const m = /^([a-z_]+):\s*(.+)$/.exec(line);
    if (m) {
      mode = "top";
      scalars[m[1] as string] = (m[2] as string).trim();
    }
  }

  const windowId = scalars.window_id;
  const client = scalars.client;
  if (windowId === undefined || client === undefined) return undefined;

  const num = (key: string): number | undefined => {
    const v = scalars[key];
    if (v === undefined) return undefined;
    const n = Number(v);
    return Number.isFinite(n) ? n : undefined;
  };

  return {
    windowId,
    client,
    branch: scalars.branch,
    pr: num("pr"),
    parentIssue: num("parent_issue"),
    kind: scalars.kind,
    openedAt: num("opened_at"),
    closedAt: num("closed_at"),
    phases,
    sessions: num("sessions"),
    prompts: num("prompts"),
    toolCalls: num("tool_calls"),
    toolFailures: num("tool_failures"),
    interrupts: num("interrupts"),
    skills: block.includes("skills:") ? skills : undefined,
  };
}

/**
 * 本文から H2 節を読み返す。
 *
 * @remarks
 * 書く側（`renderBody`）と対で置きます。節見出しの形が片方だけ変われば、週次が読む節が
 * 黙って空になるためです。
 *
 * 知らない見出しは捨て、空の節は返しません。「書かれていない」と「該当なしと書かれた」を
 * 呼び出し側が区別できるようにするためで、空の節を空文字で返すと両者が同じ形になります。
 */
export function parseSections(body: string): Record<string, string> {
  const known = new Set<string>(BODY_SECTIONS);
  const sections: Record<string, string> = {};
  let current: string | undefined;
  let buffer: string[] = [];

  const flush = () => {
    if (current !== undefined) {
      const text = buffer.join("\n").trim();
      if (text !== "") sections[current] = text;
    }
    buffer = [];
  };

  for (const raw of body.split("\n")) {
    const line = raw.replace(/\s+$/, "");
    const heading = /^##\s+(.+?)\s*$/.exec(line);
    if (heading !== null) {
      flush();
      current = known.has(heading[1] as string) ? (heading[1] as string) : undefined;
      continue;
    }
    if (current !== undefined) buffer.push(line);
  }
  flush();

  return sections;
}
