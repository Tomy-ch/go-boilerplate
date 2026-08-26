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
/**
 * 説明コメント付きの `.PHONY` 行。1 行に複数ターゲットを書いた場合は全件を一覧に出す。
 *
 * ターゲット側を `[^#]*` で一息に取って端の空白を呼び出し元で落とすのは、`[^#]` が空白も
 * 含むためで、区切りの空白を別に書くと同じ位置を両方が取り合って後戻りが起きる。
 */
const DOCUMENTED_PHONY = /^\.PHONY:([^#]*)##(.*)$/;
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
/** 見出し行なら、一覧へ出す行を返す。見出しでなければ `null`。 */
function categoryLines(line: string): string[] | null {
  const category = CATEGORY.exec(line);

  return category ? ["", `📂 ${category[1]}`] : null;
}

/** 説明コメント付きの `.PHONY` 行なら、ターゲット 1 件につき 1 行を返す。そうでなければ `null`。 */
function targetLines(line: string): string[] | null {
  const documented = DOCUMENTED_PHONY.exec(line);
  const targets = documented?.[1].trim();
  if (!documented || !targets) return null;

  const description = documented[2].trimStart();

  return targets
    .split(/[ \t]+/)
    .map((target) => `🛠  ${target.padEnd(TARGET_COLUMN_WIDTH)} ${description}`);
}

export function renderHelp(sources: readonly MakefileSource[]): HelpOutput {
  const lines: string[] = ["📦 Makeターゲット一覧", "-------------------------------------------"];
  const undocumented: string[] = [];

  for (const source of sources) {
    for (const line of source.content.split("\n")) {
      const rendered = categoryLines(line) ?? targetLines(line);
      if (rendered) {
        lines.push(...rendered);
        continue;
      }

      if (UNDOCUMENTED_PHONY.test(line)) {
        undocumented.push(`${source.path}: ${line.trim()}`);
      }
    }
  }

  return { lines, undocumented };
}

/**
 * ヘルプの材料として読むファイルかを判定する。
 *
 * @remarks
 * 拡張子だけで決めます。`.makefiles/` にはターゲット定義以外（README など）も置かれるため、
 * ディレクトリ配下を無条件に読むと、宣言でない行から偽のターゲットを拾います。
 */
export function isMakefileSource(fileName: string): boolean {
  return fileName.endsWith(".mk");
}
