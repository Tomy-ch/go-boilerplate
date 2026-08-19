import { z } from "zod";

import { META_KEY, portalManifestSchema } from "./portal-manifest";

/** ルート直下の `docs/*.md` をまとめる section id。 */
const ROOT_MD_SECTION_ID = "architecture";

const subgroupConfigSchema = z.object({
  title: z.string(),
  items: z.array(z.string()).default([]),
});

const groupConfigSchema = z.object({
  title: z.string(),
  sections: z.array(z.string()).default([]),
});

// 各フィールドが既定値を持つため、`{}` を渡せば全項目が埋まる。外側の既定は置かない。
const metaSchema = z.object({
  groups: z.array(groupConfigSchema).default([]),
  section_titles: z.record(z.string(), z.string()).default({}),
  reference_links: z.array(z.string()).default([]),
  subgroups: z.record(z.string(), z.array(subgroupConfigSchema)).default({}),
});

/** `docs/` を走査して得た内容。 */
export type DiscoveredDirectory = {
  name: string;
  hasIndexHtml: boolean;
  enFiles: readonly string[];
  jaFiles: readonly string[];
};

export type DiscoveredDocs = {
  directories: readonly DiscoveredDirectory[];
  rootEnFiles: readonly string[];
  rootJaFiles: readonly string[];
};

export type DocItem = {
  name: string;
  path: string;
  lang: "en" | "ja" | "all";
  source: string;
  guideId: string;
};

export type Subgroup = {
  title: string;
  items: DocItem[];
};

export type DocSection = {
  id: string;
  slug: string;
  title: string;
  items: DocItem[];
  subgroups?: Subgroup[];
};

export type DocGroup = {
  title: string;
  slug: string;
  sections: DocSection[];
};

export type ReferenceLink = {
  sectionId: string;
  title: string;
  path: string;
  source: string;
};

export type DocsJson = {
  title: string;
  subtitle: string;
  groups: DocGroup[];
  referenceLinks: ReferenceLink[];
};

type WorkingSection = {
  id: string;
  title: string;
  items: DocItem[];
  seenPaths: Set<string>;
  subgroups?: Subgroup[];
};

