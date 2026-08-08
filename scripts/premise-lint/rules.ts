// 「その文書より先に失効する前提」が、失効後も残る文書に書かれていないかの判定。
//
// 規則の出所は docs/rules.md の Documentation Rules「No premise the document will outlive」。
// 前提を書いてよいのは fork 時に破棄・書き換えされる文書（README / docs/get-started/**）だけで、
// それ以外へ書くと、前提が偽になった瞬間に文書が自分を偽る。

/**
 * 前提を書いてよい領域。ここから外れた文書だけを検査する。
 *
 * @remarks
 * `README.md` はセットアップ手順が全面書き換えを指示し、`docs/get-started/**` はセットアップの
 * 完了で用済みになります。どちらも前提と一緒に消えるので、前提が残っても嘘になりません。
 */
export const ALLOWED_PREFIXES: readonly string[] = [
  "README.md",
  "README.ja.md",
  "docs/get-started/",
];

/** 検査対象の領域。ここに挙げた場所の文書は fork 後も残る。 */
export const CHECKED_PREFIXES: readonly string[] = [
  "docs/adr/",
  "docs/design/",
  "docs/spec/",
  "docs/project/",
  "docs/maintenance/",
  "docs/rules.md",
  "docs/architecture.md",
  "docs/decisions.md",
  "docs/index.md",
  ".makefiles/README.md",
  ".makefiles/README.ja.md",
];

/** 層 README も対象。`internal/**` / `pkg/**` の README は fork 後も読まれ続ける。 */
const LAYER_README_RE = /^(?:internal|pkg)\/.*README(?:\.ja)?\.md$/;

/**
 * 前提の焼き込みが取る言い回し。
 *
 * @remarks
 * 裸の名詞（`boilerplate` / `template` / `scaffold` / `downstream`）では検査になりません。
 * 実測で 381 件が当たり、その大半は別語義でした——`scaffold-*` スキル名、outbox の下流
 * コンシューマ、定型コードの意、Go の `text/template`。許可リストで潰すと、リストのほうが
 * 検査より大きくなって信号が死にます。
 *
 * 焼き込みが実際に取る形は **自己参照** です。「この文書が載っているリポジトリは、テンプレート
 * である」と述べる構文だけが、fork した瞬間に偽になります。そこに絞ると 28 件まで落ち、
 * うち 27 件は本物の焼き込みでした。
 */
export const PREMISE_PATTERNS: readonly RegExp[] = [
  /\b(?:this|the|our)\s+(?:template|boilerplate|scaffold)\b/i,
  /\badopters?\b/i,
  /\btemplate-derived\b/i,
  /\bdownstream\s+users?\b/i,
  /この(?:テンプレート|ボイラープレート|雛形)/,
  /本(?:テンプレート|ボイラープレート)/,
];

/** 相対パスの区切りを `/` に揃える。判定を OS に依らせないため。 */
function normalize(relativePath: string): string {
  return relativePath.split("\\").join("/");
}

/** 検査対象の文書か。許可域は対象外、禁止域と層 README が対象。 */
export function isChecked(relativePath: string): boolean {
  const rel = normalize(relativePath);

  if (!rel.endsWith(".md")) return false;
  if (ALLOWED_PREFIXES.some((prefix) => rel === prefix || rel.startsWith(prefix))) return false;

  if (rel.startsWith("docs/ja/")) {
    // ミラーは `docs/ja/` 配下に居て `.ja.md` で終わる。正本の綴りへ戻してから領域を見る。
    const canonical = `docs/${rel.slice("docs/ja/".length)}`.replace(/\.ja\.md$/, ".md");

    if (ALLOWED_PREFIXES.some((prefix) => canonical.startsWith(prefix))) return false;

    return CHECKED_PREFIXES.some((prefix) => canonical.startsWith(prefix));
  }

  return CHECKED_PREFIXES.some((prefix) => rel.startsWith(prefix)) || LAYER_README_RE.test(rel);
}

/** 1 行が前提の言い回しを含むか。 */
export function hasPremisePhrase(line: string): boolean {
  return PREMISE_PATTERNS.some((pattern) => pattern.test(line));
}

export type Allowance = {
  /** 対象ファイルの相対パス。 */
  file: string;
  /**
   * 許容する行の、前提の言い回しに当たっている部分。
   *
   * @remarks
   * 行番号でなく本文で綴じるのは、行が動いても壊れないためです。当たっている部分そのものを
   * 綴じるのは、宣言だけを読んで「何を許したのか」が分かるようにするためです。同じ行の
   * 無関係な部分を綴じると、後から別の前提が同じ行へ紛れ込んでも黙って通ります。
   */
  contains: string;
  /** なぜ前提ではないのか。 */
  reason: string;
};

export type Finding = {
  file: string;
  line: number;
  text: string;
};

/**
 * 1 ファイル分の判定。
 *
 * @remarks
 * `strip` には呼び出し側がマーカー除去後の本文を渡します。マーカーで囲まれた記述は fork へ
 * 届かないので、前提を書いてよい場所です——それを検査すると、正しく囲った記述まで落ちます。
 */
export function inspect(
  relativePath: string,
  survivingContent: string,
  allowances: readonly Allowance[],
): Finding[] {
  const rel = normalize(relativePath);
  const allowed = allowances.filter((entry) => normalize(entry.file) === rel);

  return survivingContent
    .split("\n")
    .map((text, index) => ({ file: rel, line: index + 1, text }))
    .filter(({ text }) => hasPremisePhrase(text))
    .filter(({ text }) => !allowed.some((entry) => text.includes(entry.contains)));
}
