/**
 * Claude / Codex のセッション記録を、同一のイベント形へ正規化する。
 *
 * @remarks
 * 2 つのクライアントは記録の形が違います。Claude は全エントリに `sessionId` と `gitBranch` を
 * 刻み、スキル呼出を `tool_use` として持ちます。Codex は `session_meta` に一度だけ識別子を書き、
 * スキル呼出に相当する構造化記録を持ちません。差はここで吸収し、上位は形を 1 つだけ見ます。
 *
 * 取れないものは 0 ではなく `undefined` にします。「Codex でスキルが呼ばれていない」と
 * 「Codex ではスキル呼出を観測できない」を同じ数字で表すと、棚卸しの判断を誤ります。
 */

/** どちらのクライアントの記録か。 */
export type Client = "claude" | "codex";

/** 正規化されたイベントの種別。 */
export type EventKind =
  | "prompt"
  | "tool_call"
  | "tool_result"
  | "skill_invoke"
  | "interrupt"
  | "compact";

/** クライアントに依らない 1 件のイベント。 */
export type Event = {
  readonly client: Client;
  readonly kind: EventKind;
  /** epoch 秒。時刻を持たない記録は取り込まない。 */
  readonly at: number;
  readonly sessionId?: string;
  readonly branch?: string;
  readonly cwd?: string;
  readonly tool?: string;
  readonly skill?: string;
  /** `tool_result` のみ。成否が観測できない記録では `undefined`。 */
  readonly ok?: boolean;
  readonly durationMs?: number;
  /**
   * `prompt` のみ。候補抽出が本文を必要とするため持ちます。
   *
   * 集計はこれを読みません。数える処理に本文が要らないのは意図的で、
   * 本文が要るのは「どのターンを AI に読ませるか」を選ぶときだけです。
   */
  readonly text?: string;
};

/** どのイベントにも共通する属性。種別ごとの項目を足す前の土台。 */
type EventBase = Pick<Event, "client" | "at" | "sessionId" | "branch" | "cwd">;

/** 1 セッションぶんの決定論的な事実。取得できない指標は `undefined`。 */
export type SessionFacts = {
  readonly client: Client;
  readonly sessionId?: string;
  readonly startedAt?: number;
  readonly endedAt?: number;
  readonly branches: readonly string[];
  readonly prompts: number;
  readonly toolCalls: number;
  readonly interrupts: number;
  readonly compactions: number;
  /** 失敗したツール呼出。成否を観測できないクライアントでは `undefined`。 */
  readonly toolFailures?: number;
  /** スキル名 → 呼出回数。観測できないクライアントでは `undefined`。 */
  readonly skillCalls?: Readonly<Record<string, number>>;
};

const toEpochSec = (iso: unknown): number | undefined => {
  if (typeof iso !== "string") return undefined;
  const ms = Date.parse(iso);
  return Number.isNaN(ms) ? undefined : Math.floor(ms / 1000);
};

const isRecord = (v: unknown): v is Record<string, unknown> =>
  typeof v === "object" && v !== null && !Array.isArray(v);

/**
 * Claude のトランスクリプト 1 行を正規化する。
 *
 * @remarks
 * 1 行が複数のイベントを含みます（1 つの assistant エントリに複数の `tool_use`）。
 * 該当が無ければ空配列を返し、壊れた行は黙って捨てます。1 行のために
 * セッション全体の集計を落とさないためです。
 */
/** 1 行を JSON として読む。読めなければ `undefined`。壊れた 1 行で集計を落とさないため。 */
function readLine(line: string): Record<string, unknown> | undefined {
  try {
    const raw: unknown = JSON.parse(line);
    return isRecord(raw) ? raw : undefined;
  } catch {
    return undefined;
  }
}

