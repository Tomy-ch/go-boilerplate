import path from "node:path";

export type Finding = { file: string; message: string };
export type ADRIndex = ReadonlyMap<string, string>;

const GENERATED_DOC_PREFIXES = ["docs/openapi/", "docs/coverage/", "docs/db-schema/", "docs/godoc/", "docs/portal/"];
const GENERATED_SUFFIXES = [".gen.go", ".gen.sql", ".sql.go", ".gen.yaml"];
export const ADR_FILE = /^docs\/adr\/(\d{4})-(.+)\.md$/;
const REFERENCE = /ADR-(\d{4})/g;
// パス形式の参照。`docs/adr/NNNN-slug.md` も `](../docs/adr/NNNN-slug.md)` も同じ形で拾う。
const PATH_REFERENCE = /adr\/(\d{4})-([a-z0-9-]+)(\.ja)?\.md/g;
// slug 注釈は参照の直後にしか現れない。
const ANNOTATION = /\s*\(([-a-z0-9]+)\)/y;
// `kind: adr` に属する id / slug を探す行数。次の `kind:` に当たった時点でも打ち切る。
const ENTRY_SPAN = 6;

export const TRANSLATION_EXCLUSIONS = [
  { prefix: "docs/spec/", reason: "feature specifications are intentionally English-only" },
] as const;

function isGenerated(file: string): boolean {
  return GENERATED_DOC_PREFIXES.some((prefix) => file.startsWith(prefix)) || GENERATED_SUFFIXES.some((suffix) => file.endsWith(suffix));
}

export function isEligible(file: string): boolean {
  return /\.(md|go|sql|ya?ml|[cm]?js|ts)$/.test(file) && !file.endsWith(".test.ts") && !isGenerated(file) && file !== "AGENTS.md";
}

export function adrIndex(files: readonly string[]): Map<string, string> {
  return new Map(files.flatMap((file) => {
    const matched = ADR_FILE.exec(file);
    return matched === null ? [] : [[matched[1], matched[2]] as const];
  }));
}

export function normalizeReferences(source: string, adrs: ADRIndex): string {
  return source.replace(/ADR-(\d{4})(?!\s*\()/g, (whole, number: string) =>
    adrs.has(number) ? `${whole} (${adrs.get(number)})` : whole,
  );
}

export function checkReferences(file: string, source: string, adrs: ADRIndex): Finding[] {
  return [...source.matchAll(REFERENCE)].flatMap((match) => {
    const slug = adrs.get(match[1]);
    if (slug === undefined) return [{ file, message: `ADR-${match[1]} does not exist` }];
    ANNOTATION.lastIndex = match.index + match[0].length;
    return ANNOTATION.exec(source)?.[1] === slug ? [] : [{ file, message: `ADR-${match[1]} must include (${slug})` }];
  });
}

/**
 * ADR をパスで指した参照が、番号どおりの slug を綴っているかを検査する。
 *
 * `ADR-NNNN (slug)` 形式は checkReferences が見るが、リンク先やパス表記は素通りしていた。
 * 番号と slug が整合していてもリンク先だけ別の ADR を指す形（注釈は正しいのにリンク先の slug が違う）は
 * この検査でしか捕まらない。
 */
export function checkPathReferences(file: string, source: string, adrs: ADRIndex): Finding[] {
  return [...source.matchAll(PATH_REFERENCE)].flatMap((match) => {
    const slug = adrs.get(match[1]);
    if (slug === undefined) return [{ file, message: `ADR-${match[1]} does not exist (referenced as ${match[0]})` }];
    return slug === match[2] ? [] : [{ file, message: `${match[0]} must be ${match[1]}-${slug}${match[3] ?? ""}.md` }];
  });
}

/**
 * `kind: adr` の構造化エントリが、番号と slug の両方を持ち、その 2 つが一致しているかを検査する。
 *
 * 番号と slug が別フィールドに分かれているため `ADR-NNNN (slug)` の正規表現には一致せず、
 * checkReferences も checkPathReferences も素通りする。ADR は途中に挿入されると以降の番号が
 * ずれるため、食い違ったときに正しいのは slug の側で、番号を引き直す。
 */
export function checkStructuredReferences(file: string, source: string, adrs: ADRIndex): Finding[] {
  const numbers = new Map([...adrs].map(([number, slug]) => [slug, number] as const));
  const lines = source.split("\n");
  return lines.flatMap((line, index) => {
    if (!/^\s*(-\s+)?kind:\s*["']?adr["']?\s*$/.test(line)) return [];
    const indent = line.replace(/^(\s*)(-\s+)?.*$/, "$1$2").length;
    const window = lines.slice(index + 1, index + 1 + ENTRY_SPAN);
    // 次の要素（`- ` 始まり）か、この要素より浅い行に当たったらそこで打ち切る。
    const stop = window.findIndex((next) => /^\s*-\s/.test(next) || next.search(/\S/) < indent);
    const entry = stop === -1 ? window : window.slice(0, stop);
    const id = entry.map((next) => /^\s*id:\s*["']?(\d{4})["']?\s*$/.exec(next)?.[1]).find((value) => value !== undefined);
    const slug = entry.map((next) => /^\s*slug:\s*["']?([a-z0-9-]+)["']?\s*$/.exec(next)?.[1]).find((value) => value !== undefined);
    const at = `L${index + 1}`;
    if (id === undefined || slug === undefined) return [{ file, message: `${at}: kind: adr entry must carry both id and slug` }];
    if (adrs.get(id) === slug) return [];
    const expected = numbers.get(slug);
    if (expected !== undefined) return [{ file, message: `${at}: slug ${slug} is id "${expected}", not "${id}"` }];
    return [{ file, message: adrs.has(id) ? `${at}: id "${id}" is slug ${adrs.get(id)}, not ${slug}` : `${at}: id "${id}" and slug ${slug} both name no ADR` }];
  });
}

export function expectedTranslation(file: string): string | null {
  if (!file.startsWith("docs/") || !file.endsWith(".md") || file.startsWith("docs/ja/") || file.startsWith("docs/adr/") || isGenerated(file) || translationExclusion(file) !== null) return null;
  return path.join("docs/ja", path.relative("docs", path.dirname(file)), `${path.basename(file, ".md")}.ja.md`);
}

export function translationExclusion(file: string): string | null {
  return TRANSLATION_EXCLUSIONS.find(({ prefix }) => file.startsWith(prefix))?.reason ?? null;
}

export function checkTranslations(files: readonly string[]): Finding[] {
  const present = new Set(files);
  const missing = files.flatMap((file) => {
    const translation = expectedTranslation(file);
    return translation !== null && !present.has(translation) ? [{ file, message: `missing ${translation}` }] : [];
  });
  const orphans = files.flatMap((file) => {
    if (!file.startsWith("docs/ja/") || file.startsWith("docs/ja/adr/") || !file.endsWith(".ja.md")) return [];
    const canonical = path.join("docs", path.relative("docs/ja", file).replace(/\.ja\.md$/, ".md"));
    return expectedTranslation(canonical) !== file || !present.has(canonical) ? [{ file, message: `orphan translation for ${canonical}` }] : [];
  });
  return [...missing, ...orphans];
}

export function checkTranslationExclusions(files: readonly string[]): Finding[] {
  return TRANSLATION_EXCLUSIONS.flatMap(({ prefix }) => {
    return files.some((file) => file.startsWith(prefix)) ? [] : [{ file: prefix, message: "stale translation exclusion" }];
  });
}
