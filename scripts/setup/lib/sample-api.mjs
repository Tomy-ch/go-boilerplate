// サンプルAPI削除のマーカー除去ロジック。削除対象の宣言（SAMPLE_DOMAINS / MARKER_FILES / BUILD_STEPS）は
// sample-manifest.mjs を参照。

// マーカーはコメント（// / # / <!-- のいずれか）に書かれる前提。コメント記号を必須にして、
// 文字列リテラルやドキュメント本文中の同一トークンを誤って拾わないようにする。
// markdown（<!-- ... -->）コメント行も対象に含める。
const BLOCK_BEGIN = /(?:\/\/|#|<!--)\s*sample-api:begin\b/
const BLOCK_END = /(?:\/\/|#|<!--)\s*sample-api:end\b/
const LINE_MARKER = /(?:\/\/|#|<!--)\s*sample-api:line\b/

// replace マーカー: `replace-begin`〜`replace-with` の有効行（サンプル在時に生きるコード）を除去し、
// `replace-with`〜`replace-end` の差し替え行（`// =` / `# =` でコメント化された退避コード）をアンコメントして残す。
// 削除後にだけ有効化したい代替コード（例: 削除後のみ有効な既定値やテストケース）を、単純な行/ブロック除去では
// 表現できない「置換」として扱うための仕組み。退避コメントは `//` 直後にスペースを置く（gocritic 準拠）。
const REPLACE_BEGIN = /(?:\/\/|#|<!--)\s*sample-api:replace-begin\b/
const REPLACE_WITH = /(?:\/\/|#|<!--)\s*sample-api:replace-with\b/
const REPLACE_END = /(?:\/\/|#|<!--)\s*sample-api:replace-end\b/
// 差し替え行の退避コメント。先頭の空白（インデント）は保持し、`//`/`#` と `=` マーカー・直後の空白1つだけ剥がす。
const REPLACE_CONTENT = /^(\s*)(?:\/\/|#)\s*=\s?(.*)$/

// `sample-api:begin`〜`sample-api:end` で囲まれた行と、行末に `sample-api:line` を持つ行を除去する。
// さらに `sample-api:replace-begin`/`replace-with`/`replace-end` による置換にも対応する。
// ネストにも対応するため depth カウンターで管理し、対応の取れないマーカーは throw する。
export function stripSampleMarkers(content) {
  const lines = content.split("\n")
  const out = []
  let depth = 0
  let removed = 0
  // 0: 置換外 / 1: 有効側（除去中） / 2: 差し替え側（アンコメント中）
  let replaceState = 0

  for (const line of lines) {
    if (REPLACE_BEGIN.test(line)) {
      if (replaceState !== 0) {
        throw new Error("sample-api:replace ブロックは入れ子にできません。")
      }
      replaceState = 1
      removed++
      continue
    }
    if (REPLACE_WITH.test(line)) {
      if (replaceState !== 1) {
        throw new Error("sample-api:replace-with に対応する sample-api:replace-begin がありません。")
      }
      replaceState = 2
      removed++
      continue
    }
    if (REPLACE_END.test(line)) {
      if (replaceState === 0) {
        throw new Error("sample-api:replace-end に対応する sample-api:replace-begin がありません。")
      }
      replaceState = 0
      removed++
      continue
    }
    if (replaceState === 1) {
      // 有効側（サンプル在時のコード）は除去する。
      removed++
      continue
    }
    if (replaceState === 2) {
      // 差し替え側は退避コメントをアンコメントして残す。
      const matched = REPLACE_CONTENT.exec(line)
      if (matched === null) {
        throw new Error(`sample-api:replace-with 〜 replace-end の行は //= または #= で始めてください: ${line}`)
      }
      out.push(matched[1] + matched[2])
      continue
    }

    if (BLOCK_BEGIN.test(line)) {
      depth++
      removed++
      continue
    }
    if (BLOCK_END.test(line)) {
      if (depth === 0) {
        throw new Error("sample-api:end に対応する sample-api:begin が見つかりません。")
      }
      depth--
      removed++
      continue
    }
    if (depth > 0 || LINE_MARKER.test(line)) {
      removed++
      continue
    }
    out.push(line)
  }

  if (depth > 0) {
    throw new Error("sample-api:begin に対応する sample-api:end が見つかりません。")
  }
  if (replaceState !== 0) {
    throw new Error("sample-api:replace-begin に対応する sample-api:replace-end が見つかりません。")
  }

  return { content: out.join("\n"), removed }
}
