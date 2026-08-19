/**
 * Feedback Issue を横断し、検討課題として並べるためのスコアを付ける。
 *
 * @remarks
 * スコアは順位づけの道具であって、良し悪しの判定ではありません。何を改善するかは
 * レトロで人が決めます（ADR-0008）。ここが担うのは「どれから見るか」だけです。
 *
 * 評価要素は 4 つ。Frequency / Impact / Human Intervention / Recurrence。
 * 重みは運用データで調整する前提なので、既定値を定数に置いて差し替え可能にしてあります。
 */

import type { Observation } from "./issue";

/** 所見の分類。Feedback Issue のラベル `feedback/*` に対応する。 */
export type FindingKind =
  | "skill"
  | "architecture"
  | "documentation"
  | "tooling"
  | "ai-misread"
  | "ci"
  | "developer-experience";

/** 集計に載せる 1 件の Feedback Issue。 */
export type FeedbackIssue = {
  readonly number: number;
  readonly kinds: readonly FindingKind[];
  readonly observation: Observation;
  /** Issue が作られた epoch(秒)。窓の `openedAt` とは別物で、こちらは所見が記録された時刻。 */
  readonly createdAt?: number;
  /** Issue が閉じられた epoch(秒)。開いたままなら `undefined`。 */
  readonly resolvedAt?: number;
  /** 本文の H2 節。読解が無ければ空。 */
  readonly sections?: Readonly<Record<string, string>>;
};

/** 分類ラベルの接頭辞。`feedback/skill` の `feedback/` の部分。 */
export const KIND_LABEL_PREFIX = "feedback/";

/** 分類が付いていない Issue をまとめる鍵。 */
export const UNCLASSIFIED = "unclassified";

/** 取りうる分類の全体。ラベルからの変換で許可リストとして使う。 */
export const FINDING_KINDS: readonly FindingKind[] = [
  "skill",
  "architecture",
  "documentation",
  "tooling",
  "ai-misread",
  "ci",
  "developer-experience",
] as const;

/**
 * Issue のラベル名から分類を取り出す。
 *
 * @remarks
 * 知らない分類は捨てます。ラベルは人が付けるので、綴り違いや将来増えた名前が混ざりえます。
 * ただし捨てた結果が空になっても、それは `unclassified` として集計され消えはしません。
 */
export function labelsToKinds(labels: readonly string[]): FindingKind[] {
  const known = new Set<string>(FINDING_KINDS);
  return labels
    .filter((n) => n.startsWith(KIND_LABEL_PREFIX))
    .map((n) => n.slice(KIND_LABEL_PREFIX.length))
    .filter((n): n is FindingKind => known.has(n));
}

/** スコアの重み。運用データで調整する前提の既定値。 */
export type Weights = {
  readonly frequency: number;
  readonly impact: number;
  readonly humanIntervention: number;
  readonly recurrence: number;
};

export const DEFAULT_WEIGHTS: Weights = {
  frequency: 1,
  impact: 2,
  humanIntervention: 3,
  recurrence: 4,
};

/** 同種の問題をまとめた 1 件。 */
export type Cluster = {
  readonly key: string;
  readonly issues: readonly number[];
  /** 件数。 */
  readonly frequency: number;
  /** 影響。ツール失敗の合計を代理指標とする。 */
  readonly impact: number;
  /** 人間の介入。中断の合計。 */
  readonly humanIntervention: number;
  /** 再発。2 件以上に跨って現れたか。単発と反復を分けるための 0/1。 */
  readonly recurrence: number;
  readonly score: number;
  /** 複数の Issue に跨っていれば反復事象。 */
  readonly isRecurring: boolean;
};

/**
 * 失敗率。ツール呼出が観測できていなければ `undefined`。
 *
 * @remarks
 * 率を出すのは、規模の違う窓を並べても比較できるようにするためです。失敗 100 件でも
 * 呼出 10,000 件なら 1% であり、失敗 20 件で呼出 200 件の窓の方が苦しんでいます。
 */
export function failureRate(observation: Observation): number | undefined {
  const calls = observation.toolCalls;
  const failures = observation.toolFailures;
  if (calls === undefined || failures === undefined || calls === 0) return undefined;
  return failures / calls;
}

/**
 * PR が開いてからマージされるまでの秒。観測できなければ `undefined`。
 *
 * @remarks
 * 実装が速くてもマージが遅ければリードタイムは縮みません。実装時間と並べて出すために、
 * フェーズから待ち時間だけを取り出します。
 */
export function mergeWaitSec(observation: Observation): number | undefined {
  return observation.phases.find((p) => p.from === "prOpenedAt" && p.to === "mergedAt")?.sec;
}

/**
 * クラスタリングの鍵。
 *
 * @remarks
 * 分類ラベルが付いていればそれを鍵にし、無ければ `unclassified` にまとめます。意味による
 * クラスタリングはここでは行いません（`docs/design/closed-loop.md`
 * 「Deterministic first, model second」）。
 */
