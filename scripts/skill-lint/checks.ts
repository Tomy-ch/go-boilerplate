import path from "node:path";
import { fileURLToPath } from "node:url";

// `.claude/**` / `.codex/**` の定義を意味的に検査するための、判断を含まない部品群。
// ファイルシステムへ触るのは呼び出し元（scripts/skill-lint/index.ts）の責務にする。

/**
 * 片側の環境にしか存在しない skill と、その理由。
 *
 * @remarks
 * 未移植そのものは異常ではありません（`sync-ai` は逐語コピーではなく意味ポートであり、移植には
 * 判断が要ります）。異常なのは「未移植であることが宣言されていない」状態なので、理由を書かせた
 * うえで許可します。両側に揃ったらこの表から消してください（残すと stale として落ちます）。
 */
export const PLATFORM_ONLY_SKILLS: ReadonlyMap<string, string> = new Map([
  [
    "supply-chain-triage",
    "Codex へ未移植。Codex 側の冷却窓スキル群が本スキルへの連鎖を持たないため、移植方針の判断が保留されている。",
  ],
]);

/** allowlist の在り処。違反メッセージから編集先へ辿れるようにするため、自身のパスを持つ。 */
export function allowlistLocation(repoRoot: string): string {
  // `URL#pathname` はパーセントエンコードを解かず、空白や非 ASCII を含む clone 先で壊れた相対パスになる。
  return path.relative(repoRoot, fileURLToPath(import.meta.url));
}

/** ファイル索引・参照検査から外すディレクトリ（生成物 / 実行時成果物 / 外部由来）。 */
export const EXCLUDE_DIRS: ReadonlySet<string> = new Set([".git", "node_modules", "vendor", "tmp"]);

/**
 * 参照検査の対象外にする先頭セグメント。
 *
 * @remarks
 * `tmp/` 配下はスキル実行中に生成されるため、静的なファイルシステム検査では存在しないのが正常です。
 */
export const PATH_ROOT_DENY: ReadonlySet<string> = new Set(["tmp", ".git"]);

/** 意図的に実在しない参照（仮定の例示・任意配置）を抑止するための行内ディレクティブ。 */
export const IGNORE_DIRECTIVE = "<!-- skill-lint-ignore -->";

/**
 * ディレクトリを伴わない設定ファイル名（`mise.toml` / `tools.yaml`）の形。
 *
 * @remarks
 * 設定ファイルはリポジトリ内で名前が一意に定まるため、配置を書かずに名前だけで参照されることが
 * 多く、SSOT が移動・改名しても本文だけが古い名前で残りやすい。
 */
export const CONFIG_FILE_RE = /^[.\w][\w.-]*\.(ya?ml|toml|json)$/;

export const WILDCARD_RE = /[*<]/;

/**
 * `{a,b}` を展開して候補文字列の配列にする。
 *
 * @remarks
 * Make ターゲットでは実ターゲット名の列挙（全て実在すべき）、パスでは glob の選択（どれか 1 つ
 * 当たれば良い）と意味が異なるため、判定側で all / any を使い分けます。
 */
export function expandBraces(text: string): string[] {
  const matched = /\{([^{}]*)\}/.exec(text);
  if (!matched) return [text];

  return matched[1]
    .split(",")
    .flatMap((alternative) =>
      expandBraces(
        text.slice(0, matched.index) + alternative + text.slice(matched.index + matched[0].length),
      ),
    );
}

/**
 * ドキュメント中のプレースホルダ表記を正規表現へ変換する。
 *
 * @remarks
 * `<name>` は書き手が埋める任意の 1 セグメント、`**` は任意階層、`*` は 1 セグメント内の任意文字列。
 */
export function placeholderToRegExp(
  text: string,
  { segmentSeparator }: { segmentSeparator: boolean },
): RegExp {
  const anySegmentChars = segmentSeparator ? "[^/]*" : ".*";
  const placeholderChars = segmentSeparator ? "[^/]+" : ".+";
  let out = "";

  for (let i = 0; i < text.length; i++) {
    const character = text[i];

    if (character === "<") {
      const close = text.indexOf(">", i);
      if (close === -1) {
        out += "<";
        continue;
      }
      out += placeholderChars;
      i = close;
      continue;
    }

    if (character === "*") {
      if (segmentSeparator && text.slice(i, i + 3) === "**/") {
        out += "(?:[^/]+/)*";
        i += 2;
        continue;
      }
      if (segmentSeparator && text.slice(i, i + 2) === "**") {
        out += ".*";
        i += 1;
        continue;
      }
      out += anySegmentChars;
      continue;
    }

    out += character.replace(/[.+^${}()|[\]\\?]/g, "\\$&");
  }

  return new RegExp(`^${out}$`);
}

export type SourceLine = {
  line: string;
  lineNo: number;
};

