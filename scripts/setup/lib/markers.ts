// マーカーで囲まれた行を取り除く共通機構。除去する側（サンプル削除・初期化ツールの撤去）は
// どちらも一度きりで自消滅するため、規則をどちらかの中へ置くと、先に消えた方と一緒に消える。
//
// マーカーはコメント（// / # / <!-- のいずれか）に書かれる前提。コメント記号を必須にして、
// 文字列リテラルやドキュメント本文中の同一トークンを誤って拾わないようにする。
// markdown（<!-- ... -->）コメント行も対象に含める。

/** マーカー除去の結果。`removed` は取り除いた行数（マーカー行そのものを含む）。 */
export type StripResult = {
  content: string;
  removed: number;
};

// 置換マーカーの走査状態。
const OUTSIDE = 0;
const ACTIVE = 1;
const SUBSTITUTE = 2;

// 差し替え行の退避コメント。先頭の空白（インデント）は保持し、コメント記号と `=` マーカー・
// 直後の空白1つだけ剥がす。
//
// `<!-- = ... -->` 形式を別の枝に分けているのは、閉じ記号を剥がす処理が行末に触れるため。
// `//`/`#` 側の行末はそのまま返す必要がある（Markdown の行末 2 スペースは hard line break で、
// 落とすと意味が変わる）。HTML コメント側は閉じ記号を必須にして、閉じ忘れを通さない。
//
// HTML コメント側の本文を貪欲に取り、行末の空白は呼び出し元で落とす。`.` は空白も含むため、
// 閉じ記号の手前を `\s*` で別に書くと同じ位置を両方が取り合って後戻りする。末尾の `$` が
// あるので、貪欲・非貪欲のどちらでも当たるのは最後の `-->` である。
const REPLACE_CONTENT = /^([ \t]*)(?:(?:\/\/|#)[ \t]*=[ \t]?(.*)|<!--[ \t]*=[ \t]?(.*)-->)$/;

/** 引用行。継ぎ目の両側が引用なら、空行では分断されてしまう。 */
const QUOTE_LINE = /^\s*>/;

/** `<comment> <marker>:<suffix>` に当たる正規表現を組み立てる。 */
function markerPattern(marker: string, suffix: string): RegExp {
  return new RegExp(String.raw`(?:\/\/|#|<!--)\s*${marker.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}:${suffix}\b`);
}

/**
 * `<marker>:begin`〜`<marker>:end` で囲まれた行と、行末に `<marker>:line` を持つ行を除去する。
 * さらに `<marker>:replace-begin`/`replace-with`/`replace-end` による置換にも対応する。
 *
 * @remarks
 * replace マーカーは `replace-begin`〜`replace-with` の有効行（対象が在るときに生きるコード）を除去し、
 * `replace-with`〜`replace-end` の差し替え行（`// =` / `# =` でコメント化された退避コード）を
 * アンコメントして残す。除去後にだけ有効化したい代替コードを、単純な行/ブロック除去では
 * 表現できない「置換」として扱うための仕組み。退避コメントは `//` 直後にスペースを置く（gocritic 準拠）。
 * Markdown 散文では `<!-- = ... -->` を使う。`# =` は 2 つ目の H1 として描画されて markdownlint の
 * MD025 に落ち、`// =` はその文字列が本文に出てしまうため、この形式でしか書けない。
 *
 * @throws 対応の取れないマーカー、または差し替え側に退避コメント以外の行がある場合。
 */
/** マーカーの各形に当たる正規表現一式。1 行につき何度も組み立て直さないためにまとめて持つ。 */
type MarkerPatterns = {
  blockBegin: RegExp;
  blockEnd: RegExp;
  lineMarker: RegExp;
  replaceBegin: RegExp;
  replaceWith: RegExp;
  replaceEnd: RegExp;
};

/**
 * 出力の積み上げ先。
 *
 * 継ぎ目の繕いに直前の行と「直前を消したか」の両方が要るため、行の配列だけでは足りない。
 */
type Sink = {
  out: string[];
  removed: number;
  cutJustBefore: boolean;
};

/** 1 行を消す。次に残す行が継ぎ目に来ることを記録する。 */
function cut(sink: Sink): void {
  sink.removed++;
  sink.cutJustBefore = true;
}

/**
 * 1 行を残す。消した跡の継ぎ目でだけ、空行の重なりと引用の分断を繕う。
 *
 * @remarks
 * ブロックの前後に空行を置くのは Markdown では普通の書き方なので、ブロックを抜くと
 * その 2 つの空行が隣り合います。繕うのを継ぎ目に限るのは、コードフェンス内の
 * 意図した連続空行を壊さないためです。
 *
 * 引用どうしの継ぎ目はさらに一手要ります。空行 1 つで隔てられた 2 つの引用は、Markdown では
 * 「途中に空行のある 1 つの引用」と読まれて壊れるため、空行を `>`（引用内の段落区切り）へ
 * 置き換えます。両側を独立した注記のまま残せる唯一の形です。
 */
function keep(sink: Sink, line: string): void {
  if (sink.cutJustBefore) {
    if (line.trim() === "" && (sink.out.at(-1) ?? "").trim() === "") {
      sink.removed++;

      return;
    }

    if (
      QUOTE_LINE.test(line) &&
      (sink.out.at(-1) ?? "").trim() === "" &&
      QUOTE_LINE.test(sink.out.at(-2) ?? "")
    ) {
      sink.out[sink.out.length - 1] = ">";
    }
  }

  sink.out.push(line);
  sink.cutJustBefore = false;
}

/**
 * replace マーカーの行なら、その次の行から属する側を返す。マーカーでなければ `null`。
 *
 * 対応の取れない並び（入れ子・片割れ）はここで落とします。読み進めてから気付くと、どこまでが
 * 差し替え対象だったのかを言えないまま出力が出来上がるためです。
 */
function replaceTransition(
  line: string,
  patterns: MarkerPatterns,
  state: number,
  marker: string,
): number | null {
  if (patterns.replaceBegin.test(line)) {
    if (state !== OUTSIDE) {
      throw new Error(`${marker}:replace ブロックは入れ子にできません。`);
    }

    return ACTIVE;
  }
  if (patterns.replaceWith.test(line)) {
    if (state !== ACTIVE) {
      throw new Error(`${marker}:replace-with に対応する ${marker}:replace-begin がありません。`);
    }

    return SUBSTITUTE;
  }
  if (patterns.replaceEnd.test(line)) {
    if (state === OUTSIDE) {
      throw new Error(`${marker}:replace-end に対応する ${marker}:replace-begin がありません。`);
    }

    return OUTSIDE;
  }

  return null;
}

/** ブロックマーカーの行なら、その次の行から数える深さを返す。マーカーでなければ `null`。 */
function blockTransition(
  line: string,
  patterns: MarkerPatterns,
  depth: number,
  marker: string,
): number | null {
  if (patterns.blockBegin.test(line)) return depth + 1;
  if (patterns.blockEnd.test(line)) {
    if (depth === 0) {
      throw new Error(`${marker}:end に対応する ${marker}:begin が見つかりません。`);
    }

    return depth - 1;
  }

  return null;
}

/** 差し替え側の 1 行をアンコメントする。退避コメントの形をしていない行は書き手の誤りとして落とす。 */
function uncommentSubstitute(line: string, marker: string): string {
  const matched = REPLACE_CONTENT.exec(line);

  if (matched === null) {
    throw new Error(
      `${marker}:replace-with 〜 replace-end の行は // = / # = / <!-- = --> のいずれかで書いてください: ${line}`,
    );
  }

  return matched[1] + (matched[2] ?? matched[3].trimEnd());
}

export function stripMarkers(content: string, marker: string): StripResult {
  const patterns: MarkerPatterns = {
    blockBegin: markerPattern(marker, "begin"),
    blockEnd: markerPattern(marker, "end"),
    lineMarker: markerPattern(marker, "line"),
    replaceBegin: markerPattern(marker, "replace-begin"),
    replaceWith: markerPattern(marker, "replace-with"),
    replaceEnd: markerPattern(marker, "replace-end"),
  };

  const sink: Sink = { out: [], removed: 0, cutJustBefore: false };
  let depth = 0;
  let replaceState: number = OUTSIDE;

  for (const line of content.split("\n")) {
    const nextReplaceState = replaceTransition(line, patterns, replaceState, marker);
    if (nextReplaceState !== null) {
      replaceState = nextReplaceState;
      cut(sink);
      continue;
    }
    // 有効側（対象が在るときのコード）は除去し、差し替え側は退避コメントを外して残す。
    if (replaceState === ACTIVE) {
      cut(sink);
      continue;
    }
    if (replaceState === SUBSTITUTE) {
      keep(sink, uncommentSubstitute(line, marker));
      continue;
    }

    const nextDepth = blockTransition(line, patterns, depth, marker);
    if (nextDepth !== null) {
      depth = nextDepth;
      cut(sink);
      continue;
    }
    if (depth > 0 || patterns.lineMarker.test(line)) {
      cut(sink);
      continue;
    }

    keep(sink, line);
  }

  if (depth > 0) {
    throw new Error(`${marker}:begin に対応する ${marker}:end が見つかりません。`);
  }
  if (replaceState !== OUTSIDE) {
    throw new Error(`${marker}:replace-begin に対応する ${marker}:replace-end が見つかりません。`);
  }

  return { content: sink.out.join("\n"), removed: sink.removed };
}
