// 撤去の計画を組み立てる（純関数）。書き込みは index.ts、規則は doc-language.ts を参照。
//
// 計画と実行を分けるのは、この撤去が 400 件超のファイルを消したり改名したりするためである。
// 途中で「宣言の無い散文」に当たったら 1 件も書かずに止めたい。読み終える前に書き始めると、
// 半分だけ畳まれたツリーが残り、その状態から復旧する手順は誰も持っていない。

import path from "node:path";

import { keepMarked, stripMarkers } from "../lib/markers";
import {
  type Mode,
  type UndeclaredLine,
  EXCLUDED_PREFIXES,
  isScanTarget,
  listDocPairs,
  describesAPair,
  namesAnyTranslation,
  redactReferences,
  resolveTarget,
  rewriteTranslationLinks,
  stripLeadingTranslationNote,
  transplantFrontmatter,
} from "./doc-language";

/** 本文側で「対訳が在るときだけ生きる記述」を囲むマーカー名。 */
const MARKER = "doc-pair";

/**
 * 「このツールが在るときだけ生きる記述」を囲むマーカー名。
 *
 * @remarks
 * `doc-pair` と分けているのは、落ちる契機が違うからです。`doc-pair` は対訳が消えたときに
 * 落ちるので `both` では残りますが、こちらは言語を選び終えた時点でツールごと消えるため
 * 3 モードとも落ちます。同じ名前空間では表せません —— 手順書や ADR のように、対訳規約の
 * 説明とツールへの言及が同じファイルに同居しているためです。
 */
const TOOL_MARKER = "lang-choice";

/**
 * マーカーを持ち得るファイルの拡張子。
 *
 * @remarks
 * 対訳の存在を前提にしているのは散文だけではありません。対訳ペアを要求する検査（`doc-ref-lint` /
 * `skill-lint`）、ポータルの取り込み表、それを呼ぶ make ターゲットも同じ前提の上に立っています。
 * Markdown だけを歩くと、それらのマーカーが剥がされないまま作成先へ渡り、撤去後に赤くなります。
 */
const MARKED_EXTENSIONS: readonly string[] = [
  ".ts",
  ".tsx",
  ".go",
  ".yaml",
  ".yml",
  ".toml",
  ".mk",
];

/**
 * 拡張子で拾えないが、撤去後に指し先を失う宣言を持つファイル。
 *
 * 無視リストと解析除外は、消えたパスを指したまま残っても止まらない。止まらないからこそ、
 * 誰も直さないまま作成先へ渡る。
 */
const MARKED_FILES: ReadonlySet<string> = new Set([
  ".gitleaksignore",
  ".graphifyignore",
  "sonar-project.properties",
]);

/** Markdown ではないが、マーカーを持ち得る走査対象か。 */
function isMarkedNonMarkdown(file: string): boolean {
  return (
    (MARKED_EXTENSIONS.some((extension) => file.endsWith(extension)) || MARKED_FILES.has(file)) &&
    !EXCLUDED_PREFIXES.some((prefix) => file.startsWith(prefix))
  );
}

/** ファイルを残したまま中身だけ差し替える 1 手。 */
export type WriteOperation = { kind: "write"; path: string; content: string };

/** 撤去で行う 1 手。 */
export type Operation =
  | { kind: "delete"; path: string }
  | WriteOperation
  /** `from` を消して `to` として書き直す（`ja` の改名）。 */
  | { kind: "rename"; from: string; to: string; content: string };

/** 行を残したまま文字列だけ差し替える宣言。`mode` を持つものはそのモードでだけ効く。 */
export type DocReplacement = {
  file: string;
  from: string;
  to: string;
  mode?: Mode;
};

/** 撤去の計画。`undeclared` と `staleReplacements` が空でなければ実行してはならない。 */
export type Plan = {
  operations: Operation[];
  undeclared: UndeclaredLine[];
  /** 本文が動いて当たらなくなった差し替え宣言。 */
  staleReplacements: DocReplacement[];
};

/** 本文の読み出し。存在しなければ `null`。 */
export type ReadFile = (relativePath: string) => string | null;

