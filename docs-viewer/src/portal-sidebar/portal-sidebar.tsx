import { ExternalLink } from "lucide-react";
import { useId } from "react";

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/design-system/container/accordion/accordion";

import type { PortalGroup, PortalReferenceLink } from "../docs-json/docs-json";
import { formatHashRoute } from "../hash-route/hash-route";

export type PortalSidebarProps = {
  groups: readonly PortalGroup[];
  /** 本文側が表示している group。開いた状態で描画する。 */
  activeGroupSlug: string | null;
  referenceLinks: readonly PortalReferenceLink[];
};

/**
 * group と section への導線を並べる。
 *
 * @remarks
 * `Accordion` は native の `details` であり、常に一項目だけを開く制御を持ちません。よって表示中の
 * group 以外も開いたままにできます。ドキュメントを見比べる用途では複数開ける方が都合がよいため、
 * 排他にするための client island は足していません。
 *
 * group の見出しは開閉だけを担い、link を持ちません。`AccordionTrigger` は native の `summary` で
 * それ自体が操作要素であり、中へ link を置くと操作要素の入れ子になって keyboard の到達順が壊れます
 * （axe の `nested-interactive`）。遷移は section 側の link が担い、link は group と section の
 * 両方を指すため、section を選べば group も切り替わります。
 *
 * 遷移は位置ハッシュへの link で表します。ビューアーは静的配信されるため、共有・履歴・戻る操作に
 * 対して復元可能なのはハッシュだけです。
 */
export function PortalSidebar({ activeGroupSlug, groups, referenceLinks }: PortalSidebarProps) {
  const referenceHeadingId = useId();

  return (
    <nav aria-label="ドキュメント" className="flex flex-col gap-6">
      <Accordion>
        {groups.map((group) => (
          <AccordionItem key={group.slug} open={group.slug === activeGroupSlug}>
            <AccordionTrigger>{group.title}</AccordionTrigger>
            <AccordionContent>
              <ul className="flex list-none flex-col gap-1 p-0">
                {group.sections.map((section) => (
                  <li key={section.slug}>
                    <a
                      className="block rounded-md px-2 py-1 text-muted-foreground text-sm hover:bg-accent hover:text-accent-foreground"
                      href={formatHashRoute(group.slug, section.slug)}
                    >
                      {section.title}
                    </a>
                  </li>
                ))}
              </ul>
            </AccordionContent>
          </AccordionItem>
        ))}
      </Accordion>

      {referenceLinks.length === 0 ? null : (
        <section aria-labelledby={referenceHeadingId} className="flex flex-col gap-2">
          <h2
            className="px-2 font-medium text-muted-foreground text-xs uppercase"
            id={referenceHeadingId}
          >
            Reference
          </h2>
          <ul className="flex list-none flex-col gap-1 p-0">
            {referenceLinks.map((link) => (
              <li key={link.sectionId}>
                <a
                  className="flex items-center gap-2 rounded-md px-2 py-1 text-sm hover:bg-accent hover:text-accent-foreground"
                  href={link.path}
                  rel="noopener noreferrer"
                  target="_blank"
                >
                  {link.title}
                  <ExternalLink aria-hidden className="size-3.5 shrink-0" />
                </a>
              </li>
            ))}
          </ul>
        </section>
      )}
    </nav>
  );
}
