import type { Element, ElementContent } from "hast";

/** コードフェンス 1 個分の中身。 */
export type CodeFence = {
  /** ` ```go ` の `go`。言語表記が無いフェンスでは `null`。 */
  language: string | null;
  /** フェンスの中身そのもの。 */
  code: string;
};

/** `language-<name>` から `<name>` を取り出す。 */
const LANGUAGE_CLASS_PATTERN = /^language-(.+)$/;

function languageOf(node: Element): string | null {
  // hast は空白区切りの属性を配列で持つ。文字列で来る木は無い。
  const names = node.properties.className ?? [];

  for (const name of names) {
    const matched = LANGUAGE_CLASS_PATTERN.exec(String(name));

    if (matched) {
      return matched[1];
    }
  }

  return null;
}

/**
 * 木からテキストだけを集める。
 *
 * sanitize を通した時点でコードフェンスの中身は text ノードだけになるが、要素が残る木を
 * 渡されても中身を落とさないよう、子まで辿って連結する。
 */
function textOf(nodes: readonly ElementContent[]): string {
  return nodes
    .map((node) => {
      switch (node.type) {
        case "text":
          return node.value;
        case "element":
          return textOf(node.children);
        default:
          return "";
      }
    })
    .join("");
}

/**
 * `pre` 要素がコードフェンスなら、その言語と中身を返す。
 *
 * @remarks
 * Markdown のコードフェンスは `pre > code` へ変換されます。この形でない `pre`（HTML から直接
 * 来た整形済みテキスト）は強調表示の対象にせず `null` を返し、呼び出し元がそのまま描画します。
 *
 * @param node - 判定する `pre` 要素
 * @returns コードフェンスならその中身。そうでなければ `null`
 */
export function readCodeFence(node: Element): CodeFence | null {
  const children = node.children.filter(
    (child) => child.type !== "text" || child.value.trim() !== "",
  );

  const [only] = children;

  if (children.length !== 1 || only?.type !== "element" || only.tagName !== "code") {
    return null;
  }

  return { language: languageOf(only), code: textOf(only.children) };
}
