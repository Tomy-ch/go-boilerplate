/**
 * 窓ごとの Feedback Issue を、関心ごとの Integration Issue へ畳む。
 *
 * @remarks
 * 窓は観測の単位であって、行動の単位ではありません。1 本のブランチが無関係な関心を 3 つ生む
 * ことも、3 本のブランチが同じ関心を指すこともあります。レトロが着手するのは関心のほうなので、
 * そこへ寄せ直します。ADR-0010 が「several windows may belong to one issue」と言う "issue" は
 * この単位です。
 *
 * 畳んだ大元は閉じますが、**着地とは区別します**。`reevaluations` は「クローズ＝改善が着地した」
 * を合図に使っており、集計しただけの Issue が着地に見えると測り直しが空振りするためです。
 * 区別は GitHub の close reason で付けます——畳んだ側は `duplicate`、人が片付けたと宣言した
 * 側だけが `completed`。
 */

import { BODY_SECTIONS, IMPROVEMENT_SECTION, type Observation } from "./issue";

/** 統合 Issue に付けるラベル。`feedback` と混ざらないよう別名にする。 */
export const INTEGRATION_LABEL = "feedback-integration";

/** 畳んだ大元を閉じるときの理由。着地（`completed`）と区別する。 */
export const ROLLED_UP_REASON = "duplicate";

/** 畳む対象になった 1 件。 */
export type RollupSource = {
  readonly number: number;
  readonly observation: Observation;
  readonly sections: Readonly<Record<string, string>>;
};

/** モデルが切り出した関心 1 つ。 */
export type Concern = {
  readonly title: string;
  readonly body: string;
  /** この関心の根拠になった Feedback Issue の番号。 */
  readonly sources: readonly number[];
};

const SOURCES_LINE = /^sources:\s*(.*)$/;

/**
 * 畳む対象を選ぶ。
 *
 * @remarks
 * 開いているものだけを対象にします。一度畳んで閉じた Issue を次の週も拾うと、同じ関心の
 * 統合 Issue が毎週生まれます。「開いている」を唯一の条件にすることで、再実行しても結果が
 * 変わりません——余分な状態を持たずに冪等になります。
 *
 * 読解も改善提案も無い Issue は外します。関心へ分解する材料が無く、渡しても
 * 「該当なし」が返るだけで、モデルに読ませる分だけ費用が増えます。
 */
export function rollupTargets(sources: readonly RollupSource[]): RollupSource[] {
  return sources.filter((s) => {
    const improvement = s.sections[IMPROVEMENT_SECTION];
    return improvement !== undefined && improvement !== "" && !improvement.startsWith("該当なし");
  });
}

/**
 * 関心への分解をモデルへ求める問いを組み立てる。
 *
 * @remarks
 * 分解の軸を「窓」でも「ブランチ」でもなく関心に置くことを明示します。ブランチで切ると、
 * 同じ関心が別のブランチに現れたときに別々の課題として並び、レトロが同じ議論を 2 回します。
 */
export function buildConcernPrompt(targets: readonly RollupSource[]): string {
  const lines: string[] = [
    "以下は、AI 開発セッションごとに記録された Feedback Issue です。",
    "これらを**関心ごと**にまとめ直してください。",
    "",
    "まとめる軸は「どのブランチで起きたか」ではなく「何が問題か」です。同じ問題が別の",
    "ブランチに現れているなら 1 つにまとめ、1 つのブランチから別々の問題が出ているなら分けます。",
    "",
  ];
  for (const t of targets) {
    lines.push(`## Feedback #${t.number}`);
    if (t.observation.branch !== undefined) lines.push(`ブランチ: ${t.observation.branch}`);
    for (const name of BODY_SECTIONS) {
      const text = t.sections[name];
      if (text === undefined || text === "" || text.startsWith("該当なし")) continue;
      lines.push(`### ${name}`, text);
    }
    lines.push("");
  }

  lines.push(
    "## 出力",
    "",
    "関心ごとに次の形で並べてください。関心の数は決めうちせず、材料から素直に切れる数にします。",
    "",
    "## <関心を一行で言い表した見出し>",
    "sources: <根拠にした Feedback Issue の番号をカンマ区切りで>",
    "<その関心の説明と、取りうる対応。日本語で>",
    "",
    "書き方:",
    "- 見出しは対象（skill / rule / doc / ci / tool）が分かる言い方にする。",
    "- 材料から読み取れないことは書かない。1 件の Issue にしか根拠が無い関心も、それはそれで 1 つ。",
    "- 無理にまとめない。関係の無いものを 1 つにすると、レトロがどちらにも着手できなくなる。",
  );
  return lines.join("\n");
}

