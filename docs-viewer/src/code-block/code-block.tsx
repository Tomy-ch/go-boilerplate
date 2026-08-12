import { useEffect, useState } from "react";

import { highlightCode } from "./highlight-code";
import "./syntax-theme.css";

export type CodeBlockProps = {
  /** フェンスの中身。 */
  code: string;
  /** フェンスの言語表記。無い場合は `null`。 */
  language: string | null;
};

/**
 * コードフェンスを強調表示付きで描画します。
 *
 * @remarks
 * 最初は素のテキストで描き、highlight.js の読み込みが済んだ時点で色付きへ差し替えます。
 * 強調表示は読みやすさのためのもので、それを待って本文を出す理由がありません。
 *
 * 差し替えは highlight.js が返した HTML を流し込む形で行います。同じ木を React が持ったまま
 * DOM を書き換えると再描画で復元されるため、色付けの結果は React が中身を管理しない要素へ
 * 渡します。highlight.js は入力を escape してから span で包むため、この HTML に元のテキストが
 * markup として現れることはありません。
 */
export function CodeBlock({ code, language }: CodeBlockProps) {
  const [highlighted, setHighlighted] = useState<string | null>(null);
  const className = language ? `hljs language-${language}` : "hljs";

  useEffect(() => {
    let applies = true;

    highlightCode(code, language)
      .then((html) => {
        if (applies) {
          setHighlighted(html);
        }
      })
      .catch(() => undefined);

    return () => {
      applies = false;
    };
  }, [code, language]);

  return (
    <pre>
      {highlighted === null ? (
        <code className={className}>{code}</code>
      ) : (
        <code className={className} dangerouslySetInnerHTML={{ __html: highlighted }} />
      )}
    </pre>
  );
}
