// マーカー行の分布を固定するための判定。
//
// 撤去マーカー（`boilerplate-only` / `sample-api`）は、本物として発火してほしい行と、規約を
// 説明するための例示とが、**同じ形**をしている。位置でも構文でも区別できない——`internal/
// controller/job/README.md` の ```go フェンス内マーカーは発火してほしい本物であり、
// `docs/tutorial/build-user-feature.md` の同じ形は例示である。
//
// 区別できるのは書き手の意図だけなので、除去側は「例示だ」という宣言（各撤去ツールの
// `MARKER_LITERAL_FILES`）を持つ。問題はその宣言を**忘れたとき**で、
//
// - 対応の取れないマーカーになる場合 → 除去が中断する（声が出る）
// - 対応が取れている場合           → その区域が黙って消える（声が出ない）
//
// 後者が本題である。均衡した `begin` 〜 `end` こそ、規約を説明するときに人が自然に書く形で、
// markdownlint は空フェンスを valid と見なすため、撤去後のツリーを検査しても何も鳴らない。
//
// 見張るのは「マーカー行が増えたこと」だけでよい。例示を書けば必ず増えるからである。行数は
// マーカーを足した / 消した瞬間にしか動かないので、区域内の散文を直しても差分は出ない。

/** 撤去マーカーの行。両名前空間・全接尾辞を 1 本で見る。 */
const MARKER_LINE =
  /(?:\/\/|#|<!--)\s*(?:boilerplate-only|sample-api):(?:begin|end|line|replace-begin|replace-with|replace-end)\b/;

/** 走査から外すディレクトリ名。依存の取得物と VCS の内部。 */
export const EXCLUDED_DIRECTORIES: ReadonlySet<string> = new Set([
  ".git",
  "node_modules",
  "vendor",
]);

/**
 * 走査から外す相対パス接頭辞。
 *
 * @remarks
 * 生成物は再生成で戻るので、固定しても差分が出るだけです。この判定自身のディレクトリを外すのは、
 * 宣言とテストがマーカーの形を入力として持つためです——外さないと、自分を数えて自分と食い違い
 * ます。各撤去ツールが `MARKER_LITERAL_FILES` や `SELF_DIR` で同じことをしています。
 */
export const EXCLUDED_PATH_PREFIXES: readonly string[] = [
  "docs/portal/guides/",
  "docs/coverage/",
  "docs/db-schema/",
  "docs/godoc/",
  "graphify-out/",
  "tmp/",
  "scripts/marker-baseline/",
];

/** ファイルごとのマーカー行数。値が 0 の項目は持たない（持つと「無い」の表現が 2 通りになる）。 */
export type Baseline = Readonly<Record<string, number>>;

/** 走査対象か。ディレクトリ名の除外は列挙側が行うため、ここは接頭辞だけを見る。 */
export function isBaselineTarget(relativePath: string): boolean {
  const normalized = relativePath.split("\\").join("/");

  return !EXCLUDED_PATH_PREFIXES.some((prefix) => normalized.startsWith(prefix));
}

/** 本文が含むマーカー行の数。 */
export function countMarkerLines(content: string): number {
  return content.split("\n").filter((line) => MARKER_LINE.test(line)).length;
}

/**
 * 実際とベースラインの食い違い。人が読んで判断できる文にする。
 *
 * @remarks
 * 増えた側だけでなく減った側も出します。マーカーが移動・削除されたのに宣言が古いままだと、
 * 次に増えたときの基準がずれ、検査は在るのに何も守っていない状態になります。
 */
export function diffBaseline(actual: Baseline, expected: Baseline): string[] {
  const failures: string[] = [];

  for (const [file, count] of Object.entries(actual)) {
    const before = expected[file];

    if (before === undefined) {
      failures.push(
        `マーカー行が現れました: ${file}（${count} 行）` +
          " — 本物のマーカーならベースラインへ、規約の例示なら撤去ツールの MARKER_LITERAL_FILES へ",
      );
      continue;
    }
    if (before !== count) {
      failures.push(`マーカー行数が変わりました: ${file}（${before} → ${count} 行）`);
    }
  }

  for (const file of Object.keys(expected)) {
    if (actual[file] === undefined) {
      failures.push(`マーカー行が無くなりました: ${file} — ベースラインのほうが古い`);
    }
  }

  return failures.sort();
}
