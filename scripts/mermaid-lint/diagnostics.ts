// 検証結果と例外を、人間が読む出力へ変える。
//
// mermaid の parse 失敗は「図の文法が壊れている」で、依存のロード失敗は「環境が整っていない」。
// 前者は直す先が Markdown、後者は直す先がコンテナなので、出力の時点で取り違えないよう分ける。

/** 壊れたブロック 1 件。`startLine` はフェンス開始行、`index` はファイル内での通し番号。 */
export type Failure = {
  rel: string;
  startLine: number;
  index: number;
  msg: string;
};

/**
 * 例外から表示用のメッセージを取り出す。
 *
 * @remarks
 * `Error` でない値を投げるライブラリがあるため、`message` の有無ではなく値そのものへ落とします。
 * 空メッセージの `Error` も文字列化側へ回すことで、空行だけの出力になるのを避けます。
 */
export function errorMessage(e: unknown): string {
  return (e instanceof Error && e.message ? e.message : String(e)).trim();
}

/**
 * 依存を解決できなかった（＝環境未整備の）例外かを判定する。
 *
 * @remarks
 * `code` と本文の両方を見ます。`import()` は `ERR_MODULE_NOT_FOUND` を付けますが、`require` 経由の
 * 失敗はコードを持たないことがあり、片方だけでは取りこぼします。
 */
export function isDependencyMissing(e: unknown): boolean {
  const code = (e as NodeJS.ErrnoException | undefined)?.code;

  return code === "ERR_MODULE_NOT_FOUND" || /cannot find (package|module)/i.test(errorMessage(e));
}

/**
 * 壊れたブロック一覧を失敗出力へ整形する。
 *
 * @remarks
 * mermaid のメッセージは複数行なので、各行を字下げして図の位置行にぶら下げます。字下げしないと
 * 次のブロックの見出しと同じ桁に並び、どのブロックの説明か読めなくなります。
 */
export function formatFailures(failures: readonly Failure[]): string {
  return failures
    .map((f) => {
      const detail = f.msg.split("\n").map((line) => `    ${line}`);

      return [`  ${f.rel}:${f.startLine}  (block #${f.index})`, ...detail, ""].join("\n");
    })
    .join("\n");
}

/** 検証件数の要約。読めず skip したファイルがあれば、黙って落とさず件数を添える。 */
export function summarize(blockCount: number, fileWithBlocks: number, skipped: number): string {
  const suffix = skipped > 0 ? `（読めず skip: ${skipped} 件）` : "";

  return `${blockCount} ブロック / ${fileWithBlocks} ファイル${suffix}`;
}
