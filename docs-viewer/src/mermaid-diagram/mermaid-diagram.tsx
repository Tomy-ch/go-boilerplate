import { useEffect, useId, useState } from "react";

import { CodeBlock } from "../code-block/code-block";
import { renderMermaid } from "./render-mermaid";

/**
 * `useId` が返す値は `:` を含み、SVG の id と mermaid が生成する CSS 選択子に使えない。
 * 一意性は保ったまま、選択子として書ける文字だけに落とす。
 */
function toElementId(generated: string): string {
  return `mermaid${generated.replace(/[^a-zA-Z0-9]/g, "")}`;
}

export type MermaidDiagramProps = {
  /** mermaid の図の定義。 */
  code: string;
};

/**
 * mermaid のコードフェンスを図として描画します。
 *
 * @remarks
 * 描画は SVG を流し込む形で行います。mermaid は文字列として SVG を返し、React が管理する木へ
 * 要素として組み込む経路を持ちません。`securityLevel: "strict"` により、図の定義に書かれた
 * HTML と script は実行されません。
 *
 * 描画できない図は定義をそのまま出します。mermaid の文法誤りは書き手にしか直せず、図が
 * 消えるだけでは何が起きたのか読み手にも書き手にも伝わりません。
 */
export function MermaidDiagram({ code }: MermaidDiagramProps) {
  const id = toElementId(useId());
  const [svg, setSvg] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let applies = true;

    renderMermaid(id, code)
      .then((rendered) => {
        if (applies) {
          setSvg(rendered);
        }
      })
      .catch(() => {
        if (applies) {
          setFailed(true);
        }
      });

    return () => {
      applies = false;
    };
  }, [id, code]);

  if (failed) {
    return <CodeBlock code={code} language="mermaid" />;
  }

  if (svg === null) {
    return null;
  }

  return (
    <div
      className="flex justify-center overflow-x-auto"
      dangerouslySetInnerHTML={{ __html: svg }}
      data-slot="mermaid-diagram"
    />
  );
}
