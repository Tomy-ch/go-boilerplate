/**
 * セッションの事実を期間で切り、横断して畳む。
 *
 * @remarks
 * 期間は絶対日付で受け取ります。過去分の取り直しが主目的である以上、「いつ基準の 1 週間か」が
 * 実行時刻に依存してはならないためです（相対指定は入口側の糖衣にとどめます）。
 */

import type { SessionFacts } from "./events";

/** 集計対象の期間。両端とも epoch 秒で、`from` を含み `to` を含みます。 */
export type Period = {
  readonly from: number;
  readonly to: number;
};

/** 期間内のセッションを畳んだ結果。観測できない指標は `undefined` のまま残します。 */
export type PeriodSummary = {
  readonly period: Period;
  readonly sessions: number;
  readonly byClient: Readonly<Record<string, number>>;
  readonly prompts: number;
  readonly toolCalls: number;
  readonly interrupts: number;
  readonly compactions: number;
  /** 成否を観測できたセッションが 1 つも無ければ `undefined`。 */
  readonly toolFailures?: number;
  /** 失敗率。分母となる観測ができなければ `undefined`。 */
  readonly toolFailureRate?: number;
  /** スキル名 → 呼出回数。観測できたセッションが無ければ `undefined`。 */
  readonly skillCalls?: Readonly<Record<string, number>>;
  readonly branches: readonly string[];
};

const DAY_SEC = 86_400;

/**
 * 日の境界をどのタイムゾーンで切るか。UTC からの秒。
 *
 * @remarks
 * このリポジトリの運用は JST です。UTC で日を切ると `--to 2026-08-19` が実際には
 * JST 8/19 09:00 〜 8/20 08:59 を指し、**その日の午前が丸ごと落ちます**。指定した日と
 * 集計された範囲が 9 時間ずれるのは、数字が出てしまうぶん静かな失敗になります。
 *
 * 固定値にしてあるのは、実行環境の `TZ` に従わせると同じ引数が機械によって別の期間を
 * 指すためです。期間は絶対日付で指定する設計なので、その解釈も固定でなければ意味がありません。
 */
export const DAY_BOUNDARY_OFFSET_SEC = 9 * 3600;

/**
 * 期間を解決する。
 *
 * @param from `YYYY-MM-DD`。省略時は `to` の 7 日前。
 * @param to `YYYY-MM-DD`。省略時は `now` の日。
 * @param now 基準時刻（epoch 秒）。呼び出し側が渡すことで、同じ入力が常に同じ期間になる。
 * @returns 解決された期間。境界は {@link DAY_BOUNDARY_OFFSET_SEC} のタイムゾーンで切り、
 *   `to` はその日の終端まで含む。
 * @throws 日付として解釈できない、または `from` が `to` より後の場合。
 */
export function resolvePeriod(from: string | undefined, to: string | undefined, now: number): Period {
  const endDay = to === undefined ? startOfDay(now) : parseDay(to);
  const startDay = from === undefined ? endDay - 7 * DAY_SEC : parseDay(from);
  const end = endDay + DAY_SEC - 1;
  // 逆転しうるのは `from` が明示された場合だけ。省略時は `to` の 7 日前を置くので、
  // 構造上そちらが後ろに来ることはない。前提を条件に書いておくことで、
  // 到達しない分岐をメッセージ側に抱えずに済む。
  if (from !== undefined && startDay > end) {
    throw new Error(`期間が逆転している: from=${from} to=${to ?? "(既定)"}`);
  }
  return { from: startDay, to: end };
}

function parseDay(day: string): number {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(day)) throw new Error(`日付は YYYY-MM-DD で指定する: ${day}`);
  const ms = Date.parse(`${day}T00:00:00Z`);
  if (Number.isNaN(ms)) throw new Error(`日付として解釈できない: ${day}`);
  return Math.floor(ms / 1000) - DAY_BOUNDARY_OFFSET_SEC;
}

