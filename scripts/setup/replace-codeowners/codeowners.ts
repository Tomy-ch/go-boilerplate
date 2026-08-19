export const CODEOWNERS_FILE = ".github/CODEOWNERS";

const OWNER_TOKEN = String.raw`(?:@[^\s#]+|[^@\s#]+@[^\s#]+)`;

// ルール行は `<パターン><空白><所有者...>[ #コメント]`。空白は列揃えを保つため保持する。
// 所有者の形をしたトークンだけを所有者欄と認める。空白のみを境界にすると、空白を
// エスケープしたパターン(`foo\ bar.txt @o`)や複数語のセクション見出し(`[My Team]`)を
// パターンと所有者の境目で切って壊す。
const RULE_LINE = new RegExp(
  String.raw`^(\S+)([ \t]+)(${OWNER_TOKEN}(?:[ \t]+${OWNER_TOKEN})*)([ \t]*#.*)?[ \t]*$`,
);

// ヘッダーが記載例として挙げている所有者を保つため、コメント行は対象外とする。
const COMMENT_LINE = /^[ \t]*#/;

/**
 * 所有者を持たない行か。パターンだけの行は継承の打ち消しであり、書き換え対象ではない。
 *
 * 空行もここに含める。判定を正規表現 1 本ではなく trim と分割で書くのは、前後の空白と語が
 * 同じ位置を取り合って後戻りするためで、`[ \t]*\S*[ \t]*` の形はそれを避けられない。
 */
function isOwnerless(line: string): boolean {
  const trimmed = line.trim();

  return trimmed === "" || !/[ \t]/.test(trimmed);
}

export type CodeownersUpdate = {
  content: string;
  replaced: number;
  /**
   * 書き換え対象のはずが所有者を判定できなかった行番号（1 始まり）。
   *
   * 一括置換の取りこぼしが黙って残ると、レビューを予約したつもりで誰にも予約できて
   * いない状態になるため、呼び出し元から必ず表示する。
   */
  skippedLines: number[];
};

/** `--owners` の値を所有者の配列へ分解する。 */
export function parseOwners(raw: string): string[] {
  return raw
    .trim()
    .split(/\s+/)
    .filter(Boolean);
}

/**
 * CODEOWNERS の全ルール行の所有者欄を `owners` へ置き換える。
 *
 * @throws 所有者を持つルール行が 1 行も無い場合。
 */
export function replaceCodeowners(content: string, owners: string): CodeownersUpdate {
  const skippedLines: number[] = [];
  let replaced = 0;

  const updated = content
    .split("\n")
    .map((rawLine, index) => {
      // CRLF のファイルで書き換えた行だけ LF になり改行が混在するのを防ぐ。
      const eol = rawLine.endsWith("\r") ? "\r" : "";
      const line = eol === "" ? rawLine : rawLine.slice(0, -1);

      if (COMMENT_LINE.test(line) || isOwnerless(line)) {
        return rawLine;
      }

      const matched = RULE_LINE.exec(line);

      if (!matched) {
        skippedLines.push(index + 1);

        return rawLine;
      }

      replaced += 1;

      return `${matched[1]}${matched[2]}${owners}${matched[4] ?? ""}${eol}`;
    })
    .join("\n");

  if (replaced === 0) {
    throw new Error(`${CODEOWNERS_FILE} に所有者を持つルール行が見つかりませんでした。`);
  }

  return { content: updated, replaced, skippedLines };
}