/**
 * 撤去の計画を組み立てる。
 *
 * @param mode 残す言語。
 * @param files リポジトリが追跡している全ファイル（相対パス）。
 * @param read 本文の読み出し。
 * @param declaredLines 落とすと宣言された行。
 * @param removedPaths 撤去ごと消えるディレクトリ / ファイル（前方一致）。畳む対象から外す。
 * @param allowedMentions 畳んだ後も残してよいと宣言された行（前後の空白を除いた完全一致）。
 * @param replacements 行を残したまま文字列だけ差し替える宣言（表のセルなど）。
 *
 * @remarks
 * 本文側の `doc-pair` マーカー（節ごとの撤去と、言い回しの差し替え）はここで剥がします。
 * マーカーが持つのは「対訳が在ることを前提にした記述」で、宣言と違って本文の隣に住むため、
 * 周りが書き換わっても一緒に動きます。
 *
 * @remarks
 * `en` と `ja` は消すファイルこそ逆ですが、行き着く先はどちらも「`<name>.md` 1 本」です。
 * 違うのは残った本文に対して何が無効になるかで、`en` では `<name>.ja.md` という名前が世界から
 * 消え、`ja` では自分自身を指すリンクだけが無効になります（相手の名前は残り、中身の言語が変わる）。
 */
export function planRemoval(
  mode: Mode,
  files: readonly string[],
  read: ReadFile,
  declaredLines: ReadonlySet<string> = new Set(),
  removedPaths: readonly string[] = [],
  replacements: readonly DocReplacement[] = [],
  allowedMentions: ReadonlySet<string> = new Set(),
): Plan {
  // 撤去ごと消えるものを先に外す。畳んだ後の姿を問うても、そこには何も残らない。
  const markdown = files
    .filter(isScanTarget)
    .filter((file) => !removedPaths.some((prefix) => file === prefix || file.startsWith(`${prefix}/`)));
  const pairs = listDocPairs(markdown);
  const marked = files
    .filter(isMarkedNonMarkdown)
    .filter((file) => !removedPaths.some((prefix) => file === prefix || file.startsWith(`${prefix}/`)))
    .sort((a, b) => a.localeCompare(b));
  const applicable = replacements.filter((entry) => entry.mode === undefined || entry.mode === mode);
  // 空振りの検査は絞る前に行う。モードで効かない宣言も、本文が動けば直す必要がある。
  const stale = replacements.filter(({ file, from }) => !(read(file) ?? "").includes(from));
  const rewrite = replacingReader(read, applicable);

  const plan =
    mode === "en"
      ? planEnglishOnly(markdown, pairs.map((pair) => pair.translation), read, rewrite, declaredLines)
      : planJapaneseOnly(markdown, pairs, read, rewrite, declaredLines);

  const stripped: Operation[] = [];
  const strayMentions: UndeclaredLine[] = [];

  for (const file of marked) {
    const original = read(file);
    const content = rewrite(file);

    if (original === null || content === null) {
      continue;
    }

    // 散文と同じ基準で見張る。ここを見ないと、検査スクリプトや取り込み表に残った言及が
    // 報告もされないまま作成先へ渡り、撤去後に初めて赤くなる。
    content.split("\n").forEach((line, index) => {
      if (namesAnyTranslation(line)) {
        strayMentions.push({ file, line: index + 1, text: line });
      }
    });

    if (content !== original) {
      stripped.push({ kind: "write", path: file, content });
    }
  }

  return {
    ...plan,
    operations: [...plan.operations, ...stripped],
    // 宣言された言及は「判断が要る散文」ではない。本文は触らず、報告からだけ外す。
    undeclared: [...plan.undeclared, ...strayMentions].filter(
      ({ text }) => !allowedMentions.has(text.trim()),
    ),
    staleReplacements: stale,
  };
}

/** ツールの痕跡を落とした計画と、それを解決済みで読む読み出し。 */
export type FootprintPlan = {
  operations: WriteOperation[];
  stale: DocReplacement[];
  read: ReadFile;
};

/**
 * ツールと共に死ぬ記述を落とす計画。3 モードで同じものを返す。
 *
 * @remarks
 * 畳む 2 モードでは `doc-pair` の除去がこれを兼ねられますが、`both` はマーカーの中身を
 * 残す向きに解決するため、ツールへの言及だけが取り残されます。呼べば失敗する make ターゲットが
 * 手順書つきで残る状態になるので、契機の違うこちらを先に解決します。
 *
 * 返す `read` は痕跡を解決済みの本文を返します。後段の計画にこれを渡すのは、`doc-pair` の
 * 走査より前に落としておかないと、撤去されるはずの散文が「宣言の無い散文」として報告され、
 * 撤去そのものが止まるためです。
 */
