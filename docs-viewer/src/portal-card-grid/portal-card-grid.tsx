import { ExternalLink } from "lucide-react";
import { useCallback } from "react";

import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/design-system/display/card/card";

import type { PortalItem } from "../docs-json/docs-json";

/** Markdown は面の中で開き、生成 HTML や外部ツールは別タブへ送る。 */
function isDocument(item: PortalItem): boolean {
  return item.path.endsWith(".md");
}

/**
 * カードの面全体を操作の当たり判定にする class。
 *
 * `Card` は `asChild` を持たず `div` を描画するため、カード自体を `button` や `a` にできない。
 * 操作要素を title に置いたうえで、擬似要素で当たり判定だけをカード全体へ広げる。役割は本物の
 * `button` / `a` が持つため、支援技術には正しく伝わる。
 */
const STRETCHED_TARGET = "after:absolute after:inset-0 after:content-['']";

type PortalCardProps = {
  item: PortalItem;
  onOpenDocument: (item: PortalItem) => void;
};

function PortalCard({ item, onOpenDocument }: PortalCardProps) {
  const onOpen = useCallback(() => onOpenDocument(item), [item, onOpenDocument]);

  return (
    <Card className="relative h-full transition-colors focus-within:outline-2 focus-within:outline-foreground focus-within:outline-offset-2 hover:bg-accent/50">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          {isDocument(item) ? (
            <button
              className={`${STRETCHED_TARGET} text-left focus-visible:outline-none`}
              onClick={onOpen}
              type="button"
            >
              {item.name}
            </button>
          ) : (
            <>
              <a
                className={`${STRETCHED_TARGET} focus-visible:outline-none`}
                href={item.path}
                rel="noopener noreferrer"
                target="_blank"
              >
                {item.name}
              </a>
              <ExternalLink aria-hidden className="size-4 shrink-0" />
            </>
          )}
        </CardTitle>
        <CardDescription>{item.source ?? item.path}</CardDescription>
      </CardHeader>
    </Card>
  );
}

export type PortalCardGridProps = {
  items: readonly PortalItem[];
  /** Markdown の項目が選ばれたときに呼ばれる。 */
  onOpenDocument: (item: PortalItem) => void;
};

/**
 * 項目をカードとして並べる。
 *
 * @remarks
 * 項目の行き先によって操作要素を変えます。Markdown はこのページの中で面を開くため `button`、
 * それ以外は別の文書への移動なので `a` です。見た目を揃えるために片方へ寄せると、keyboard と
 * 支援技術には「押すと何が起きるか」が伝わらなくなります。
 */
export function PortalCardGrid({ items, onOpenDocument }: PortalCardGridProps) {
  return (
    <ul className="grid list-none grid-cols-1 gap-4 p-0 sm:grid-cols-2 xl:grid-cols-3">
      {items.map((item) => (
        <li key={`${item.name}-${item.path}`}>
          <PortalCard item={item} onOpenDocument={onOpenDocument} />
        </li>
      ))}
    </ul>
  );
}
