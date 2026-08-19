/**
 * 打刻ファイルから開発の窓を組み立て、フェーズ区間を導出する。
 *
 * @remarks
 * 窓とは 1 つの作業単位であり、セッションではありません。何を 1 つと数えるかの決定は
 * ADR-0010 (development-window-as-feedback-unit) にあります。打刻の側は
 * `.agents/closed-loop/marks.sh` が担い、このモジュールはその出力を読む側だけを持ちます。
 */

/**
 * 打刻の正準順序。フェーズ区間は、この順序で隣り合う「実在するマーカー」の間に張られます。
 *
 * @remarks
 * 順序を実行時のタイムスタンプではなく宣言で持つのは、時刻が前後したときに
 * 「順序がおかしい」と言えるようにするためです。時刻順に並べ替えてしまうと、
 * 異常が正常な区間に化けて消えます。
 */
export const MARK_ORDER: readonly string[] = [
  "openedAt",
  "planApprovedAt",
  "implStartedAt",
  "commitAt",
  "reviewStartedAt",
  "prOpenedAt",
  "mergedAt",
  "closedAt",
] as const;

/** 1 つの窓。`marks` の値は打刻された epoch(秒) を打刻順に並べたもの。 */
export type Window = {
  readonly id: string;
  readonly marks: Readonly<Record<string, readonly number[]>>;
};

/** 隣り合う 2 つのマーカーが張る区間。 */
export type Phase = {
  readonly from: string;
  readonly to: string;
  readonly startedAt: number;
  readonly endedAt: number;
  /** 秒。`endedAt < startedAt` のときは負になりうるため、利用側は {@link anomaliesOf} を先に見ること。 */
  readonly durationSec: number;
};

/** 窓に見つかった、そのままでは指標にできない状態。 */
export type Anomaly =
  | { readonly kind: "missing-open"; readonly detail: string }
  | { readonly kind: "unclosed"; readonly detail: string }
  | { readonly kind: "out-of-order"; readonly detail: string };

/**
 * 打刻ファイルの中身を epoch の配列にする。
 *
 * @remarks
 * 空行と数値でない行は落とします。打刻は `date +%s >> file` の追記なので、
 * 並行実行で行が壊れることは原理的に起きませんが、人が手で触った場合に備えます。
 */
export function parseMarkFile(content: string): number[] {
  return content
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
    .map((line) => Number(line))
    .filter((n) => Number.isInteger(n) && n > 0);
}

/**
 * 打刻ファイル群から窓を組み立てる。
 *
 * @param id 窓 ID（`marks/` 直下のディレクトリ名）。
 * @param files マーカー名 → ファイル内容。正準順序に無い名前は捨てます。
 */
export function toWindow(id: string, files: Readonly<Record<string, string>>): Window {
  const marks: Record<string, readonly number[]> = {};
  for (const name of MARK_ORDER) {
    const content = files[name];
    if (content === undefined) continue;
    const epochs = parseMarkFile(content);
    if (epochs.length > 0) marks[name] = epochs;
  }
  return { id, marks };
}

/**
 * 各マーカーの代表時刻。最初の打刻を使う。
 *
 * @remarks
 * 「そのフェーズに最初に到達した時刻」がフェーズ境界の意味であり、
 * 繰り返しは回数として別に数えます（レビュー 2 周を平均で潰さないため）。
 */
export function representativeAt(window: Window, mark: string): number | undefined {
  return window.marks[mark]?.[0];
}

/** 打刻された回数。マーカーが無ければ 0。 */
export function stampCount(window: Window, mark: string): number {
  return window.marks[mark]?.length ?? 0;
}

/**
 * 隣り合う実在マーカーの間にフェーズ区間を張る。
 *
 * @remarks
 * 欠けたマーカーは飛ばして次の実在マーカーへ繋ぎます。欠測は例外ではなく常態です
 * （ADR-0010 (development-window-as-feedback-unit)）。飛ばした区間は `from` / `to` に
 * どのマーカー間かが残るので、後から補完できます。
 */
export function phasesOf(window: Window): Phase[] {
  const present = MARK_ORDER.filter((m) => representativeAt(window, m) !== undefined);
  const phases: Phase[] = [];
  for (let i = 0; i + 1 < present.length; i += 1) {
    const from = present[i] as string;
    const to = present[i + 1] as string;
    const startedAt = representativeAt(window, from) as number;
    const endedAt = representativeAt(window, to) as number;
    phases.push({ from, to, startedAt, endedAt, durationSec: endedAt - startedAt });
  }
  return phases;
}

