/**
 * AI に読ませる価値のあるターンを、決定論的に選ぶ。
 *
 * @remarks
 * 意味の分類は AI がやりますが、何を読ませるかは AI に選ばせません。窓 1 件でも人の発話は
 * 数十件あり、全部を読ませればコストが見合いません。ここで絞ってから渡すのは
 * 節約のためだけではなく、絞り込みが決定論なら「なぜこのターンが選ばれたか」を後から
 * 説明できるからでもあります。
 *
 * 選ぶのは 3 種類。人が是正した発話、中断の直前、ツールが失敗した直後。いずれも
 * 「うまくいかなかった瞬間」の代理指標であって、摩擦そのものではありません。だから
 * 見落としもします。取りこぼしを許してでも読む量を抑える、という判断です。
 */

import type { Event } from "./events";

/** なぜこのターンが選ばれたか。AI へ渡す際に添える。 */
export type CandidateReason = "corrective" | "before-interrupt" | "after-failure";

/** AI に読ませる 1 件。 */
export type Candidate = {
  readonly at: number;
  readonly reason: CandidateReason;
  readonly text: string;
};

/**
 * 人が是正したことを示す語。
 *
 * @remarks
 * 丁寧な訂正、質問の形をした指摘、黙って直した場合は捕まりません。ここは網羅ではなく
 * 「読む価値が高い順に絞る」ための足切りです。語を増やすほど絞りが緩みます。
 */
export const CORRECTIVE_MARKERS: readonly string[] = [
  "違う",
  "ではなく",
  "じゃなく",
  "勘違い",
  "間違",
  "戻して",
  "そうじゃ",
  "誤読",
  "なんで",
  "why not",
] as const;

/** 既定の上限。1 窓ぶんとして読ませても費用が見合う量。 */
export const DEFAULT_LIMIT = 12;

/**
 * 人ではなくシステムが注入した本文の目印。
 *
 * @remarks
 * `user` エントリの本文には、人が打ったものだけでなくハーネスが差し込んだものも混ざります。
 * 実データでは task 通知と再開時の要約が該当し、絞り込んだ 12 件のうち 4 件を占めていました。
 * これらは人の発話ではないので、是正の語を含んでいても摩擦の証拠になりません。
 */
export const INJECTED_MARKERS: readonly string[] = [
  "<task-notification>",
  "<local-command-caveat>",
  "<command-name>",
  "<system-reminder>",
  "This session is being continued from a previous conversation",
] as const;

/** ハーネスが注入した本文か。 */
export function isInjected(text: string): boolean {
  return INJECTED_MARKERS.some((m) => text.includes(m));
}

/**
 * 秘密情報らしき形。
 *
 * @remarks
 * 候補は public な Issue へ逐語で投稿されうるので、パターンを緩めるときは**疑わしきは落とす**
 * を崩さないこと——落とした 1 件は他の候補で埋まりますが、出てしまった 1 件は取り消せません。
 * この関門が置かれている理由と実測の根拠は `docs/design/security.md`「Secrets」にあります。
 */