export function autoTitle(value: string): string {
  return value
    .replace(/\.md$/, "")
    .replace(/\.ja$/, "")
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

export function slugify(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    // 上の置換で `-` は 1 個へ潰れているため、端に落とすのも 1 個で足りる。`-+` と書くと
    // 端に無い連なりを走査するたびに後戻りが起きる。
    .replace(/^-|-$/g, "");
}

/** 経路の末尾要素。`split().pop()` と違い、常に文字列になる。 */
function basename(filePath: string): string {
  return filePath.slice(filePath.lastIndexOf("/") + 1);
}

function langOf(destination: string): "en" | "ja" {
  return destination.includes("/ja/") || destination.endsWith(".ja.md") ? "ja" : "en";
}

/**
 * 項目を subgroup へ割り当てるための識別子を導く。
 *
 * @remarks
 * `.ja.md` は 2 段、それ以外の拡張子は 1 段だけ落とします。EN と JA の対を同じ識別子へ寄せつつ、
 * `foo.html.md` と `foo.html` を別物として扱うためです。
 */
export function guideIdOf(filePath: string): string {
  const base = basename(filePath);

  if (/\.ja\.md$/i.test(base)) return base.replace(/\.ja\.md$/i, "");
  if (/\.md$/i.test(base)) return base.replace(/\.md$/i, "");
  if (/\.html$/i.test(base)) return base.replace(/\.html$/i, "");

  return base;
}

/** `lang` の並び順。EN を先、JA を後、言語を持たない項目を最後にする。 */
function langOrder(lang: DocItem["lang"]): number {
  if (lang === "en") return 0;
  if (lang === "ja") return 1;

  return 2;
}

/** 1 つの取り込み元が寄せるセクションと、その中身。 */
type SectionSource = {
  id: string;
  fallbackTitle: string;
  items: DocItem[];
};

/** manifest が宣言した取り込み。 */
function manifestSections(parsed: Record<string, unknown>): SectionSource[] {
  return Object.entries(parsed)
    .filter(([group, entries]) => group !== META_KEY && Array.isArray(entries))
    .map(([group, entries]) => ({
      id: group,
      fallbackTitle: autoTitle(group),
      items: (entries as { dst: string; src: string }[]).map((entry) => ({
        name: autoTitle(basename(entry.dst)),
        path: entry.dst.replace(/^docs\/portal\//, "./"),
        lang: langOf(entry.dst),
        source: entry.src,
        guideId: guideIdOf(entry.dst),
      })),
    }));
}

/**
 * `docs/<dir>` の走査結果。静的 HTML（`index.html`）と、同じディレクトリに並ぶ正本・対訳の
 * Markdown をまとめて 1 セクションにする。
 *
 * 見出しは `index.html` の項目名にも使うため、`ensureSection` と同じ解決（宣言があればそれ、
 * 無ければディレクトリ名から起こす）をここでも行う。
 */
function directorySections(
  directories: readonly DiscoveredDirectory[],
  sectionTitles: Record<string, string>,
): SectionSource[] {
  return directories
    .filter((dir) => dir.hasIndexHtml || dir.enFiles.length > 0 || dir.jaFiles.length > 0)
    .map((dir) => {
      const title = sectionTitles[dir.name] ?? autoTitle(dir.name);
      const indexItem: DocItem[] = dir.hasIndexHtml
        ? [
            {
              name: title,
              path: `../${dir.name}/index.html`,
              lang: "all",
              source: `docs/${dir.name}/index.html`,
              guideId: dir.name,
            },
          ]
        : [];

      return {
        id: dir.name,
        fallbackTitle: autoTitle(dir.name),
        items: [
          ...indexItem,
          ...dir.enFiles.map((file) => markdownItem(file, `../${dir.name}/${file}`, "en", dir.name)),
          ...dir.jaFiles.map((file) =>
            markdownItem(file, `../ja/${dir.name}/${file}`, "ja", dir.name),
          ),
        ],
      };
    });
}

/** `docs/` 直下の正本と対訳。ディレクトリを持たないため 1 つのセクションへ集約する。 */
function rootSection(discovered: DiscoveredDocs): SectionSource[] {
  if (discovered.rootEnFiles.length === 0 && discovered.rootJaFiles.length === 0) return [];

  return [
    {
      id: ROOT_MD_SECTION_ID,
      fallbackTitle: "Architecture Docs",
      items: [
        ...discovered.rootEnFiles.map((file) => markdownItem(file, `../${file}`, "en")),
        ...discovered.rootJaFiles.map((file) => markdownItem(file, `../ja/${file}`, "ja")),
      ],
    },
  ];
}

/** 走査で見つけた Markdown 1 件を項目にする。`source` は正本の位置で、対訳でも `ja/` を挟まない。 */
function markdownItem(
  file: string,
  path: string,
  lang: "en" | "ja",
  directory?: string,
): DocItem {
  return {
    name: autoTitle(file),
    path,
    lang,
    source: directory === undefined ? `docs/${file}` : `docs/${directory}/${file}`,
    guideId: guideIdOf(file),
  };
}

/**
 * manifest と `docs/` の走査結果から docs.json を組み立てる。
 *
 * @remarks
 * 記述の誤り（存在しない section id、複数グループへの重複記載）はビューアーの表示が欠けるだけで
 * 生成自体は成立するため、例外ではなく警告として返します。生成を止めるとポータル全体が
 * 出なくなり、1 か所の記述ミスの影響が広がりすぎます。
 */
export function buildDocsJson(
  manifest: unknown,
  discovered: DiscoveredDocs,
): { docs: DocsJson; warnings: string[] } {
  const warnings: string[] = [];
  const parsed = portalManifestSchema.parse(manifest);
  const meta = metaSchema.parse(parsed[META_KEY] ?? {});

  const sections = new Map<string, WorkingSection>();

  const ensureSection = (id: string, fallbackTitle: string): WorkingSection => {
    const existing = sections.get(id);
    if (existing) return existing;

    const created: WorkingSection = {
      id,
      title: meta.section_titles[id] ?? fallbackTitle,
      items: [],
      seenPaths: new Set(),
    };
    sections.set(id, created);
    return created;
  };

  // 同一 path の item を 2 回積まないためのガード。manifest 由来と auto-discovery 由来が
  // 同じファイルを指した場合に、同じカードが 2 枚出るのを防ぐ。
  const addItem = (section: WorkingSection, item: DocItem): void => {
    if (section.seenPaths.has(item.path)) {
      warnings.push(`重複 item をスキップ: section="${section.id}" path="${item.path}"`);
      return;
    }
    section.seenPaths.add(item.path);
    section.items.push(item);
  };

  for (const source of [
    ...manifestSections(parsed),
    ...directorySections(discovered.directories, meta.section_titles),
    ...rootSection(discovered),
  ]) {
    const section = ensureSection(source.id, source.fallbackTitle);

    for (const item of source.items) {
      addItem(section, item);
    }
  }

  for (const section of sections.values()) {
    section.items.sort(
      (left, right) =>
        langOrder(left.lang) - langOrder(right.lang) || left.name.localeCompare(right.name),
    );
  }

  applySubgroups(sections, meta.subgroups, warnings);

  const { groups, placedSectionIds } = buildGroups(sections, meta.groups, warnings);

  for (const id of meta.reference_links) placedSectionIds.add(id);

  const orphans = [...sections.values()].filter((section) => !placedSectionIds.has(section.id));

  if (orphans.length) {
    groups.push({
      title: "Uncategorized",
      slug: "uncategorized",
      sections: orphans
        .toSorted((left, right) => left.title.localeCompare(right.title))
        .map(toDocSection),
    });
    warnings.push(
      `meta.groups 未配置のセクション (${orphans.map((section) => section.id).join(", ")}) を "Uncategorized" に集約しました`,
    );
  }

  return {
    docs: {
      title: "go-boilerplate Documentation",
      subtitle:
        "This document portal provides access to the repository implementation, test results, E-R diagrams, and more.",
      groups,
      referenceLinks: buildReferenceLinks(sections, meta.reference_links, warnings),
    },
    warnings,
  };
}

/**
 * `meta.subgroups` に登録された section へ小見出しを付ける。
 *
 * @remarks
 * `items` はそのまま残し、`subgroups` を追加で持たせます。どの小見出しにも割り当てられなかった
 * 項目は末尾の "Other" へ寄せます。落とすと、登録漏れが「表示されない」形で現れるためです。
 */
function applySubgroups(
  sections: Map<string, WorkingSection>,
  configs: Record<string, { title: string; items: string[] }[]>,
  warnings: string[],
): void {
  for (const [sectionId, subgroupConfigs] of Object.entries(configs)) {
    const section = sections.get(sectionId);

    if (!section) {
      warnings.push(`meta.subgroups: section id "${sectionId}" は存在しないので無視します`);
      continue;
    }

    const subgroups = resolveSubgroups(sectionId, section, subgroupConfigs, warnings);
    if (subgroups.length) section.subgroups = subgroups;
  }
}

/** section の項目を guide id で引けるようにする。guide id を持たない項目はどの小見出しにも入らない。 */
function indexByGuideId(items: readonly DocItem[]): Map<string, DocItem[]> {
  const byGuideId = new Map<string, DocItem[]>();

  for (const item of items) {
    if (!item.guideId) continue;
    const bucket = byGuideId.get(item.guideId) ?? [];
    bucket.push(item);
    byGuideId.set(item.guideId, bucket);
  }

  return byGuideId;
}

/** section 1 つ分の小見出しを組み立てる。割り当てられなかった項目は末尾の "Other" へ寄せる。 */
function resolveSubgroups(
  sectionId: string,
  section: WorkingSection,
  subgroupConfigs: readonly { title: string; items: string[] }[],
  warnings: string[],
): Subgroup[] {
  const byGuideId = indexByGuideId(section.items);
  const subgroups: Subgroup[] = [];
  const placedGuideIds = new Set<string>();

  for (const config of subgroupConfigs) {
    const items = config.items.flatMap((guideId) => {
      const matched = byGuideId.get(guideId);

      if (!matched?.length) {
        warnings.push(
          `meta.subgroups[${sectionId}][${config.title}]: guide id "${guideId}" は存在しないので無視します`,
        );

        return [];
      }

      placedGuideIds.add(guideId);

      return matched;
    });

    if (items.length) subgroups.push({ title: config.title, items });
  }

  const others = section.items.filter((item) => !placedGuideIds.has(item.guideId));
  if (others.length) subgroups.push({ title: "Other", items: others });

  return subgroups;
}

function buildGroups(
  sections: Map<string, WorkingSection>,
  configs: readonly { title: string; sections: string[] }[],
  warnings: string[],
): { groups: DocGroup[]; placedSectionIds: Set<string> } {
  const groups: DocGroup[] = [];
  const placedSectionIds = new Set<string>();

  for (const config of configs) {
    const groupSections: WorkingSection[] = [];

    for (const id of config.sections) {
      const section = sections.get(id);

      if (!section) {
        warnings.push(
          `meta.groups: section id "${id}" は存在しないので無視します (group: ${config.title})`,
        );
        continue;
      }
      // 別グループで既に配置済みの section id は重複配置しない（DOM id 衝突 / 二重表示を防ぐ）。
      if (placedSectionIds.has(id)) {
        warnings.push(
          `meta.groups: section id "${id}" が複数グループに記載されています。"${config.title}" 側は無視します`,
        );
        continue;
      }

      groupSections.push(section);
      placedSectionIds.add(id);
    }

    if (groupSections.length) {
      groups.push({
        title: config.title,
        slug: slugify(config.title),
        sections: groupSections.map(toDocSection),
      });
    }
  }

  return { groups, placedSectionIds };
}

/** サイドバー下部の常設リンク。`meta.reference_links` の順を保ち、各 section の先頭項目を代表にする。 */
function buildReferenceLinks(
  sections: Map<string, WorkingSection>,
  ids: readonly string[],
  warnings: string[],
): ReferenceLink[] {
  const links: ReferenceLink[] = [];

  for (const id of ids) {
    const section = sections.get(id);

    if (!section) {
      warnings.push(`meta.reference_links: section id "${id}" は存在しないので無視します`);
      continue;
    }

    const primary = section.items[0];
    if (!primary) continue;

    links.push({ sectionId: id, title: section.title, path: primary.path, source: primary.source });
  }

  return links;
}

/** 内部フィールド（重複検出用の集合）を落として出力形へ移す。 */
function toDocSection(section: WorkingSection): DocSection {
  return {
    id: section.id,
    slug: slugify(section.id),
    title: section.title,
    items: section.items,
    ...(section.subgroups ? { subgroups: section.subgroups } : {}),
  };
}

/** ビューアー自身と翻訳ツリーは section にしない。前者は生成物、後者は各 section の一部。 */
const NON_SECTION_DIRECTORIES: ReadonlySet<string> = new Set(["portal", "ja"]);

/** `docs/` 直下のディレクトリを section として扱うか。 */
export function isSectionDirectory(name: string): boolean {
  return !NON_SECTION_DIRECTORIES.has(name);
}

/** ビューアーへ載せる Markdown か。 */
export function isMarkdownFile(fileName: string): boolean {
  return fileName.endsWith(".md");
}

/**
 * section 名を表示順へ整列する。
 *
 * @remarks
 * `localeCompare` を使うのは、`readdir` の順がファイルシステム依存で、同じツリーでも
 * 環境ごとにビューアーの並びが変わるためです。生成物の差分が環境で揺れると、
 * 生成物の drift 検査が意味を失います。
 */
export function sortSectionNames(names: readonly string[]): string[] {
  return [...names].sort((left, right) => left.localeCompare(right));
}
