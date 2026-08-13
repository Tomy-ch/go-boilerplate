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

/** マーカー行に当たる正規表現を組み立てる。両名前空間を 1 本で見る。 */
function marker(suffix: string): RegExp {
  return new RegExp(`(?:\\/\\/|#|<!--)\\s*(?:boilerplate-only|sample-api):${suffix}\\b`);
}

const BLOCK_BEGIN = marker("begin");
const BLOCK_END = marker("end");
const LINE_MARKER = marker("line");
const REPLACE_BEGIN = marker("replace-begin");
const REPLACE_WITH = marker("replace-with");
const REPLACE_END = marker("replace-end");

/** 退避コメント。`//` / `#` / `<!-- -->` のいずれかで `=` に続けて書かれる。 */
const ESCROW = /^\s*(?:(?:\/\/|#)\s*=\s?(.*)|<!--\s*=\s?(.*?)\s*-->)$/;

/**
 * fork したあとに残る本文。マーカーで囲まれた記述を落とす。
 *
 * @remarks
 * マーカーの中は前提を書いてよい場所です。`boilerplate-only` は fork 時の除去で、`sample-api` は
 * サンプル撤去で消えるので、fork した先の文書には届きません。検査の前に落とさないと、正しく
 * 囲った記述まで違反として数えます。
 *
 * `replace` は前後で扱いが逆になります。`replace-begin`〜`replace-with` は上流版なので落とし、
 * `replace-with`〜`replace-end` の退避コメントは**アンコメントして残します**。ここを一括で
 * 落とすと、fork 先へ実際に残る文面が検査から消えます。そこに前提が書かれていても、この検査は
 * 「0 件」と報告します——この検査が塞ごうとしている無言の失敗と、同じ形の穴になります。
 *
 * マーカー除去のロジックを独自に持つ理由（`stripMarkers` を呼ばない理由）は
 * `scripts/setup/lib/markers.ts` 冒頭が持ちます。
 */
export function survivingText(content: string): string {
  const out: string[] = [];
  let depth = 0;
  let inUpstreamSide = false;
  let inEscrowSide = false;

  for (const line of content.split("\n")) {
    if (REPLACE_BEGIN.test(line)) {
      inUpstreamSide = true;
      continue;
    }
    if (REPLACE_WITH.test(line)) {
      inUpstreamSide = false;
      inEscrowSide = true;
      continue;
    }
    if (REPLACE_END.test(line)) {
      inUpstreamSide = false;
      inEscrowSide = false;
      continue;
    }
    if (inUpstreamSide) continue;

    if (inEscrowSide) {
      const matched = ESCROW.exec(line);

      // 退避コメントの形をしていない行は、アンコメントの規則が読めない。落とすと fork 先の
      // 本文を検査から消すことになるので、そのまま検査へ通す。
      out.push(matched === null ? line : (matched[1] ?? matched[2]));
      continue;
    }

    if (BLOCK_BEGIN.test(line)) {
      depth++;
      continue;
    }
    if (BLOCK_END.test(line)) {
      depth = Math.max(0, depth - 1);
      continue;
    }
    if (depth > 0 || LINE_MARKER.test(line)) continue;

    out.push(line);
  }

  return out.join("\n");
}

/** 相対パスの区切りを `/` に揃える。判定を OS に依らせないため。 */
function normalize(relativePath: string): string {
  return relativePath.split("\\").join("/");
}

/** 検査対象の文書か。許可域は対象外、禁止域と層 README が対象。 */
export function isChecked(relativePath: string): boolean {
  const rel = normalize(relativePath);

  if (!rel.endsWith(".md")) return false;
  if (ALLOWED_PREFIXES.some((prefix) => rel === prefix || rel.startsWith(prefix))) return false;

  if (rel.startsWith("docs/") && rel.endsWith(".ja.md")) {
    // CHECKED_PREFIXES はディレクトリだけでなく `docs/rules.md` のようなファイル名も持つため、
    // 対訳は正本の綴りへ戻してから領域を見る。許可域の再判定は要らない —— 対訳は正本の隣に
    // 居るので、許可域に入るなら上の検査が既に落としている。
    // 層 README の対訳は LAYER_README_RE が `.ja.md` ごと拾うので、最後の行に任せる。
    const canonical = rel.replace(/\.ja\.md$/, ".md");

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