export const SECRET_PATTERNS: readonly RegExp[] = [
  /\b(?:sk|pk|rk)-[A-Za-z0-9_-]{16,}/,                 // OpenAI 系
  /\bgh[pousr]_[A-Za-z0-9]{16,}/,                       // GitHub token
  /\bgithub_pat_[A-Za-z0-9_]{20,}/,
  /\bAKIA[0-9A-Z]{16}\b/,                               // AWS access key id
  /\bASIA[0-9A-Z]{16}\b/,
  /\bxox[abposr]-[A-Za-z0-9-]{10,}/,                    // Slack
  /\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}/, // JWT
  /-----BEGIN [A-Z ]*PRIVATE KEY-----/,
  /\b[A-Za-z][A-Za-z0-9+.-]*:\/\/[^\s:@/]+:[^\s:@/]+@/,   // URL の userinfo
  // 単語境界を前に置かないのは、実測で見つかった実例が `SONAR_TOKEN=` だったため。
  // `\btoken\b` は `_TOKEN` に当たらず、環境変数名の形をした鍵をまるごと取り逃す。
  /(?:token|secret|credential|password|passwd|api[_-]?key)\s*[:=]\s*["']?[A-Za-z0-9_\-.+/]{12,}/i,
  // `pass` / `pwd` は語として短く、`passing` や `cwd` のような無関係な語に当たる。前後を
  // 区切って `DB_PASS=` や `pwd:` の形だけを拾う。
  /(?:^|[\s_\-.])(?:pass|pwd)\s*[:=]\s*["']?[A-Za-z0-9_\-.+/]{8,}/i,
  // `curl -u user:secret` の形。URL の userinfo とは別の経路で、こちらは `://` を持たない。
  /\s-u\s+[^\s:@/]+:[^\s]{6,}/,
  /\bBearer\s+[A-Za-z0-9_\-.=]{20,}/i,
] as const;

/** 秘密情報らしき形を含むか。 */
export function looksSecret(text: string): boolean {
  return SECRET_PATTERNS.some((re) => re.test(text));
}

/** 本文が是正の合図を含むか。注入された本文は人の発話ではないので含めない。 */
export function isCorrective(text: string): boolean {
  return !isInjected(text) && CORRECTIVE_MARKERS.some((m) => text.includes(m));
}

/**
 * 本文を要約せずに切り詰める。
 *
 * @remarks
 * 切るのは長さだけで、意味には触れません。要約は AI の仕事であり、ここで先に要約すると
 * 「決定論的に選んだ」と言えなくなります。
 */
export function excerpt(text: string, maxChars: number): string {
  const flat = text.replace(/\s+/g, " ").trim();
  return flat.length <= maxChars ? flat : `${flat.slice(0, maxChars)}…`;
}

/**
 * 読ませる候補を選ぶ。
 *
 * @param events 窓に属するイベント列。時刻順である必要はない。
 * @param limit 上限。多い場合は是正 → 中断前 → 失敗後 の順に残す。
 * @param maxChars 1 件あたりの本文の長さ。
 *
 * @remarks
 * 優先順位は「人が明示的に是正した」を最上位に置きます。中断と失敗は機械が観測した
 * 兆候にすぎませんが、是正は人がそう言った事実だからです。
 */
export function selectCandidates(
  events: readonly Event[],
  limit: number = DEFAULT_LIMIT,
  maxChars = 300,
): Candidate[] {
  const ordered = [...events].sort((a, b) => a.at - b.at);
  const prompts = ordered.filter(
    (e) =>
      e.kind === "prompt" &&
      e.text !== undefined &&
      e.text !== "" &&
      !isInjected(e.text) &&
      // 秘密情報らしき本文は、どの選定理由に当たっても候補にしない。投稿は取り消せない。
      !looksSecret(e.text),
  );

  const corrective: Candidate[] = [];
  const beforeInterrupt: Candidate[] = [];
  const afterFailure: Candidate[] = [];
  const taken = new Set<number>();

  const push = (into: Candidate[], e: Event, reason: CandidateReason) => {
    if (taken.has(e.at)) return;
    taken.add(e.at);
    into.push({ at: e.at, reason, text: excerpt(e.text as string, maxChars) });
  };

  for (const p of prompts) {
    if (isCorrective(p.text as string)) push(corrective, p, "corrective");
  }

  // 中断の直前の発話。何を止めたのかは、その手前で人が言ったことに書いてある。
  for (const e of ordered) {
    if (e.kind !== "interrupt") continue;
    const prior = [...prompts].reverse().find((p) => p.at <= e.at);
    if (prior !== undefined) push(beforeInterrupt, prior, "before-interrupt");
  }

  // 失敗の直後の発話。人がそこで何と言ったかに、失敗の意味が現れる。
  for (const e of ordered) {
    if (e.kind !== "tool_result" || e.ok !== false) continue;
    const next = prompts.find((p) => p.at >= e.at);
    if (next !== undefined) push(afterFailure, next, "after-failure");
  }

  return [...corrective, ...beforeInterrupt, ...afterFailure].slice(0, Math.max(limit, 0));
}

/**
 * 候補を Issue コメントの本文にする。
 *
 * @remarks
 * 本文ではなくコメントへ載せるのは、本文が AI の出力面だからです。入力と出力を同じ
 * テキストに混ぜると、AI が自分の入力を上書きしうるうえ、後から「何を読んで書いたか」を
 * 追えなくなります。
 */
export function renderCandidateComment(candidates: readonly Candidate[]): string {
  if (candidates.length === 0) {
    return "## 読解候補\n\n決定論的な絞り込みで該当したターンはありませんでした。";
  }
  const lines = [
    "## 読解候補",
    "",
    "以下は機械的に選んだターンです。要約ではなく逐語の抜粋で、選定理由を添えてあります。",
    "本文の H2 節を埋めるときの材料にしてください。",
    "",
  ];
  for (const c of candidates) {
    lines.push(`- \`${c.reason}\` <!-- at:${c.at} -->`);
    lines.push(`  > ${c.text}`);
  }
  return lines.join("\n");
}
