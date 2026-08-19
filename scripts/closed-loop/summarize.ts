/**
 * 観測と読解候補から読解を求める問いを組み立て、返ってきた本文を節ごとに読み分ける。
 *
 * @remarks
 * 読解そのものはここでは行いません。ここが持つのは「何を尋ねるか」と「返ってきたものを
 * どう読み分けるか」だけで、モデルの呼び出しは入口が担います。
 *
 * 読解を CI ではなく手元で行うのは、CI へ材料を届ける手段が「逐語を public な Issue へ
 * 投稿して読み返させる」しかないためです（`docs/design/security.md`「シークレット」）。
 * 手元なら公開するのは読解結果だけで済みます。
 */

import type { Candidate } from "./candidates";
import { BODY_SECTIONS, parseSections, type Observation } from "./issue";
import { FINDING_KINDS, KIND_LABEL_PREFIX, type FindingKind } from "./score";

/**
 * 手元の読解に渡す候補の上限と 1 件あたりの長さ。公開時の既定より大きく取ります。
 *
 * @remarks
 * 公開する候補の件数を絞るのは、逐語が 1 件外に出るごとに取り消せない露出が増えるからです。
 * 手元の読解にその制約は掛かりません。掛かるのはプロンプト長だけで、こちらのほうがずっと
 * 緩い。同じ数字を使い回すと、公開のために決めた上限が読解の材料まで削ることになります。
 */
export const LOCAL_CANDIDATE_LIMIT = 40;
export const LOCAL_EXCERPT_CHARS = 600;

/** 手元で読解できなかった窓に付けるラベル。CI 側の取り直しはこれだけを待ち受ける。 */
export const NEEDS_SUMMARY_LABEL = "feedback/needs-summary";

/** モデルが返した読解。節名は `BODY_SECTIONS` のもの。 */
export type Summary = {
  readonly sections: Readonly<Record<string, string>>;
  readonly kinds: readonly FindingKind[];
};

const KINDS_LINE = /^kinds:\s*(.*)$/;

/**
 * 読解を求める問いを組み立てる。
 *
 * @remarks
 * 「読み取れないことは書かない」を最初に置きます。埋めること自体が目的になると、根拠の無い
 * 所見が週次の数に入り、スコアが観測ではなく作文を順位づけ始めます。
 */
export function buildPrompt(observation: Observation, candidates: readonly Candidate[]): string {
  const lines: string[] = [
    "あなたは開発セッションの記録を読み、そこで何が摩擦になったかを言葉にする担当です。",
    "",
    "## 観測（決定論的に数えた値）",
    "",
    `窓 ID: ${observation.windowId}`,
    `クライアント: ${observation.client}`,
  ];
  if (observation.branch !== undefined) lines.push(`ブランチ: ${observation.branch}`);
  if (observation.pr !== undefined) lines.push(`PR: #${observation.pr}`);
  if (observation.prompts !== undefined) lines.push(`人の発話: ${observation.prompts}`);
  if (observation.toolCalls !== undefined) lines.push(`ツール呼び出し: ${observation.toolCalls}`);
  if (observation.toolFailures !== undefined) lines.push(`ツール失敗: ${observation.toolFailures}`);
  if (observation.interrupts !== undefined) lines.push(`中断: ${observation.interrupts}`);
  for (const p of observation.phases) lines.push(`フェーズ ${p.from} → ${p.to}: ${p.sec} 秒`);

  lines.push("", "## 読解候補（機械が選んだ逐語の抜粋）", "");
  if (candidates.length === 0) {
    lines.push("該当したターンはありません。");
  } else {
    for (const c of candidates) {
      lines.push(`- \`${c.reason}\``);
      lines.push(`  > ${c.text}`);
    }
  }

  lines.push(
    "",
    "## 出力",
    "",
    "次の見出しをこの順で出力し、各節を日本語で埋めてください。見出し以外の前置きや後書きは書かないこと。",
    "",
    ...BODY_SECTIONS.map((s) => `## ${s}`),
    "",
    "書き方:",
    "- 読み取れないことは書かない。候補に根拠が無い節は「該当なし」とだけ書く。",
    "  埋めるために推測すると、この Issue が後で集計されたとき根拠の無い所見が数に入る。",
    "- Evidence には根拠にした候補の逐語か、PR / コミットへの参照を置く。",
    "- Suggested Improvement は、対象（skill / rule / doc / ci / tool）が特定できる場合だけ書く。",
    "",
    "最後の行に、該当する分類をカンマ区切りで 1 行だけ出力してください。該当が無ければ空にすること。",
    `  kinds: ${FINDING_KINDS.join(" / ")} のうち該当するもの`,
  );

  return lines.join("\n");
}

/**
 * モデルの出力を節ごとに読み分ける。
 *
 * @remarks
 * 知らない見出しと `kinds` の未知の値は捨てます。ここが緩いと、モデルが見出しを言い換えた
 * だけで本文が丸ごと欠けたことに気づけません。既知の節が 1 つも取れなければ `undefined` を
 * 返し、呼び出し側は読解が無かったものとして扱います。
 */
export function parseSummary(output: string): Summary | undefined {
  const kindSet = new Set<string>(FINDING_KINDS);
  const kinds: FindingKind[] = [];

  for (const raw of output.split("\n")) {
    const kindLine = KINDS_LINE.exec(raw.trim());
    if (kindLine === null) continue;
    for (const k of (kindLine[1] as string).split(",").map((s) => s.trim())) {
      if (kindSet.has(k) && !kinds.includes(k as FindingKind)) kinds.push(k as FindingKind);
    }
  }

  // 分類の行は節の本文ではないので、節を読む前に落とす。残すと最後の節の末尾に混ざる。
  const sections = parseSections(
    output
      .split("\n")
      .filter((line) => KINDS_LINE.exec(line.trim()) === null)
      .join("\n"),
  );
  return Object.keys(sections).length === 0 ? undefined : { sections, kinds };
}

/** 分類を Issue のラベル名にする。 */
export function kindLabels(kinds: readonly FindingKind[]): string[] {
  return kinds.map((k) => `${KIND_LABEL_PREFIX}${k}`);
}

/**
 * 読解の有無から Issue に付けるラベルを決める。
 *
 * @remarks
 * 読解できなかった窓にだけ `feedback/needs-summary` を付けます。CI 側の取り直しはこの
 * ラベルだけを待ち受けるので、常に付けてしまうと読解済みの窓まで CI が走り直します。
 */
export function issueLabels(summary: Summary | undefined): string[] {
  return summary === undefined ? ["feedback", NEEDS_SUMMARY_LABEL] : ["feedback", ...kindLabels(summary.kinds)];
}

/**
 * 逐語の候補をコメントとして投稿すべきか。
 *
 * @remarks
 * 投稿は CI へ材料を渡す唯一の手段であり、それ以外の目的を持ちません。手元で読解できて
 * いれば渡す相手がいないので、公開しません（`docs/design/security.md`「シークレット」）。
 */
export function needsCandidateComment(summary: Summary | undefined): boolean {
  return summary === undefined;
}