function startOfDay(epochSec: number): number {
  const shifted = epochSec + DAY_BOUNDARY_OFFSET_SEC;
  return shifted - (shifted % DAY_SEC) - DAY_BOUNDARY_OFFSET_SEC;
}

/**
 * ある時刻が期間に入るか。両端を含む。
 *
 * @remarks
 * 期間の端の扱いを 1 箇所に集めるために置いています。同じ判定を呼び出し側が書き直すと、
 * 等号の有無が食い違って週境界の記録が静かに落ちたり二重に数えられたりします。
 */
export function withinPeriod(at: number, period: Period): boolean {
  return at >= period.from && at <= period.to;
}

/**
 * セッションが期間に掛かるか。
 *
 * @remarks
 * 開始と終了のどちらかが期間に入っていれば対象とします。セッションは 1 週間の期間を跨ぐ
 * 長さになりうるので（ADR-0010 (development-window-as-feedback-unit)）、跨ぐものを落とすと
 * 長い作業ほど集計から消えるという逆向きの偏りが出ます。
 */
export function overlapsPeriod(facts: SessionFacts, period: Period): boolean {
  const start = facts.startedAt;
  const end = facts.endedAt;
  if (start === undefined || end === undefined) return false;
  return start <= period.to && end >= period.from;
}

/**
 * 期間内のセッションを横断して畳む。
 *
 * @remarks
 * `toolFailures` と `skillCalls` は、観測できたセッションが 1 つでもあれば数値になり、
 * 1 つも無ければ `undefined` のままになります。Codex だけの期間で「スキル呼出 0 件」と
 * 表示してしまうと、棚卸しで実際に使われていないスキルと区別がつかなくなるためです。
 */
export function summarizePeriod(all: readonly SessionFacts[], period: Period): PeriodSummary {
  const inPeriod = all.filter((f) => overlapsPeriod(f, period));
  const byClient: Record<string, number> = {};
  const skillCalls: Record<string, number> = {};
  const branches = new Set<string>();
  let prompts = 0;
  let toolCalls = 0;
  let interrupts = 0;
  let compactions = 0;
  let failures = 0;
  let sawFailureObservation = false;
  let sawSkillObservation = false;

  for (const f of inPeriod) {
    byClient[f.client] = (byClient[f.client] ?? 0) + 1;
    prompts += f.prompts;
    toolCalls += f.toolCalls;
    interrupts += f.interrupts;
    compactions += f.compactions;
    for (const b of f.branches) branches.add(b);
    if (f.toolFailures !== undefined) {
      sawFailureObservation = true;
      failures += f.toolFailures;
    }
    if (f.skillCalls !== undefined) {
      sawSkillObservation = true;
      for (const [name, count] of Object.entries(f.skillCalls)) {
        skillCalls[name] = (skillCalls[name] ?? 0) + count;
      }
    }
  }

  return {
    period,
    sessions: inPeriod.length,
    byClient,
    prompts,
    toolCalls,
    interrupts,
    compactions,
    toolFailures: sawFailureObservation ? failures : undefined,
    toolFailureRate: sawFailureObservation && toolCalls > 0 ? failures / toolCalls : undefined,
    skillCalls: sawSkillObservation ? skillCalls : undefined,
    branches: [...branches].sort(),
  };
}

/**
 * 宣言されているのに期間内で一度も呼ばれなかったスキルを挙げる。
 *
 * @remarks
 * 返すのは候補の母集合であって、削除の根拠ではありません。呼出ゼロを何と併せて読むかは
 * `docs/design/closed-loop.md`「Skills are judged against their class」にあります。
 */
export function uncalledSkills(declared: readonly string[], summary: PeriodSummary): string[] {
  if (summary.skillCalls === undefined) return [];
  const called = new Set(Object.keys(summary.skillCalls));
  return declared.filter((s) => !called.has(s)).sort();
}