/**
 * モデルの出力を関心の一覧として読む。
 *
 * @remarks
 * `sources` を持たない見出しは捨てます。根拠の無い関心は、後から「なぜこれが挙がったか」を
 * 辿れず、大元を閉じる根拠にもなりません。既知の番号だけを残すのも同じ理由です。
 */
export function parseConcerns(output: string, known: readonly number[]): Concern[] {
  const valid = new Set(known);
  const concerns: Concern[] = [];
  let title: string | undefined;
  let sources: number[] = [];
  let buffer: string[] = [];

  const flush = () => {
    const body = buffer.join("\n").trim();
    const kept = sources.filter((n) => valid.has(n));
    if (title !== undefined && kept.length > 0) concerns.push({ title, body, sources: kept });
    title = undefined;
    sources = [];
    buffer = [];
  };

  for (const raw of output.split("\n")) {
    const line = raw.replace(/\s+$/, "");
    const heading = /^##(?!#)\s*(.+?)\s*$/.exec(line);
    if (heading !== null) {
      flush();
      title = heading[1] as string;
      continue;
    }
    const src = SOURCES_LINE.exec(line.trim());
    if (src !== null && title !== undefined) {
      sources = (src[1] as string)
        .split(",")
        .map((s) => Number(s.trim().replace(/^#/, "")))
        .filter((n) => Number.isInteger(n));
      continue;
    }
    if (title !== undefined) buffer.push(line);
  }
  flush();

  return concerns;
}

/**
 * 統合 Issue の本文を組み立てる。
 *
 * @remarks
 * 根拠を先に置きます。統合 Issue は「読解の読解」で、大元より 1 段抽象が上がっているぶん、
 * どの観測から来たのかを辿れないと検証できない主張になります。
 */
export function renderIntegrationBody(concern: Concern): string {
  return [
    "<!-- 週次の統合機構が作成しました。根拠は下の Feedback Issue です -->",
    "",
    concern.body,
    "",
    "## 根拠",
    "",
    ...concern.sources.map((n) => `- #${n}`),
  ].join("\n");
}

/**
 * 大元へ残すコメント。どこへ畳まれたかを、閉じられた側から辿れるようにする。
 *
 * @remarks
 * 畳み先は 1 つとは限りません。1 つの窓が別々の関心を同時に含むことは普通にあり、そのとき
 * 根拠として複数の関心から参照されます。1 つだけ書くと、残りの関心へ辿れなくなります。
 */
export function renderRollupComment(integrationIssues: readonly number[]): string {
  const refs = integrationIssues.map((n) => `#${n}`).join(" ");
  return `週次の統合で ${refs} へ畳みました。この Issue の観測はそこで扱います。`;
}

/**
 * どの大元が、どの関心へ畳まれたかを引けるようにする。
 *
 * @remarks
 * 関心ごとに閉じると、複数の関心が同じ大元を指したとき 2 回目以降が「既に閉じている」に
 * なります。実害は無いものの、コメントも 1 本ずつ増えて辿り先が散らばるので、大元を鍵に
 * 反転させてから 1 回だけ閉じます。
 */
export function rollupDestinations(
  created: readonly { readonly issue: number; readonly sources: readonly number[] }[],
): Map<number, number[]> {
  const bySource = new Map<number, number[]>();
  for (const c of created) {
    for (const n of c.sources) bySource.set(n, [...(bySource.get(n) ?? []), c.issue]);
  }
  return bySource;
}
