import { z } from "zod";

/**
 * カード 1 枚分の項目。
 *
 * `lang` の `all` は言語を問わず常に表示する項目を表し、言語フィルタの対象外になる
 * （生成物 HTML や外部リンクのように翻訳の対になる相方を持たないもの）。
 */
export const portalItemSchema = z.object({
  name: z.string(),
  path: z.string(),
  source: z.string().optional(),
  lang: z.enum(["en", "ja", "all"]),
});

/** section 内を役割で再分割したまとまり。 */
export const portalSubgroupSchema = z.object({
  title: z.string(),
  items: z.array(portalItemSchema),
});

export const portalSectionSchema = z.object({
  id: z.string(),
  slug: z.string(),
  title: z.string(),
  items: z.array(portalItemSchema),
  subgroups: z.array(portalSubgroupSchema).nullish(),
});

export const portalGroupSchema = z.object({
  title: z.string(),
  slug: z.string(),
  sections: z.array(portalSectionSchema),
});

/** サイドバー下部に常設する、生成 HTML / 外部ツールへのリンク。 */
export const portalReferenceLinkSchema = z.object({
  sectionId: z.string(),
  title: z.string(),
  path: z.string(),
});

export const docsJsonSchema = z.object({
  title: z.string(),
  subtitle: z.string(),
  groups: z.array(portalGroupSchema),
  referenceLinks: z.array(portalReferenceLinkSchema).default([]),
});

export type PortalItem = z.infer<typeof portalItemSchema>;
export type PortalSubgroup = z.infer<typeof portalSubgroupSchema>;
export type PortalSection = z.infer<typeof portalSectionSchema>;
export type PortalGroup = z.infer<typeof portalGroupSchema>;
export type PortalReferenceLink = z.infer<typeof portalReferenceLinkSchema>;
export type DocsJson = z.infer<typeof docsJsonSchema>;

/**
 * 生成物 `docs.json` を検証して読み取る。
 *
 * 生成側と閲覧側が同じリポジトリで動くため、形の不一致は配信事故であって利用者の
 * 入力エラーではない。回復を試みず例外にする。
 */
export function parseDocsJson(input: unknown): DocsJson {
  return docsJsonSchema.parse(input);
}