export function planToolFootprint(
  files: readonly string[],
  read: ReadFile,
  replacements: readonly DocReplacement[] = [],
  removedPaths: readonly string[] = [],
): FootprintPlan {
  const byFile = new Map<string, DocReplacement[]>();

  for (const replacement of replacements) {
    byFile.set(replacement.file, [...(byFile.get(replacement.file) ?? []), replacement]);
  }

  const resolveContent = (relativePath: string, source: string) => {
    const replaced = (byFile.get(relativePath) ?? []).reduce(
      (text, { from, to }) => text.split(from).join(to),
      source,
    );

    return stripMarkers(replaced, TOOL_MARKER).content;
  };

  const resolve: ReadFile = (relativePath) => {
    const source = read(relativePath);

    return source === null ? null : resolveContent(relativePath, source);
  };

  const excluded = (file: string) =>
    removedPaths.some((prefix) => file === prefix || file.startsWith(`${prefix}/`));

  const operations = files
    .filter((file) => isScanTarget(file) || isMarkedNonMarkdown(file) || byFile.has(file))
    .filter((file) => !excluded(file))
    .sort((a, b) => a.localeCompare(b))
    .flatMap<WriteOperation>((file) => {
      const original = read(file);

      if (original === null) {
        return [];
      }

      const content = resolveContent(file, original);

      return content === original ? [] : [{ kind: "write", path: file, content }];
    });

  return {
    operations,
    stale: replacements.filter(({ file, from }) => !(read(file) ?? "").includes(from)),
    read: resolve,
  };
}

/**
 * 両方の言語を残すと決めたときの計画。マーカーの宣言だけを剥がし、本文は残す。
 *
 * @remarks
 * 何もしないのではありません。マーカーは「まだ選べる」という宣言なので、選び終えたツリーに
 * 残すと次に読む人が選択の余地を読み違えます。退避コメントも同じ理由で落とします —— 有効側が
 * 残る以上あれは二度と使われない複製で、残せば読む人が二つの正解を突き合わせる羽目になります。
 *
 * 対訳を運ぶ仕組み（`canonicalize-doc`、ポータルの言語切り替え、日本語ガイドの出力先）は
 * `REMOVED_PATHS` に載っていますが、ここでは消しません。両方を残す選択とは、それらを使い
 * 続けるという意味だからです。畳む 2 モードとの違いはそこにあります。
 *
 * 宣言（`DOC_REPLACEMENTS` / `DECLARED_LINES`）も使いません。あれは消える名前への言及を
 * 直すためのもので、何も消えないこの経路では当たる相手が居ません。
 */
export function planKeepBoth(
  files: readonly string[],
  read: ReadFile,
  removedPaths: readonly string[] = [],
): WriteOperation[] {
  const excluded = (file: string) =>
    removedPaths.some((prefix) => file === prefix || file.startsWith(`${prefix}/`));

  return files
    .filter((file) => isScanTarget(file) || isMarkedNonMarkdown(file))
    .filter((file) => !excluded(file))
    .sort((a, b) => a.localeCompare(b))
    .flatMap((file) => {
      const original = read(file);

      if (!original?.includes(`${MARKER}:`)) {
        return [];
      }

      const content = keepMarked(original, MARKER).content;

      return content === original ? [] : [{ kind: "write" as const, path: file, content }];
    });
}

/**
 * 差し替えを済ませた本文を返す読み出しを組み立てる。
 *
 * @remarks
 * 参照除去より前に差し替えるのは、差し替えの目的がまさに「消える名前への言及を無くすこと」
 * だからです。順序が逆だと、差し替えれば消えるはずの行が先に「宣言の無い散文」として報告されます。
 */
function replacingReader(read: ReadFile, replacements: readonly DocReplacement[]): ReadFile {
  const byFile = new Map<string, DocReplacement[]>();

  for (const replacement of replacements) {
    byFile.set(replacement.file, [...(byFile.get(replacement.file) ?? []), replacement]);
  }

  return (relativePath) => {
    const source = read(relativePath);

    if (source === null) {
      return null;
    }

    const replaced = (byFile.get(relativePath) ?? []).reduce(
      (text, { from, to }) => text.split(from).join(to),
      source,
    );

    // マーカーは差し替えの後に剥がす。差し替えの完全一致はマーカーの外側の本文に当たるため。
    return stripMarkers(replaced, MARKER).content;
  };
}