/** Claude の 1 行から、どのイベントにも共通する属性を取る。時刻が無ければ `undefined`。 */
function claudeBase(raw: Record<string, unknown>): EventBase | undefined {
  const at = toEpochSec(raw.timestamp);
  if (at === undefined) return undefined;
  return {
    client: "claude",
    at,
    sessionId: typeof raw.sessionId === "string" ? raw.sessionId : undefined,
    branch: typeof raw.gitBranch === "string" && raw.gitBranch !== "" ? raw.gitBranch : undefined,
    cwd: typeof raw.cwd === "string" ? raw.cwd : undefined,
  };
}

/** assistant の content ブロック 1 つをイベントにする。該当しなければ `undefined`。 */
function claudeBlockEvent(base: EventBase, block: unknown): Event | undefined {
  if (!isRecord(block)) return undefined;
  if (block.type === "tool_use") {
    const tool = typeof block.name === "string" ? block.name : undefined;
    const input = isRecord(block.input) ? block.input : undefined;
    const skill = typeof input?.skill === "string" ? input.skill : undefined;
    return { ...base, kind: tool === "Skill" ? "skill_invoke" : "tool_call", tool, skill };
  }
  if (block.type === "tool_result") return { ...base, kind: "tool_result", ok: block.is_error !== true };
  if (block.type === "text" && typeof block.text === "string" && block.text.includes("Request interrupted by user")) {
    return { ...base, kind: "interrupt" };
  }
  return undefined;
}

export function parseClaudeLine(line: string): Event[] {
  const raw = readLine(line);
  if (raw === undefined) return [];
  const base = claudeBase(raw);
  if (base === undefined) return [];

  // 人の発話は `user` エントリのうち content が素の文字列のもの。`last-prompt` は使わない。
  // あれは「直近の入力」を再開のために書き戻すチェックポイントであり、発話ごとではなく
  // 何度も上書きされる（実測 11,382 件に対し、素の文字列を持つ user は 3,598 件）。
  // 加えて timestamp を持たないため、そもそも期間で絞れない。
  if (raw.type === "user" && typeof (isRecord(raw.message) ? raw.message.content : undefined) === "string") {
    const text = (raw.message as { content: string }).content;
    return [{ ...base, kind: "prompt", text }];
  }
  if (raw.subtype === "compact_boundary") return [{ ...base, kind: "compact" }];

  const content = isRecord(raw.message) ? raw.message.content : undefined;
  if (!Array.isArray(content)) return [];
  return content.map((b) => claudeBlockEvent(base, b)).filter((e): e is Event => e !== undefined);
}

/**
 * Codex の rollout 1 行を正規化する。
 *
 * @remarks
 * ツールの成否は hook payload には無く、rollout の出力本文が `Script completed` /
 * `Script failed` で始まるかで判ります。スキル呼出はどちらにも無いため、ここは決して
 * `skill_invoke` を生みません。
 */
/** Codex の `event_msg` をイベントにする。該当しなければ `undefined`。 */
function codexEventMsg(base: EventBase, payload: Record<string, unknown> | undefined): Event | undefined {
  if (payload?.type === "user_message") return { ...base, kind: "prompt" };
  if (payload?.type === "turn_aborted") {
    const ms = typeof payload.duration_ms === "number" ? payload.duration_ms : undefined;
    return { ...base, kind: "interrupt", durationMs: ms };
  }
  if (payload?.type === "context_compacted") return { ...base, kind: "compact" };
  return undefined;
}

/**
 * Codex の `response_item` をイベントにする。該当しなければ `undefined`。
 *
 * @remarks
 * ツールの成否は hook payload には無く、出力本文が `Script completed` / `Script failed` で
 * 始まるかでしか判りません。どちらでもなければ「観測できなかった」として `undefined` に
 * します——失敗が無かったことにはしません。
 */
