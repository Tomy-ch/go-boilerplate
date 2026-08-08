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
const REPLACE_CONTENT = /^(\s*)(?:(?:\/\/|#)\s*=\s?(.*)|<!--\s*=\s?(.*?)\s*-->)$/;

/** 引用行。継ぎ目の両側が引用なら、空行では分断されてしまう。 */
const QUOTE_LINE = /^\s*>/;

/** `<comment> <marker>:<suffix>` に当たる正規表現を組み立てる。 */
function markerPattern(marker: string, suffix: string): RegExp {
  return new RegExp(`(?:\\/\\/|#|<!--)\\s*${marker.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}:${suffix}\\b`);
}

/**
 * `<marker>:begin`〜`<marker>:end` で囲まれた行と、行末に `<marker>:line` を持つ行を除去する。
 * さらに `<marker>:replace-begin`/`replace-with`/`replace-end` による置換にも対応する。
 * ネストにも対応するため depth カウンターで管理する。
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
export function stripMarkers(content: string, marker: string): StripResult {
  const blockBegin = markerPattern(marker, "begin");
  const blockEnd = markerPattern(marker, "end");
  const lineMarker = markerPattern(marker, "line");
  const replaceBegin = markerPattern(marker, "replace-begin");
  const replaceWith = markerPattern(marker, "replace-with");
  const replaceEnd = markerPattern(marker, "replace-end");

  const lines = content.split("\n");
  const out: string[] = [];
  let depth = 0;
  let removed = 0;
  let replaceState: number = OUTSIDE;
  // 直前の行を消したか。消した跡で空行が隣り合うのを畳むために要る。
  let cutJustBefore = false;

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
  const keep = (line: string): void => {
    if (cutJustBefore) {
      if (line.trim() === "" && (out.at(-1) ?? "").trim() === "") {
        removed++;

        return;
      }

      if (
        QUOTE_LINE.test(line) &&
        (out.at(-1) ?? "").trim() === "" &&
        QUOTE_LINE.test(out.at(-2) ?? "")
      ) {
        out[out.length - 1] = ">";
      }
    }

    out.push(line);
    cutJustBefore = false;
  };

  for (const line of lines) {
    if (replaceBegin.test(line)) {
      if (replaceState !== OUTSIDE) {
        throw new Error(`${marker}:replace ブロックは入れ子にできません。`);
      }
      replaceState = ACTIVE;
      removed++;
      cutJustBefore = true;
      continue;
    }
    if (replaceWith.test(line)) {
      if (replaceState !== ACTIVE) {
        throw new Error(`${marker}:replace-with に対応する ${marker}:replace-begin がありません。`);
      }
      replaceState = SUBSTITUTE;
      removed++;
      cutJustBefore = true;
      continue;
    }
    if (replaceEnd.test(line)) {
      if (replaceState === OUTSIDE) {
        throw new Error(`${marker}:replace-end に対応する ${marker}:replace-begin がありません。`);
      }
      replaceState = OUTSIDE;
      removed++;
      cutJustBefore = true;
      continue;
    }
    if (replaceState === ACTIVE) {
      // 有効側（対象が在るときのコード）は除去する。
      removed++;
      cutJustBefore = true;
      continue;
    }
    if (replaceState === SUBSTITUTE) {
      // 差し替え側は退避コメントをアンコメントして残す。
      const matched = REPLACE_CONTENT.exec(line);
      if (matched === null) {
        throw new Error(
          `${marker}:replace-with 〜 replace-end の行は // = / # = / <!-- = --> のいずれかで書いてください: ${line}`,
        );
      }
      keep(matched[1] + (matched[2] ?? matched[3]));
      continue;
    }

    if (blockBegin.test(line)) {
      depth++;
      removed++;
      cutJustBefore = true;
      continue;
    }
    if (blockEnd.test(line)) {
      if (depth === 0) {
        throw new Error(`${marker}:end に対応する ${marker}:begin が見つかりません。`);
      }
      depth--;
      removed++;
      cutJustBefore = true;
      continue;
    }
    if (depth > 0 || lineMarker.test(line)) {
      removed++;
      cutJustBefore = true;
      continue;
    }
    keep(line);
  }

  if (depth > 0) {
    throw new Error(`${marker}:begin に対応する ${marker}:end が見つかりません。`);
  }
  if (replaceState !== OUTSIDE) {
    throw new Error(`${marker}:replace-begin に対応する ${marker}:replace-end が見つかりません。`);
  }

  return { content: out.join("\n"), removed };
}