/**
 * 行を走査しつつコードフェンス（``` / ~~~）の外側だけを返す。
 *
 * @remarks
 * フェンス内は例示・出力サンプルであり実在性を保証しない前提のため、検査対象から外します。
 * スキル本文は Markdown を含む Markdown（```markdown の中に ```json）を書くため、閉じ判定は
 * CommonMark どおり「情報文字列を持たない同種・同長以上のフェンス行」に限ります。
 */
export function* eachLineOutsideFence(content: string): Generator<SourceLine> {
  const lines = content.split("\n");
  let fence: string | null = null;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const marker = /^\s*(`{3,}|~{3,})(.*)$/.exec(line);

    if (fence) {
      const closes =
        marker !== null &&
        marker[1][0] === fence[0] &&
        marker[1].length >= fence.length &&
        marker[2].trim() === "";
      if (closes) fence = null;
      continue;
    }

    if (marker) {
      fence = marker[1];
      continue;
    }

    yield { line, lineNo: i + 1 };
  }
}

/** 1 行からインラインコードスパン（`...`）の中身を抜き出す。 */
export function extractInlineCode(line: string): string[] {
  const spans: string[] = [];
  const pattern = /(`+)([^`]+?)\1/g;
  let matched: RegExpExecArray | null;

  while ((matched = pattern.exec(line)) !== null) spans.push(matched[2].trim());

  return spans;
}

export type Frontmatter = {
  lines: string[];
  endLine: number;
};

/** 先頭の `---` で囲まれた frontmatter を切り出す。無ければ null。 */
export function splitFrontmatter(content: string): Frontmatter | null {
  const lines = content.split("\n");
  if (lines[0] !== "---") return null;

  for (let i = 1; i < lines.length; i++) {
    if (lines[i] === "---") return { lines: lines.slice(1, i), endLine: i + 1 };
  }

  return null;
}

/**
 * frontmatter のトップレベルキーと値を取り出す。
 *
 * @remarks
 * 折り畳みスカラ（`key: >-`）は後続のインデント行を連結して値とします。YAML パーサを
 * 持ち込まずに済む範囲に限定した簡易解析です。
 */
export function parseFrontmatterKeys(fmLines: readonly string[]): Map<string, string> {
  const keys = new Map<string, string>();

  for (let i = 0; i < fmLines.length; i++) {
    const matched = /^([A-Za-z0-9_-]+):\s*(.*)$/.exec(fmLines[i]);
    if (!matched) continue;

    let value = matched[2].trim();

    if (value === ">-" || value === ">" || value === "|" || value === "|-") {
      const folded: string[] = [];

      for (let j = i + 1; j < fmLines.length; j++) {
        if (fmLines[j].trim() !== "" && !/^\s/.test(fmLines[j])) break;
        folded.push(fmLines[j].trim());
      }

      value = folded.join(" ").trim();
    }

    keys.set(matched[1], value);
  }

  return keys;
}

export type Heading = {
  level: number;
  text: string;
  lineNo: number;
};

/** フェンス外の見出しを (レベル, テキスト) で抽出する。 */
export function extractHeadings(content: string): Heading[] {
  const headings: Heading[] = [];

  for (const { line, lineNo } of eachLineOutsideFence(content)) {
    const matched = /^(#{1,6})\s+(.*?)\s*$/.exec(line);
    if (matched) headings.push({ level: matched[1].length, text: matched[2], lineNo });
  }

  return headings;
}

/** 見出し構造のずれ。`canonical` / `translation` は該当位置の見出し（無ければ `null`）。 */
export type HeadingMismatch = {
  index: number;
  canonical: Heading | null;
  translation: Heading | null;
};

/**
 * 対訳が canonical と同じ見出し構造を持つか調べ、最初のずれを返す。
 *
 * @remarks
 * ファイルの有無だけでは節の欠落・ずれを検出できないため、見出しレベル列の一致まで見ます。
 * 見出しテキストは翻訳で変わるので、比較するのはレベルと出現順だけです。
 */
export function compareHeadingStructure(
  canonical: readonly Heading[],
  translation: readonly Heading[],
): HeadingMismatch | null {
  const max = Math.max(canonical.length, translation.length);

  for (let i = 0; i < max; i++) {
    const en = canonical[i] ?? null;
    const ja = translation[i] ?? null;

    if (en && ja && en.level === ja.level) continue;

    return { index: i, canonical: en, translation: ja };
  }

  return null;
}

/** 対訳の冒頭に canonical を指す翻訳注記（引用行）があるか。 */
export function hasTranslationNote(content: string, canonicalBasename: string): boolean {
  const firstLine = content.split("\n").find((line) => line.trim() !== "") ?? "";

  return firstLine.startsWith(">") && firstLine.includes(canonicalBasename);
}

/** 集合差から片側のみの名前を取り出す。 */
export function onlyIn(names: readonly string[], otherNames: readonly string[]): string[] {
  const other = new Set(otherNames);

  return names.filter((name) => !other.has(name));
}

/** Makefile から集めたターゲット名の索引。`patterns` は `%` を含むパターンルール。 */
export type MakeTargetIndex = {
  exact: Set<string>;
  patterns: RegExp[];
};

/**
 * `.mk` / `makefile` の内容からターゲット名を集める。
 *
 * @remarks
 * `%` を含むパターンルール（`db-migrate-up-%` など）は正規表現として保持します。
 */
export function collectMakeTargets(contents: readonly string[]): MakeTargetIndex {
  const exact = new Set<string>();
  const patterns: RegExp[] = [];

  const addTarget = (name: string): void => {
    if (name === "" || name.startsWith(".")) return;

    if (name.includes("%")) {
      patterns.push(
        new RegExp(
          `^${name
            .split("%")
            .map((part) => part.replace(/[.*+^${}()|[\]\\?]/g, "\\$&"))
            .join(".+")}$`,
        ),
      );
      return;
    }

    exact.add(name);
  };

  for (const content of contents) {
    for (const line of content.split("\n")) {
      if (line.startsWith("\t")) continue;

      const phony = /^\.PHONY:\s*(.+)$/.exec(line);
      if (phony) {
        for (const name of phony[1].split("##")[0].trim().split(/\s+/)) addTarget(name);
        continue;
      }

      const rule = /^([A-Za-z0-9_%.+/ -]+):(?!=)/.exec(line);
      if (rule) {
        for (const name of rule[1].trim().split(/\s+/)) addTarget(name);
      }
    }
  }

  return { exact, patterns };
}

/**
 * 参照された make ターゲットが実在するか判定する。
 *
 * @remarks
 * 参照側がプレースホルダ（`gen-*-oapi` / `new-migrate-<name>`）の場合は、それに当てはまる
 * 実ターゲットが 1 つでもあれば実在と見なします。
 */
export function makeTargetExists(target: string, index: MakeTargetIndex): boolean {
  return expandBraces(target).every((candidate) => {
    if (index.exact.has(candidate)) return true;
    if (index.patterns.some((pattern) => pattern.test(candidate))) return true;
    if (!WILDCARD_RE.test(candidate)) return false;

    const pattern = placeholderToRegExp(candidate, { segmentSeparator: false });

    return (
      [...index.exact].some((name) => pattern.test(name)) ||
      index.patterns.some((declared) => pattern.test(declared.source.replace(/[$^\\]/g, "")))
    );
  });
}

/**
 * インラインコードの `make ...` からターゲット名を取り出す。
 *
 * @remarks
 * 変数代入（`DB=local`）やシェル演算子（`2>&1` / `|`）以降は make の引数ではないため打ち切ります。
 */
export function extractMakeTargets(span: string): string[] {
  if (!/^make(\s|$)/.test(span)) return [];

  const targets: string[] = [];

  for (const token of span.split(/\s+/).slice(1)) {
    if (token.startsWith("-")) continue;
    if (!/^[A-Za-z0-9_%.<>{},*/-]+$/.test(token)) break;
    targets.push(token);
  }

  return targets;
}

/**
 * インラインコードが検査可能なパス参照かどうかを判定する。
 *
 * @remarks
 * 相対ファイル名（`SKILL.md` など文脈依存の記述）は解決先が一意に決まらないため対象外にし、
 * 先頭セグメントが実在するルート直下エントリであるものだけを検査します。さらに、パスと同形だが
 * 実体がファイルではない記述を次の規則で除外します。
 *
 * - 末尾セグメントに `.` も末尾 `/` も無いもの — Go の import パス（`database/sql`）と区別できない
 * - `...` を含むもの — 「以下同様」を表す省略記法
 * - `pkg/ptr.Copy` 形式 — パッケージパス + Go シンボル
 *
 * @param rootEntries - リポジトリルート直下に実在するエントリ名。
 * @param exists - リポジトリルート相対のパスが実在するかを返す関数。
 */
export function asRepoPath(
  span: string,
  rootEntries: ReadonlySet<string>,
  exists: (relativePath: string) => boolean,
): string | null {
  let text = span.trim();

  if (text.startsWith("./")) text = text.slice(2);
  if (!text.includes("/")) return null;
  if (/[\s$\\#?!"'()|`:;@]/.test(text)) return null;
  if (text.includes("...")) return null;

  const isDirRef = text.endsWith("/");
  if (isDirRef) text = text.slice(0, -1);

  if (!rootEntries.has(text.split("/")[0])) return null;
  if (!isDirRef && !path.basename(text).includes(".")) return null;

  const symbol = /^(.*)\.[A-Z][A-Za-z0-9_]*$/.exec(text);
  if (symbol && exists(symbol[1])) return null;

  return text;
}
