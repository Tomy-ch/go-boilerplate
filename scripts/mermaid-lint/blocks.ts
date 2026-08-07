import path from "node:path";

/** Markdown 中の ` ```mermaid ` フェンス 1 つ分。`startLine` は 1 始まり。 */
export type MermaidBlock = {
  startLine: number;
  code: string;
};

// markdownlint-cli2 の MD_GLOBS と対象範囲を揃える（生成物・vendor・AGENTS.md を除外）。
// DIRS は名前がどこに現れても、PREFIXES はリポジトリルートからの相対パスの先頭一致で、
// FILES は完全一致で除外する。
export const EXCLUDE_DIRS: ReadonlySet<string> = new Set(["vendor", "node_modules", ".git", "tmp"]);
export const EXCLUDE_PREFIXES: readonly string[] = [
  "docs/portal/guides",
  "docs/coverage",
  "docs/db-schema",
  "graphify-out",
];
export const EXCLUDE_FILES: ReadonlySet<string> = new Set(["AGENTS.md"]);

/**
 * 除外プレフィックス配下かどうかを判定する。
 *
 * @remarks
 * 境界は区切り文字で見ます。先頭一致だけで判定すると `docs/coverage` が
 * `docs/coverage-report` にも一致し、除外するつもりのないツリーまで消えます。
 */
export function isExcludedPath(relativePath: string): boolean {
  return EXCLUDE_PREFIXES.some(
    (prefix) => relativePath === prefix || relativePath.startsWith(`${prefix}${path.sep}`),
  );
}

/** 検証対象の Markdown かどうかを判定する。 */
export function isTargetMarkdown(relativePath: string): boolean {
  return (
    relativePath.endsWith(".md") && !EXCLUDE_FILES.has(relativePath) && !isExcludedPath(relativePath)
  );
}

/**
 * Markdown から ` ```mermaid ` フェンスを開始行付きで抜き出す。
 *
 * @remarks
 * 閉じ判定は「フェンス文字だけの行」に限ります。開始フェンスと同じ文字数以上でなければ
 * 閉じないという CommonMark の規約に乗せてあり、図の中に現れるバッククォートで途中終了しません。
 */
export function extractMermaidBlocks(content: string): MermaidBlock[] {
  const lines = content.split("\n");
  const blocks: MermaidBlock[] = [];
  const fence = /^(\s*)(`{3,}|~{3,})\s*mermaid\s*$/;

  for (let i = 0; i < lines.length; i++) {
    const matched = fence.exec(lines[i]);
    if (!matched) continue;

    const marker = matched[2];
    const close = new RegExp(`^\\s*${marker[0] === "\`" ? "\`" : "~"}{${marker.length},}\\s*$`);
    const body: string[] = [];
    let j = i + 1;

    for (; j < lines.length; j++) {
      if (close.test(lines[j])) break;
      body.push(lines[j]);
    }

    blocks.push({ startLine: i + 1, code: body.join("\n") });
    i = j;
  }

  return blocks;
}

/**
 * 走査中に出会ったディレクトリへ降りるかを判定する。
 *
 * @remarks
 * ディレクトリ名（`node_modules` 等どこにあっても外すもの）と、リポジトリルートからの相対パス
 * （`docs/portal/guides` 等その場所だけ外すもの）の両方を見ます。片方だけだと、生成物の
 * ディレクトリ名がありふれた名前になった時点で検査対象へ紛れ込みます。
 */
export function shouldDescend(dirName: string, relativePath: string): boolean {
  return !EXCLUDE_DIRS.has(dirName) && !isExcludedPath(relativePath);
}