function planEnglishOnly(
  markdown: readonly string[],
  translations: readonly string[],
  read: ReadFile,
  rewrite: ReadFile,
  declaredLines: ReadonlySet<string>,
): Omit<Plan, "staleReplacements"> {
  const removed = new Set(translations);
  const operations: Operation[] = translations.map((path) => ({ kind: "delete", path }));
  const undeclared: UndeclaredLine[] = [];

  for (const file of markdown.filter((file) => !removed.has(file))) {
    const original = read(file);
    const source = rewrite(file);

    if (original === null || source === null) {
      continue;
    }

    const result = redactReferences(
      source,
      file,
      pointsInto(file, removed),
      declaredLines,
      // 実在パスに解決できる参照だけでは足りない。`*.ja.md` のようなグロブや綴りだけの言及も
      // 宛先を失うが、解決できないので撤去対象の集合には当たらない。接尾辞で見る。
      namesAnyTranslation,
    );

    undeclared.push(...result.undeclared);

    // 比べる相手はディスク上の本文。差し替えだけが要るファイルはここでしか拾えない。
    if (result.content !== original) {
      operations.push({ kind: "write", path: file, content: result.content });
    }
  }

  return { operations, undeclared };
}

function planJapaneseOnly(
  markdown: readonly string[],
  pairs: readonly { canonical: string; translation: string }[],
  read: ReadFile,
  rewrite: ReadFile,
  declaredLines: ReadonlySet<string>,
): Omit<Plan, "staleReplacements"> {
  const operations: Operation[] = [];
  const undeclared: UndeclaredLine[] = [];
  const paired = new Set(pairs.flatMap(({ canonical, translation }) => [canonical, translation]));

  for (const { canonical, translation } of pairs) {
    const source = rewrite(translation);

    if (source === null) {
      continue;
    }

    // 移植元も差し替え済みの本文にする。生のまま移すと、正本の frontmatter が語る対訳が
    // 改名後の正本へそのまま乗る。YAML のブロックスカラーはマーカーを置けないので、
    // そこを直せるのは宣言だけである。
    const withFrontmatter = transplantFrontmatter(rewrite(canonical) ?? "", source);
    // 冒頭の翻訳注記は位置が規約で決まっているので、参照の検査に載せず先に落とす。
    // 参照だけを見ると、`SKILL.md` という最頻出の名前に触れた散文まで巻き込みます。
    const withoutNote = stripLeadingTranslationNote(withFrontmatter, path.posix.basename(canonical));
    const result = redactReferences(
      withoutNote,
      canonical,
      pointsInto(translation, new Set([canonical])),
      declaredLines,
      describesAPair,
    );

    undeclared.push(...result.undeclared);
    operations.push({
      kind: "rename",
      from: translation,
      to: canonical,
      // 相手の名前へ寄せるのは最後。先に済ませると、対訳への言及と自己参照が見分けられなくなる。
      content: rewriteTranslationLinks(result.content),
    });
  }

  // ペアを持たない文書も同じ基準で見る。対訳規約を語るのは対訳を持つ文書だけではなく、
  // 規約を運用する側の指示（検査エージェントの除外指定など）も同じ前提の上に立っている。
  // ペアだけを歩くと、そこに残った `*.ja.md` は報告もされないまま作成先へ渡る。
  for (const file of markdown.filter((file) => !paired.has(file))) {
    const original = read(file);
    const source = rewrite(file);

    if (original === null || source === null) {
      continue;
    }

    const result = redactReferences(source, file, () => false, declaredLines, describesAPair);

    undeclared.push(...result.undeclared);

    const content = rewriteTranslationLinks(result.content);

    if (content !== original) {
      operations.push({ kind: "write", path: file, content });
    }
  }

  return { operations, undeclared };
}

/** `file` から見た参照が `removed` のどれかを指すかを答える述語を返す。 */
function pointsInto(file: string, removed: ReadonlySet<string>): (target: string) => boolean {
  const fromDir = path.posix.dirname(file);

  return (target) => {
    const resolved = resolveTarget(target, fromDir);

    return resolved !== null && removed.has(resolved);
  };
}
