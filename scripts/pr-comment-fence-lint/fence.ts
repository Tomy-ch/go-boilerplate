import { usesActionPattern } from "../lib/workflow";

export const COMMENT_ACTION = "./.github/actions/upsert-pr-comment";

const FIXED_FENCE = /(?:^|[\s'"])(`{3,})(?:[A-Za-z0-9_-]*)['"]?\s*$/;
const FENCE_FOR_BLOCK = /^\s*fence_for\(\)\s*\{$/;
const COMMENT_ACTION_USE = usesActionPattern(COMMENT_ACTION, false);
const DETAILS_SUMMARY = /^\s*details-summary:\s*(.*)$/;
const STEP_BULLET = /^\s*-\s/;
// バッククォート 1 個で開いて 1 個で閉じる span の中に、シェル変数展開か printf の変換指定がある形。
const INTERPOLATED_SPAN = /`[^`\n]*(?:\$\{|\$[A-Za-z_]|%[-0-9.*]*[sb])[^`\n]*`/;

/** 違反箇所の行番号（1 始まり）と、その行の内容。 */
export type Hit = {
  line: number;
  text: string;
};

/**
 * `fence_for` の実装本文を取り出す。
 *
 * @remarks
 * インデント差で不一致にならないよう、各行を trim して比較単位にします。実装を composite action へ
 * 集約できない（フェンス文字列を output で受け取るとシェルの二重引用符文脈でコマンド置換になる）ため
 * 複製が残っており、その複製が片方だけ直る事故を突き合わせで止めます。
 */
export function extractFenceFor(lines: readonly string[]): string | null {
  const start = lines.findIndex((line) => FENCE_FOR_BLOCK.test(line));
  if (start < 0) return null;

  const body: string[] = [];

  for (let i = start; i < lines.length; i++) {
    const trimmed = lines[i].trim();
    body.push(trimmed);
    if (i > start && trimmed === "}") return body.join("\n");
  }

  return null;
}

/**
 * 固定長のフェンスを出力している `echo` 行を探す。
 *
 * @remarks
 * 変数でフェンスを組む行（`echo "${fence}text"`）は対象外です。本文の最長バッククォート連から
 * 長さを決める実装はここで弾く対象ではありません。
 */
export function findFixedFences(lines: readonly string[]): Hit[] {
  const hits: Hit[] = [];

  lines.forEach((line, index) => {
    const trimmed = line.trim();

    if (!trimmed.startsWith("echo ")) return;
    if (trimmed.includes("${")) return;
    if (FIXED_FENCE.test(trimmed)) hits.push({ line: index + 1, text: trimmed });
  });

  return hits;
}

/**
 * その行がアクションに本文をフェンスさせるか。
 *
 * @remarks
 * アクションがフェンスするのは `details-summary` が空でない値を持つときだけ（action.yaml の
 * `detailsSummary ? ... : content`）です。キーの有無で判定すると `details-summary: ''` が
 * 「フェンス済み」に化けて検査が黙ります。式は静的に空か判定できないので、フェンスされない側へ倒します。
 */
export function fencesBody(line: string): boolean {
  const matched = DETAILS_SUMMARY.exec(line);
  if (matched === null) return false;

  const value = matched[1].trim();
  if (value === "" || value === "''" || value === '""') return false;

  return !value.includes("${{");
}

/**
 * `details-summary` を渡さない呼び出しがあるか。
 *
 * @remarks
 * ステップの範囲は `uses:` の深さではなく、そのステップを開いた `-` の桁で決めます。深さで見ると
 * `- uses:` から書き始めたステップが自分の `-` の桁を `uses:` の桁として測り、次のステップの `-`
 * で止まれません。素通しの呼び出しが後続ステップの `details-summary` を拾って隠れる向きに外れます。
 */
export function hasPassThroughCall(lines: readonly string[]): boolean {
  for (let i = 0; i < lines.length; i++) {
    if (!COMMENT_ACTION_USE.test(lines[i])) continue;

    let bullet = i;
    while (bullet >= 0 && !STEP_BULLET.test(lines[bullet])) bullet--;
    const stepIndent = bullet >= 0 ? lines[bullet].length - lines[bullet].trimStart().length : 0;

    let fenced = false;

    for (let j = i + 1; j < lines.length; j++) {
      const line = lines[j];
      if (line.trim() === "") continue;

      const indent = line.length - line.trimStart().length;
      // 次のステップを開く `-`（同じ桁）か、ステップより浅いキーに出たらこの呼び出しは終わり。
      if (indent < stepIndent || (indent === stepIndent && STEP_BULLET.test(line))) break;
      if (fencesBody(line)) {
        fenced = true;
        break;
      }
    }

    if (!fenced) return true;
  }

  return false;
}

/** inline code span の内側へ値を補間している行を探す。規約そのものを説明するコメント行は除く。 */
export function findInterpolatedSpans(lines: readonly string[]): Hit[] {
  const hits: Hit[] = [];

  lines.forEach((line, index) => {
    const trimmed = line.trim();

    if (trimmed.startsWith("#")) return;
    if (INTERPOLATED_SPAN.test(trimmed)) hits.push({ line: index + 1, text: trimmed });
  });

  return hits;
}

/**
 * 解決までフェンス検査から外すワークフロー。キーはファイル名、値は根拠の issue。
 *
 * @remarks
 * 空にできるのが正常な状態です。エントリを足すときは必ず根拠を書き、直したら消します。
 * 除外は実行時に必ず出力するので、除外されたファイルが検査済みとして通ることはありません。
 */
export const PASS_THROUGH_EXCLUSIONS: ReadonlyMap<string, string> = new Map<string, string>([]);

/** 1 ワークフロー分の走査結果。 */
export type FenceScan = {
  violations: string[];
  /** そのファイルが持つ `fence_for` の実装。持たなければ null。 */
  implementation: string | null;
};

/**
 * ワークフロー 1 本を走査し、フェンスと span の違反を返す。
 *
 * @remarks
 * span 検査は「本文素通しの呼び出しを持つファイル」だけに掛けます。`details-summary` 付きの
 * 呼び出しは action 側が本文からフェンス長を決めるため、span を組む責任がワークフローに無いからです。
 * 粒度はファイル単位で、ステップから本文ファイルへのデータフローは追いません。
 */
export function scanWorkflow(
  file: string,
  lines: readonly string[],
  exclusions: ReadonlyMap<string, string> = PASS_THROUGH_EXCLUSIONS,
): FenceScan {
  const violations: string[] = [];

  for (const hit of findFixedFences(lines)) {
    violations.push(
      `${file}:${hit.line}: 固定長のフェンスを出力しています。本文がこのフェンスを閉じられます: ${hit.text}`,
    );
  }

  if (hasPassThroughCall(lines) && !exclusions.has(basename(file))) {
    for (const hit of findInterpolatedSpans(lines)) {
      violations.push(
        `${file}:${hit.line}: 本文素通しの呼び出しがあるワークフローで、inline code span へ値を補間しています。値に含まれるバッククォート 1 個が span を閉じ、以降が生 Markdown になります: ${hit.text}`,
      );
    }
  }

  return { violations, implementation: extractFenceFor(lines) };
}

/**
 * 複製された `fence_for` 実装が互いに一致するかを検査する。
 *
 * @remarks
 * 実装を composite action へ集約できないため複製が残ります。フェンス文字列を output 経由で
 * 受け取るとバッククォートがシェルの二重引用符文脈でコマンド置換になるからです。複製は意図した
 * 選択で、そのぶん片方だけが直る事故をここで止めます。先頭を基準にするのは、ずれた側だけを
 * 挙げるためです。
 */
export function compareImplementations(
  implementations: ReadonlyArray<readonly [string, string]>,
): string[] {
  if (implementations.length < 2) return [];

  const [referenceFile, referenceImpl] = implementations[0];

  return implementations
    .slice(1)
    .filter(([, implementation]) => implementation !== referenceImpl)
    .map(
      ([file]) =>
        `${file}: fence_for の実装が ${referenceFile} と一致しません。フェンス長の計算は全複製で同一である必要があります`,
    );
}

/** パス末尾のファイル名。除外宣言はファイル名で書くため、走査側で切り出す。 */
function basename(file: string): string {
  const cut = file.lastIndexOf("/");

  return cut === -1 ? file : file.slice(cut + 1);
}
