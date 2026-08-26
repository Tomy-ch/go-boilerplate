/**
 * 観測と読解候補から読解を求める問いを組み立て、返ってきた本文を節ごとに読み分ける。
 *
 * @remarks
 * 読解そのものはここでは行いません。ここが持つのは「何を尋ねるか」と「返ってきたものを
 * どう読み分けるか」だけで、モデルの呼び出しは入口が担います。
 *
 * 読解を CI ではなく手元で行うのは、CI へ材料を届ける手段が「逐語を public な Issue へ
 * 投稿して読み返させる」しかないためです（`docs/design/security.md`「Secrets」）。
 * 手元なら公開するのは読解結果だけで済みます。
 */

import { looksSecret, type Candidate } from "./candidates";
import { BODY_SECTIONS, parseSections, type Observation } from "./issue";
import { FINDING_KINDS, KIND_LABEL_PREFIX, type FindingKind } from "./score";

/**
 * 手元の読解に渡す候補の上限と 1 件あたりの長さ。
 *
 * @remarks
 * 公開時の既定（`DEFAULT_LIMIT`）と共用しないこと。公開側は露出量で、手元はプロンプト長で
 * しか縛られません（`docs/design/closed-loop.md`「Deterministic first, model second」）。
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
  const lines = output.split("\n");

  // 分類は最終行にだけ現れる約束なので、末尾の非空行 1 本しか見ない。行頭一致で全行を掃くと、
  // Evidence に引用した逐語がたまたま `kinds:` で始まっていた場合に、その行が本文から抜かれる。
  const lastIndex = lines.map((l) => l.trim()).findLastIndex((l) => l !== "");
  const lastLine = lastIndex < 0 ? "" : (lines[lastIndex] as string).trim();
  const kindLine = KINDS_LINE.exec(lastLine);

  const kindSet = new Set<string>(FINDING_KINDS);
  const kinds: FindingKind[] = [];
  if (kindLine !== null) {
    for (const k of (kindLine[1] as string).split(",").map((s) => s.trim())) {
      if (kindSet.has(k) && !kinds.includes(k as FindingKind)) kinds.push(k as FindingKind);
    }
    lines.splice(lastIndex, 1);
  }

  const { sections } = dropSecretSections(parseSections(lines.join("\n")));
  return Object.keys(sections).length === 0 ? undefined : { sections, kinds };
}

/**
 * 秘密らしき形を含む節を落とす。
 *
 * @remarks
 * 落とすのは節ごとで、迷ったら落とします。1 節欠けても他の節が残りますが、公開した 1 行は
 * 取り消せません。入口（`looksSecret` による候補の濾過）だけでなく出口もここで濾す理由は
 * `docs/design/security.md`「Secrets」にあります。
 */
export function dropSecretSections(sections: Readonly<Record<string, string>>): {
  sections: Record<string, string>;
  dropped: string[];
} {
  const kept: Record<string, string> = {};
  const dropped: string[] = [];
  for (const [name, text] of Object.entries(sections)) {
    if (looksSecret(text)) dropped.push(name);
    else kept[name] = text;
  }
  return { sections: kept, dropped };
}

/** 分類を Issue のラベル名にする。 */
export function kindLabels(kinds: readonly FindingKind[]): string[] {
  return kinds.map((k) => `${KIND_LABEL_PREFIX}${k}`);
}

/**
 * 読解が無い窓を、CI が助けられるかどうかで分ける。
 *
 * @remarks
 * 「読解が無い」には 2 つの理由があり、行き先が違います。**モデルが呼べなかった**のなら材料は
 * 手元にあるので、逐語を渡せば CI が読めます。**材料そのものが無い**（トランスクリプトを
 * 取得できなかった）なら、渡すものが無く CI にできることもありません。
 *
 * この 2 つを同じラベルに落とすと、CI は材料ゼロで起動し、何も書けないままラベルだけ外して
 * 終わります。実行の痕跡は残るのに窓は読まれないままで、後から見て「読んだが何も無かった」と
 * 区別が付きません。
 */
export type ReadingGap = "read" | "model-unavailable" | "material-unavailable";

/** 読解の結果と材料の有無から、窓の状態を決める。 */
export function readingGap(summary: Summary | undefined, hasMaterial: boolean): ReadingGap {
  if (summary !== undefined) return "read";
  return hasMaterial ? "model-unavailable" : "material-unavailable";
}

/**
 * 窓の状態から Issue に付けるラベルを決める。
 *
 * @remarks
 * `feedback/needs-summary` を付けるのは CI が助けられる窓だけです。常に付ければ読解済みの窓まで
 * 走り直し、材料の無い窓では空振りします。
 */
export function issueLabels(gap: ReadingGap, summary: Summary | undefined): string[] {
  if (gap === "read" && summary !== undefined) return ["feedback", ...kindLabels(summary.kinds)];
  if (gap === "model-unavailable") return ["feedback", NEEDS_SUMMARY_LABEL];
  return ["feedback"];
}

/**
 * 逐語の候補をコメントとして投稿すべきか。
 *
 * @remarks
 * 投稿は CI へ材料を渡す唯一の手段であり、それ以外の目的を持ちません。読解済みなら渡す相手が
 * おらず、材料が無いなら渡すものが無い。どちらも公開しません
 * （`docs/design/security.md`「Secrets」）。
 */
export function needsCandidateComment(gap: ReadingGap): boolean {
  return gap === "model-unavailable";
}
