import type { Schema } from "hast-util-sanitize";

/**
 * ブロックとして通すタグです。
 *
 * `h1` を含めます。面の title が持つのはファイル名由来の見出しであって文書の題ではないため、
 * 本文側の `h1` と競合しません。落とすとタグだけが外れて題のテキストが本文の冒頭へ浮きます。
 */
export const DOCUMENT_BLOCK_TAG_NAMES: readonly string[] = [
  "p",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "ul",
  "ol",
  "li",
  "blockquote",
  "hr",
  "pre",
  "table",
  "thead",
  "tbody",
  "tr",
  "th",
  "td",
  "img",
];

/** インラインとして通すタグです。 */
export const DOCUMENT_INLINE_TAG_NAMES: readonly string[] = [
  "strong",
  "em",
  "s",
  "del",
  "code",
  "a",
  "br",
  "sup",
  "sub",
];

export const DOCUMENT_TAG_NAMES: readonly string[] = [
  ...DOCUMENT_BLOCK_TAG_NAMES,
  ...DOCUMENT_INLINE_TAG_NAMES,
];

/** `a` の `href` と `img` の `src` に通すプロトコルです。 */
export const DOCUMENT_LINK_PROTOCOLS: readonly string[] = ["http", "https", "mailto"];

/**
 * `code` の class に通す値の形です。
 *
 * コードブロックの言語表記だけを通します。class 属性はそれ自体が任意の文字列を運べるため、
 * 通す形を限定しないと、スタイルを持つ class 名を本文から指定できてしまいます。
 */
export const DOCUMENT_CODE_CLASS_PATTERN = /^language-[\w+#.-]+$/;

/** 内容ごと取り除くタグです。 */
export const DOCUMENT_STRIPPED_TAG_NAMES: readonly string[] = ["script", "style"];

/**
 * ドキュメント表示用の sanitize schema です。
 *
 * アプリ本体の `model/rich-text` より広い範囲を通します。描画対象がリポジトリ自身の持つ
 * コミット済みドキュメントであり、表・コードブロック・図を落とすと用を成さないためです。
 * この広さを利用者の投稿内容へ適用してはならないので、schema はこのパッケージから外へ
 * 出しません。パッケージ境界がそのまま適用範囲の境界になります。
 *
 * `hast-util-sanitize` の既定 schema へ委ねる項目を残しません。未指定の項目は既定値で
 * 補完される仕様のため、上流の既定が広がったときに通過範囲が黙って広がることを防ぎます。
 */
export const DOCUMENT_SANITIZE_SCHEMA: Readonly<Schema> = {
  allowComments: false,
  allowDoctypes: false,
  // 祖先の制約。判定は変換前の木を辿るため、祖先自身が落ちた場合の子は救えない
  // （`<table>` 抜きの `<tr><td>` は `tr` だけが外れ、`td` が孤立して残る）。構造の乱れに
  // 留まりセキュリティの境界ではないため、後処理は持たない。
  ancestors: {
    li: ["ul", "ol"],
    thead: ["table"],
    tbody: ["table"],
    tr: ["table", "thead", "tbody"],
    th: ["tr"],
    td: ["tr"],
  },
  attributes: {
    a: ["href", "title"],
    img: ["src", "alt", "title"],
    code: [["className", DOCUMENT_CODE_CLASS_PATTERN]],
    ol: ["start"],
    th: ["colSpan", "rowSpan", "align"],
    td: ["colSpan", "rowSpan", "align"],
  },
  clobber: [],
  clobberPrefix: "",
  protocols: {
    href: [...DOCUMENT_LINK_PROTOCOLS],
    src: [...DOCUMENT_LINK_PROTOCOLS],
  },
  // alt を持たない図が本文へ出ると読み上げから内容が落ちる。空文字で補い、装飾として
  // 読み飛ばせる状態にする。
  required: { img: { alt: "" } },
  strip: [...DOCUMENT_STRIPPED_TAG_NAMES],
  tagNames: [...DOCUMENT_TAG_NAMES],
};
