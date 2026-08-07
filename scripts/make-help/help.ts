/** 1 つの `.mk` ファイルの内容。読み出しは呼び出し元の責務にする。 */
export type MakefileSource = {
  path: string;
  content: string;
};

export type HelpOutput = {
  /** 標準出力へ出す一覧本文。 */
  lines: string[];
  /** 説明コメントを持たないため一覧に出せなかった `.PHONY` 行（`<path>: <line>` 形式）。 */
  undocumented: string[];
};

/** カテゴリ見出し（`## ...`）。 */
const CATEGORY = /^## (.*)/;
/** 説明コメント付きの `.PHONY` 行。1 行に複数ターゲットを書いた場合は全件を一覧に出す。 */
const DOCUMENTED_PHONY = /^\.PHONY:\s+([^#]+?)\s*##\s*(.*)$/;
/** 説明コメントを持たない `.PHONY` 行。 */
const UNDOCUMENTED_PHONY = /^\.PHONY:(?!.*##)/;

const TARGET_COLUMN_WIDTH = 24;

/**
 * `.mk` の内容から `make help` の一覧本文を組み立てる。
 *
 * @remarks
 * 説明コメントを持たない `.PHONY` 行は一覧に出せません。黙って落とすと利用者から見えない
 * ターゲットになるため、本文とは別に返して呼び出し元が警告できるようにしています。
 */
export function renderHelp(sources: readonly MakefileSource[]): HelpOutput {
  const lines: string[] = ["📦 Makeターゲット一覧", "-------------------------------------------"];
  const undocumented: string[] = [];

  for (const source of sources) {
    for (const line of source.content.split("\n")) {
      const category = CATEGORY.exec(line);
      if (category) {
        lines.push("", `📂 ${category[1]}`);
        continue;
      }

      const documented = DOCUMENTED_PHONY.exec(line);
      if (documented) {
        for (const target of documented[1].split(/\s+/)) {
          lines.push(`🛠  ${target.padEnd(TARGET_COLUMN_WIDTH)} ${documented[2]}`);
        }
        continue;
      }

      if (UNDOCUMENTED_PHONY.test(line)) {
        undocumented.push(`${source.path}: ${line.trim()}`);
      }
    }
  }

  return { lines, undocumented };
}