/**
 * 指標にする前に人が見るべき状態を列挙する。
 *
 * @remarks
 * 異常を黙って捨てず、かつ正常な数字にも混ぜないための境目です。とくに `out-of-order` は、
 * 時刻順に並べ替える実装なら消えてしまう欠陥で、消えると負の所要時間が「短い作業」に化けます。
 */
export function anomaliesOf(window: Window): Anomaly[] {
  const anomalies: Anomaly[] = [];
  if (representativeAt(window, "openedAt") === undefined) {
    anomalies.push({ kind: "missing-open", detail: "openedAt が無く、窓の開始時刻が決まらない" });
  }
  if (representativeAt(window, "closedAt") === undefined) {
    anomalies.push({ kind: "unclosed", detail: "closedAt が無く、まだ開いているか閉じ損ねている" });
  }
  for (const phase of phasesOf(window)) {
    if (phase.durationSec < 0) {
      anomalies.push({
        kind: "out-of-order",
        detail: `${phase.from} が ${phase.to} より後に打刻されている`,
      });
    }
  }
  return anomalies;
}

/**
 * 窓が「存在した」以上のことを記録したか。`openedAt` / `closedAt` 以外の打刻を返す。
 *
 * @remarks
 * 開始と終了は窓の器であって中身ではありません。両端しか無い窓は、開いて閉じただけで
 * 何も起きなかったことを意味します。`/clear` を続けて打つ、セッションを開いて何もせず
 * 終える、といった場面で実際に生まれます。
 */
export function substantiveMarks(window: Window): string[] {
  return MARK_ORDER.filter((m) => m !== "openedAt" && m !== "closedAt" && stampCount(window, m) > 0);
}

/**
 * 送るに値する窓か。
 *
 * @param activity その窓の期間に観測されたセッションの活動。観測できなければ `undefined`。
 *
 * @remarks
 * 実質的な打刻があるか、期間内に活動があれば送ります。活動が観測できないとき
 * （トランスクリプトを渡されない手動実行）は打刻だけで判断し、送らない側へ倒します。
 * 空の窓は公開 issue に恒久的なノイズを残して手作業でしか消せず、送り損ねた窓は
 * 打刻が `tmp/` に残って `closed-loop-report` から見えるからです。
 */
export function isSubstantive(
  window: Window,
  activity?: { readonly prompts: number; readonly toolCalls: number },
): boolean {
  if (substantiveMarks(window).length > 0) return true;
  if (activity === undefined) return false;
  return activity.prompts > 0 || activity.toolCalls > 0;
}

/** 窓全体の所要時間（秒）。開始か終了が欠けていれば `undefined`。 */
export function totalDurationSec(window: Window): number | undefined {
  const opened = representativeAt(window, "openedAt");
  const closed = representativeAt(window, "closedAt");
  if (opened === undefined || closed === undefined) return undefined;
  return closed - opened;
}

/** 打刻ディレクトリを読む継ぎ目。実体は入口が `fs` から作って渡す。 */
export type MarksReader = {
  readonly listWindowIds: () => readonly string[];
  readonly listFiles: (windowId: string) => readonly string[];
  readonly readFile: (windowId: string, file: string) => string;
};

/**
 * 打刻ディレクトリ群から窓を組み立てる。
 *
 * @remarks
 * 読めないファイルは無いものとして扱い、その窓の残りと他の窓は取り込みます。打刻は複数の
 * プロセス（git フックと進行中のセッション）から書かれるので書き込み途中に当たることが
 * 普通に起こり、1 件の読み取り失敗で全体を落とすとループが同時実行に依存してしまいます
 * （ADR-0010 (development-window-as-feedback-unit)）。
 *
 * 窓 ID で整列するのは、読む側が実行のたびに違う順序を受け取らないようにするためです。
 */
export function collectWindows(read: MarksReader): Window[] {
  return [...read.listWindowIds()]
    .sort((a, b) => a.localeCompare(b))
    .map((id) => {
      const files: Record<string, string> = {};
      for (const f of read.listFiles(id)) {
        try {
          files[f] = read.readFile(id, f);
        } catch {
          // 読めないファイルは無いものとして扱う
        }
      }
      return toWindow(id, files);
    });
}
