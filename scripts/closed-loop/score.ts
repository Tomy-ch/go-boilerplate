/**
 * Feedback Issue を横断し、検討課題として並べるためのスコアを付ける。
 *
 * @remarks
 * スコアは順位づけの道具であって、良し悪しの判定ではありません。何を改善するかは
 * レトロで人が決めます（ADR-0008）。ここが担うのは「どれから見るか」だけです。
 *
 * 評価要素は要件 §5.5 の 4 つ。Frequency / Impact / Human Intervention / Recurrence。
 * 重みは運用データで調整する前提なので、既定値を定数に置いて差し替え可能にしてあります。
 *
 * 判定はここに置き、gh の呼び出しは入口が担います。
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
};

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
 * クラスタリングは AI の担当で、ここは決定論的に置ける鍵だけを扱います。両者を混ぜると、
 * 「数えた値」と「読解した値」が同じスコアの中で見分けられなくなります。
 */
export function clusterKey(issue: FeedbackIssue): string {
  return issue.kinds.length === 0 ? "unclassified" : [...issue.kinds].sort().join("+");
}

/**
 * Issue 群を鍵ごとにまとめ、スコアを付けて降順に並べる。
 *
 * @remarks
 * スコアは 4 要素の重み付き和です。再発を最も重く見るのは、要件 §10 が
 * 「同じ誤読・不要な介入が複数セッションで反復したら、個人の利用ミスより基盤側の
 * 改善候補として優先的に扱う」と定めているためです。
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
