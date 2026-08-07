// サンプルAPI削除のマーカー除去ロジック。削除対象の宣言（SAMPLE_DOMAINS / MARKER_FILES / BUILD_STEPS）は
// sample-manifest.ts を参照。

// マーカーはコメント（// / # / <!-- のいずれか）に書かれる前提。コメント記号を必須にして、
// 文字列リテラルやドキュメント本文中の同一トークンを誤って拾わないようにする。
// markdown（<!-- ... -->）コメント行も対象に含める。
const BLOCK_BEGIN = /(?:\/\/|#|<!--)\s*sample-api:begin\b/;
const BLOCK_END = /(?:\/\/|#|<!--)\s*sample-api:end\b/;
const LINE_MARKER = /(?:\/\/|#|<!--)\s*sample-api:line\b/;

// replace マーカー: `replace-begin`〜`replace-with` の有効行（サンプル在時に生きるコード）を除去し、
// `replace-with`〜`replace-end` の差し替え行（`// =` / `# =` でコメント化された退避コード）をアンコメントして残す。
// 削除後にだけ有効化したい代替コード（例: 削除後のみ有効な既定値やテストケース）を、単純な行/ブロック除去では
// 表現できない「置換」として扱うための仕組み。退避コメントは `//` 直後にスペースを置く（gocritic 準拠）。
const REPLACE_BEGIN = /(?:\/\/|#|<!--)\s*sample-api:replace-begin\b/;
const REPLACE_WITH = /(?:\/\/|#|<!--)\s*sample-api:replace-with\b/;
const REPLACE_END = /(?:\/\/|#|<!--)\s*sample-api:replace-end\b/;
// 差し替え行の退避コメント。先頭の空白（インデント）は保持し、`//`/`#` と `=` マーカー・直後の空白1つだけ剥がす。
const REPLACE_CONTENT = /^(\s*)(?:\/\/|#)\s*=\s?(.*)$/;

/** マーカー除去の結果。`removed` は取り除いた行数（マーカー行そのものを含む）。 */
export type StripResult = {
  content: string;
  removed: number;
};

// 置換マーカーの走査状態。
const OUTSIDE = 0;
const ACTIVE = 1;
const SUBSTITUTE = 2;

/**
 * `sample-api:begin`〜`sample-api:end` で囲まれた行と、行末に `sample-api:line` を持つ行を除去する。
 * さらに `sample-api:replace-begin`/`replace-with`/`replace-end` による置換にも対応する。
 * ネストにも対応するため depth カウンターで管理する。
 *
 * @throws 対応の取れないマーカー、または差し替え側に退避コメント以外の行がある場合。
 */
export function stripSampleMarkers(content: string): StripResult {
  const lines = content.split("\n");
  const out: string[] = [];
  let depth = 0;
  let removed = 0;
  let replaceState: number = OUTSIDE;

  for (const line of lines) {
    if (REPLACE_BEGIN.test(line)) {
      if (replaceState !== OUTSIDE) {
        throw new Error("sample-api:replace ブロックは入れ子にできません。");
      }
      replaceState = ACTIVE;
      removed++;
      continue;
    }
    if (REPLACE_WITH.test(line)) {
      if (replaceState !== ACTIVE) {
        throw new Error("sample-api:replace-with に対応する sample-api:replace-begin がありません。");
      }
      replaceState = SUBSTITUTE;
      removed++;
      continue;
    }
    if (REPLACE_END.test(line)) {
      if (replaceState === OUTSIDE) {
        throw new Error("sample-api:replace-end に対応する sample-api:replace-begin がありません。");
      }
      replaceState = OUTSIDE;
      removed++;
      continue;
    }
    if (replaceState === ACTIVE) {
      // 有効側（サンプル在時のコード）は除去する。
      removed++;
      continue;
    }
    if (replaceState === SUBSTITUTE) {
      // 差し替え側は退避コメントをアンコメントして残す。
      const matched = REPLACE_CONTENT.exec(line);
      if (matched === null) {
        throw new Error(
          `sample-api:replace-with 〜 replace-end の行は //= または #= で始めてください: ${line}`,
        );
      }
      out.push(matched[1] + matched[2]);
      continue;
    }

    if (BLOCK_BEGIN.test(line)) {
      depth++;
      removed++;
      continue;
    }
    if (BLOCK_END.test(line)) {
      if (depth === 0) {
        throw new Error("sample-api:end に対応する sample-api:begin が見つかりません。");
      }
      depth--;
      removed++;
      continue;
    }
    if (depth > 0 || LINE_MARKER.test(line)) {
      removed++;
      continue;
    }
    out.push(line);
  }

  if (depth > 0) {
    throw new Error("sample-api:begin に対応する sample-api:end が見つかりません。");
  }
  if (replaceState !== OUTSIDE) {
    throw new Error("sample-api:replace-begin に対応する sample-api:replace-end が見つかりません。");
  }

  return { content: out.join("\n"), removed };
}

/**
 * 削除対象が ROOT_DIR の内側に収まっているかを判定する。
 *
 * @remarks
 * manifest への追記ミス（`..` を含むパス・空文字・絶対パス）で ROOT_DIR の外や ROOT_DIR 自体を
 * 消してしまわないための安全策です。dry-run でも必ず通し、削除の前に検証します。
 * 区切り文字を足してから前方一致を見るのは、`/repo` に対する `/repo-backup` のような
 * 「接頭辞は一致するが別のディレクトリ」を内側と誤判定しないためです。
 */
export function isWithinRoot(absolutePath: string, rootDir: string, separator: string): boolean {
  const rootWithSeparator = rootDir.endsWith(separator) ? rootDir : rootDir + separator;

  return absolutePath !== rootDir && absolutePath.startsWith(rootWithSeparator);
}
