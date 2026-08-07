export const LICENSE_FILE = "LICENSE";

// 行頭固定。本文中で著作権表示に言及する散文（`... Copyright (c) ...`）には当てない。
const COPYRIGHT_LINE = /^Copyright \(c\) .*/m;

/**
 * LICENSE の著作権表示行を `Copyright (c) <year> <holder>` へ書き換える。
 *
 * 権利者名に `$&` などが含まれても置換パターンとして解釈されないよう、関数で置換する。
 *
 * @throws 著作権表示行が見つからない場合。
 */
export function replaceCopyright(content: string, holder: string, year: string): string {
  if (!COPYRIGHT_LINE.test(content)) {
    throw new Error("LICENSE に著作権表示が見つかりませんでした。");
  }

  return content.replace(COPYRIGHT_LINE, () => `Copyright (c) ${year} ${holder}`);
}
