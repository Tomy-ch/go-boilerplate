import type { Element } from "hast";
import { type Components, toJsxRuntime } from "hast-util-to-jsx-runtime";
import type { ComponentProps } from "react";
import { Fragment, jsx, jsxs } from "react/jsx-runtime";

import { cn } from "@/components/cn";

import { CodeBlock } from "../code-block/code-block";
import { readCodeFence } from "../code-fence/code-fence";
import { MermaidDiagram } from "../mermaid-diagram/mermaid-diagram";
import type { SanitizedDocument } from "../sanitize/sanitized-document";

/**
 * {@link DocumentContent} が受け取る props です。
 *
 * `children` と `dangerouslySetInnerHTML` は受け取りません。本文は `content` だけが決めます。
 */
export type DocumentContentProps = Omit<
  ComponentProps<"div">,
  "children" | "content" | "dangerouslySetInnerHTML"
> & {
  /** 表示する文書。sanitize を通した値だけがこの型を持ちます。 */
  content: SanitizedDocument;
};

type DocumentPreProps = ComponentProps<"pre"> & {
  /** 変換元の hast ノード。`passNode` により渡されます。 */
  node: Element;
};

/**
 * コードフェンスを言語ごとの描画へ振り分けます。
 *
 * @remarks
 * 判定に変換前の hast ノードを使います。React 要素になった後の子から元のテキストを組み直すと、
 * 強調表示のために作った要素まで本文として拾ってしまいます。
 */
function DocumentPre({ node, ...props }: DocumentPreProps) {
  const fence = readCodeFence(node);

  if (!fence) {
    return <pre {...props} />;
  }

  if (fence.language === "mermaid") {
    return <MermaidDiagram code={fence.code} />;
  }

  return <CodeBlock code={fence.code} language={fence.language} />;
}

const DOCUMENT_COMPONENTS: Partial<Components> = { pre: DocumentPre };

/**
 * sanitize 済みのドキュメントを本文として表示する。
 *
 * @remarks
 * 組版は `typeset` の CSS 基盤が持ち、ドキュメント用の preset を既定で当てる。描画方式と
 * コードフェンスの振り分けは README（Rendering documents）参照。
 */
export function DocumentContent({ content, className, ...props }: DocumentContentProps) {
  return (
    <div className={cn("typeset typeset-docs", className)} data-slot="document-content" {...props}>
      {toJsxRuntime(content.root, {
        Fragment,
        components: DOCUMENT_COMPONENTS,
        jsx,
        jsxs,
        passNode: true,
      })}
    </div>
  );
}