export function clusterKey(issue: FeedbackIssue): string {
  return issue.kinds.length === 0 ? UNCLASSIFIED : [...issue.kinds].sort((a, b) => a.localeCompare(b)).join("+");
}

/**
 * Issue 群を鍵ごとにまとめ、スコアを付けて降順に並べる。
 *
 * @remarks
 * スコアは 4 要素の重み付き和で、重みは {@link DEFAULT_WEIGHTS}。配分の理由は
 * `scripts/README.md`「AI Feedback Loop」。
 */
export function clusterIssues(issues: readonly FeedbackIssue[], weights: Weights = DEFAULT_WEIGHTS): Cluster[] {
  const groups = new Map<string, FeedbackIssue[]>();
  for (const issue of issues) {
    const key = clusterKey(issue);
    groups.set(key, [...(groups.get(key) ?? []), issue]);
  }

  const clusters: Cluster[] = [];
  for (const [key, members] of groups) {
    const frequency = members.length;
    const impact = members.reduce((n, m) => n + (m.observation.toolFailures ?? 0), 0);
    const humanIntervention = members.reduce((n, m) => n + (m.observation.interrupts ?? 0), 0);
    const isRecurring = frequency > 1;
    const recurrence = isRecurring ? 1 : 0;
    clusters.push({
      key,
      issues: members.map((m) => m.number).sort((a, b) => a - b),
      frequency,
      impact,
      humanIntervention,
      recurrence,
      isRecurring,
      score:
        frequency * weights.frequency +
        impact * weights.impact +
        humanIntervention * weights.humanIntervention +
        recurrence * weights.recurrence,
    });
  }

  return clusters.sort((a, b) => b.score - a.score || a.key.localeCompare(b.key));
}

/**
 * 待ち時間が実装時間を上回っている Issue を挙げる。
 *
 * @remarks
 * 「PR が早く上がってもマージが遅ければ意味がない」を機械的に言い直したもの。
 * ここに載る窓は、実装を速くしても縮まないため、改善の向きが他と違います。
 */
export function waitDominated(issues: readonly FeedbackIssue[]): number[] {
  return issues
    .filter((i) => {
      const wait = mergeWaitSec(i.observation);
      if (wait === undefined) return false;
      const work = i.observation.phases
        .filter((p) => p.to !== "mergedAt")
        .reduce((n, p) => n + Math.max(p.sec, 0), 0);
      return wait > work;
    })
    .map((i) => i.number)
    .sort((a, b) => a - b);
}

/**
 * 着地から測り直しまでの日数。
 *
 * @remarks
 * ADR-0008 がループの条件に置いた「改善が効いたかを後で判定する」の待ち時間です。短すぎれば
 * 母数が足りず、長すぎれば次の改善と混ざります。14 日は最初の設定値であり、運用データで
 * 動かす前提で定数に置いてあります。
 */
export const REEVALUATION_DAYS = 14;

/** 着地した改善 1 件と、その後の再発。 */
export type Reevaluation = {
  /** 対象のクラスタ鍵。 */
  readonly key: string;
  /** 着地の合図になった Issue。同じ鍵のうち最後に閉じられたもの。 */
  readonly landedIssue: number;
  /** その Issue が閉じられた epoch(秒)。 */
  readonly landedAt: number;
  /** 着地後に同じ鍵で新しく作られた Issue。 */
  readonly recurred: readonly number[];
  /** 測り直しの時期が来ているか。来ていなければ、再発が無くても判定材料にならない。 */
  readonly due: boolean;
};

/**
 * 着地した改善を、その後の再発と並べて返す。
 *
 * @remarks
 * 「着地」の合図は Feedback Issue のクローズです（`docs/design/closed-loop.md`
 * 「What the loop may not do」）。
 *
 * `unclassified` は除きます。何についての所見か分かっていない以上、「同じものが再発したか」に
 * 答えられないためです。
 *
 * 判定はしません。再発の件数と、測り直しの時期が来たかを並べるだけです。
 */
export function reevaluations(issues: readonly FeedbackIssue[], now: number): Reevaluation[] {
  const byKey = new Map<string, FeedbackIssue[]>();
  for (const issue of issues) {
    const key = clusterKey(issue);
    if (key === UNCLASSIFIED) continue;
    byKey.set(key, [...(byKey.get(key) ?? []), issue]);
  }

  const out: Reevaluation[] = [];
  for (const [key, group] of byKey) {
    const landed = group
      .filter((i): i is FeedbackIssue & { resolvedAt: number } => i.resolvedAt !== undefined)
      .sort((a, b) => b.resolvedAt - a.resolvedAt)[0];
    if (landed === undefined) continue;

    const recurred = group
      .filter((i) => i.createdAt !== undefined && i.createdAt > landed.resolvedAt)
      .map((i) => i.number)
      .sort((a, b) => a - b);

    out.push({
      key,
      landedIssue: landed.number,
      landedAt: landed.resolvedAt,
      recurred,
      due: now >= landed.resolvedAt + REEVALUATION_DAYS * 86_400,
    });
  }

  return out.sort((a, b) => b.recurred.length - a.recurred.length || a.key.localeCompare(b.key));
}
