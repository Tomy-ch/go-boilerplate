import type { PortalGroup, PortalItem } from "../docs-json/docs-json";

/**
 * 検索対象 1 件。所属する section / group の名前を項目へ畳み込み、検索結果が
 * どこの項目なのかを単体で示せるようにする。
 */
export type SearchEntry = PortalItem & {
  sectionTitle: string;
  groupTitle: string;
};

/** 検索でマッチさせるキー。Fuse など外部の検索実装へ渡す。 */
export const searchKeys = ["name", "sectionTitle", "groupTitle", "source", "path"] as const;

/**
 * 表示中の group から検索コーパスを組む。
 *
 * subgroup 側の項目も平坦化して含める。subgroup だけに置かれた項目が検索から漏れると、
 * 利用者からは「存在するのに引けない」状態になるため。
 */
export function buildSearchCorpus(groups: readonly PortalGroup[]): SearchEntry[] {
  const entries: SearchEntry[] = [];

  for (const group of groups) {
    for (const section of group.sections) {
      const items = [...section.items, ...(section.subgroups ?? []).flatMap((sub) => sub.items)];

      for (const item of items) {
        entries.push({ ...item, sectionTitle: section.title, groupTitle: group.title });
      }
    }
  }

  return entries;
}
