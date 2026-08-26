// ワークフロー lint が見つけた違反の持ち方と、失敗出力の組み立て。
//
// 3 本の lint（pr-comment-secret / pr-comment-fence / actions-cutoff）が同じ形で報告するのは、
// 読む側が同じ CI ログだから。整形を各入口に複製すると、片方だけ書式が動いたときに、
// 出力を機械で拾っている側が黙って壊れる。

/** 違反 1 件。`line` は 1 始まりで、その違反を直すために開く行。 */
export type Finding = {
  file: string;
  line: number;
  message: string;
};

/**
 * 違反一覧をファイル単位にまとめた失敗出力へ整形する。
 *
 * @remarks
 * ファイルの並びは渡された順のままにします。走査は選別済みのファイル順に回るため、
 * ここで並べ替えると「どこまで進んだか」が出力から読めなくなります。
 */
export function formatFindings(findings: readonly Finding[]): string {
  const lines: string[] = [];
  let current: string | null = null;

  for (const finding of findings) {
    if (finding.file !== current) {
      if (current !== null) lines.push("");
      lines.push(`  ${finding.file}`);
      current = finding.file;
    }
    lines.push(`    :${finding.line}  ${finding.message}`);
  }

  return lines.join("\n");
}