function codexResponseItem(base: EventBase, payload: Record<string, unknown>): Event | undefined {
  if (payload.type === "custom_tool_call" || payload.type === "function_call") {
    const tool = typeof payload.name === "string" ? payload.name : undefined;
    return { ...base, kind: "tool_call", tool };
  }
  if (payload.type === "custom_tool_call_output" || payload.type === "function_call_output") {
    const first = Array.isArray(payload.output) && isRecord(payload.output[0]) ? payload.output[0].text : payload.output;
    const text = typeof first === "string" ? first : "";
    const ok = text.startsWith("Script failed") ? false : text.startsWith("Script completed") ? true : undefined;
    return { ...base, kind: "tool_result", ok };
  }
  return undefined;
}

export function parseCodexLine(line: string): Event[] {
  const raw = readLine(line);
  if (raw === undefined) return [];
  const at = toEpochSec(raw.timestamp);
  if (at === undefined) return [];

  const payload = isRecord(raw.payload) ? raw.payload : undefined;
  const base: EventBase = {
    client: "codex",
    at,
    sessionId: typeof payload?.session_id === "string" ? payload.session_id : undefined,
    cwd: typeof payload?.cwd === "string" ? payload.cwd : undefined,
  };

  if (raw.type === "event_msg") {
    const e = codexEventMsg(base, payload);
    return e === undefined ? [] : [e];
  }
  if (raw.type === "response_item" && payload) {
    const e = codexResponseItem(base, payload);
    return e === undefined ? [] : [e];
  }
  return [];
}

/** イベント列を種別ごとに数えた素の値。 */
type Tally = {
  skillCalls: Record<string, number>;
  prompts: number;
  toolCalls: number;
  interrupts: number;
  compactions: number;
  failures: number;
  /** 成否を 1 度でも観測できたか。観測できなければ失敗数は 0 ではなく `undefined` になる。 */
  sawResultStatus: boolean;
};

/** イベントを種別ごとに数える。`skill_invoke` はツール呼び出しでもあるので両方に入る。 */
function tally(events: readonly Event[]): Tally {
  const t: Tally = {
    skillCalls: {},
    prompts: 0,
    toolCalls: 0,
    interrupts: 0,
    compactions: 0,
    failures: 0,
    sawResultStatus: false,
  };
  for (const e of events) {
    if (e.kind === "prompt") t.prompts += 1;
    else if (e.kind === "tool_call") t.toolCalls += 1;
    else if (e.kind === "interrupt") t.interrupts += 1;
    else if (e.kind === "compact") t.compactions += 1;
    else if (e.kind === "skill_invoke") {
      t.toolCalls += 1;
      if (e.skill !== undefined) t.skillCalls[e.skill] = (t.skillCalls[e.skill] ?? 0) + 1;
    } else if (e.kind === "tool_result" && e.ok !== undefined) {
      t.sawResultStatus = true;
      if (!e.ok) t.failures += 1;
    }
  }
  return t;
}

/**
 * イベント列を 1 セッションぶんの事実に畳む。
 *
 * @remarks
 * `skillCalls` と `toolFailures` は、そのクライアントで観測できる場合にのみ生えます。
 * Codex は前者を観測できないため常に `undefined` になり、「0 回」とは区別されます。
 */
export function summarizeSession(client: Client, events: readonly Event[]): SessionFacts {
  const times = events.map((e) => e.at);
  const branches = [...new Set(events.map((e) => e.branch).filter((b): b is string => b !== undefined))].sort((a, b) => a.localeCompare(b));
  const { skillCalls, prompts, toolCalls, interrupts, compactions, failures, sawResultStatus } = tally(events);

  return {
    client,
    sessionId: events.find((e) => e.sessionId !== undefined)?.sessionId,
    startedAt: times.length > 0 ? Math.min(...times) : undefined,
    endedAt: times.length > 0 ? Math.max(...times) : undefined,
    branches,
    prompts,
    toolCalls,
    interrupts,
    compactions,
    toolFailures: sawResultStatus ? failures : undefined,
    skillCalls: client === "codex" ? undefined : skillCalls,
  };
}
