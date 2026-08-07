import type { PortalGroup, PortalItem, PortalSection } from "../docs-json/docs-json";

/** 利用者が選べる表示言語。`docs.json` の項目言語（`en` / `ja` / `all`）とは別の軸。 */
export type PortalLang = "EN" | "JA";

/**
 * section に実際に適用する言語を決める。
 *
 * JA を選んでいても JA の項目が 1 件も無い section は EN へ落とす。section 単位で
 * 決めた言語を配下の subgroup へ共有させることで、同じ section の中で subgroup ごとに
 * 言語が混ざる状態を防ぐ。
 */
export function effectiveLangFor(items: readonly PortalItem[], lang: PortalLang): PortalLang {
  if (lang !== "JA") {
    return "EN";
  }

  return items.some((item) => item.lang === "ja") ? "JA" : "EN";
}

/** 決定済みの言語で項目を絞る。`all` の項目は言語によらず先頭に残る。 */
export function filterItemsByLang(
  items: readonly PortalItem[],
  effectiveLang: PortalLang,
): PortalItem[] {
  const always = items.filter((item) => item.lang === "all");
  const target = effectiveLang === "EN" ? "en" : "ja";

  return [...always, ...items.filter((item) => item.lang === target)];
}

function filterSection(section: PortalSection, lang: PortalLang): PortalSection {
  const sectionLang = effectiveLangFor(section.items, lang);
  const subgroups = section.subgroups
    ?.map((subgroup) => ({
      title: subgroup.title,
      items: filterItemsByLang(subgroup.items, sectionLang),
    }))
    .filter((subgroup) => subgroup.items.length > 0);

  return {
    id: section.id,
    slug: section.slug,
    title: section.title,
    items: filterItemsByLang(section.items, sectionLang),
    subgroups: subgroups ?? null,
  };
}

/**
 * 表示言語で group / section / subgroup を絞り込み、空になったものを落とす。
 *
 * section は items と subgroups のどちらかに中身が残っていれば保持する。items を持たず
 * subgroups だけで構成される section を落とさないため。
 */
export function applyLangFilter(groups: readonly PortalGroup[], lang: PortalLang): PortalGroup[] {
  return groups
    .map((group) => ({
      title: group.title,
      slug: group.slug,
      sections: group.sections
        .map((section) => filterSection(section, lang))
        .filter((section) => section.items.length > 0 || (section.subgroups?.length ?? 0) > 0),
    }))
    .filter((group) => group.sections.length > 0);
}
