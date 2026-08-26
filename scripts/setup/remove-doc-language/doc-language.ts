// 対訳ペアを片方の言語へ畳む規則（純関数）。対象の宣言は language-manifest.ts、
// ファイル操作とコミットは index.ts を参照。
//
// `en` と `ja` はファイル操作こそ逆向きだが、行き着く先は同じである。どちらを選んでも対訳ペアは
// 消え、残るのは `<name>.md` 1 本になる（`ja` は `<name>.ja.md` を `<name>.md` へ改名して正本に
// する）。だから「対訳が在ること」を前提にした検査と散文の撤去は、モードによらず 1 種類で足りる。
//
// 残った本文から消えた側への参照を落とすとき、機械的に触るのは Markdown リンクだけに限る。
// コードスパンでの言及（`SKILL.ja.md` など）は対訳規約そのものを説明している散文であることが
// 多く、参照だけ剥がしても文が成立しない。判断が要る行は黙って書き換えず、宣言が無ければ報告する。

import path from "node:path";

/** 撤去後に残す言語。`both` は入口で弾くのでここには来ない。 */
export type Mode = "en" | "ja";

/** 対訳ペア。パスはリポジトリルートからの相対。 */
export type DocPair = {
  /** 英語正本 `<name>.md`。実在するとは限らない（孤児の対訳がある）。 */
  canonical: string;
  /** 日本語訳 `<name>.ja.md`。 */
  translation: string;
};

/** 機械的に判断できず、宣言も無かった行。 */
export type UndeclaredLine = {
  file: string;
  line: number;
  text: string;
};

/** 参照除去の結果。 */
export type Redaction = {
  content: string;
  undeclared: UndeclaredLine[];
};

/**
 * 走査から外すディレクトリ。生成物と依存物は撤去の対象ではない。
 *
 * @remarks
 * `docs/portal/guides/**` は manifest から作られる複製なので、消すのではなく作られなくします。
 */
export const EXCLUDED_PREFIXES: readonly string[] = [
  "vendor/",
  "node_modules/",
  ".git/",
  "tmp/",
  "graphify-out/",
  "docs/portal/guides/",
  "docs/coverage/",
  "docs/db-schema/",
  "docs/godoc/",
];

const TRANSLATION_SUFFIX = ".ja.md";

/**
 * 参照が剥がれた跡に残ってよい言語ラベル。
 *
 * 長い順に並べる。`English canonical` を先に消さないと `English` だけが落ちて `canonical` が残る。
 */
const LABEL_PHRASES: readonly string[] = [
  "English canonical",
  "English",
  "Japanese",
  "日本語版",
  "日本語",
  "英語版",
];

/**
 * 対訳の存在を述べる注記に必ず現れる語。
 *
 * 相手への参照と同じ行に在るときだけ注記と見なす（{@link isTranslationNotice}）。
 */
const NOTICE_PHRASES: readonly string[] = [
  "日本語訳",
  "参考訳",
  "対訳",
  "翻訳",
  "同期",
  "sibling",
  "日本語参考訳",
  "英語正本",
  "英語正典",
  "英語 canonical",
  "英語版",
  "canonical",
  "Japanese reference translation",
  "Japanese translation",
  "Japanese version",
  "translation in sync",
  "re-sync",
];

/** 飾りとして扱う記号。言語ラベルを外した残りがこれだけなら、その行は参照のためだけに在った。 */
const DECORATION_CHARS = /[|>\-*:：、。\s]/g;

/**
 * Markdown のインラインリンク。括弧の中は丸ごと取る。
 *
 * @remarks
 * タイトル付き（`[t](p "title")`）を任意の後続要素として書くと、行き先の量指定子と
 * 取り合いになって後戻りが生じます。中身を 1 つの塊で取り、行き先は
 * {@link hrefOf} が切り出します。
 */
const INLINE_LINK = /\[[^\]]*\]\(([^)]*)\)/g;

/** リンクの括弧の中から行き先だけを取る。タイトルは空白で区切られた後半。 */
function hrefOf(inside: string): string {
  const [href = ""] = inside.trim().split(/\s/);

  return href;
}

/**
 * 参照形式のリンク定義（`[ラベル]: 行き先`）。
 *
 * インラインリンクと同じく参照であって言及ではありません。別に見るのは、書式が違うので
 * {@link INLINE_LINK} に当たらないためです。ADR は相互参照をこの形で末尾にまとめており、
 * 見落とすと 1 ファイルあたり 7 件の「宣言の無い言及」として積み上がります。
 */
const REFERENCE_DEFINITION = /^\s*\[[^\]]+\]:\s*(\S+)/;

/** リンク以外の形で `.md` に触れている箇所（コードスパン・素のパス）。 */
const BARE_MENTION = /[^\s()[\]"'`|]{1,256}\.md/g;

/** 対訳を名指す綴り。`*.ja.md` のようなグロブは、畳んだ後に当たるファイルを 1 つも持たない。 */
const TRANSLATION_GLOB = /[*?]/;

/**
 * 訳文の冒頭に置かれた翻訳注記（引用行）を落とす。
 *
 * @remarks
 * 「このファイルは `X` の日本語訳です」で始まるのはこのリポジトリの規約で、`skill-lint` が
 * 実際に検査しています。改名して正本にした後もこれが残ると、正本が自分を訳だと名乗ります。
 *
 * 冒頭の引用ブロックに限るのは、この位置が規約で決まっているからです。本文中の引用まで
 * 対象にすると、正本の名前に触れただけの引用を巻き込みます。
 */
export function stripLeadingTranslationNote(content: string, canonicalBasename: string): string {
  const lines = content.split("\n");
  let head = 0;

  if (lines[0] === "---") {
    const close = lines.indexOf("---", 1);

    if (close < 0) {
      return content;
    }

    head = close + 1;
  }

  let start = head;
  while (start < lines.length && lines[start].trim() === "") start++;

  let end = start;
  while (end < lines.length && lines[end].trimStart().startsWith(">")) end++;

  const note = lines.slice(start, end).join("\n");

  if (end === start || !note.includes(canonicalBasename)) {
    return content;
  }

  while (end < lines.length && lines[end].trim() === "") end++;

  return [...lines.slice(0, start), ...lines.slice(end)].join("\n");
}

/**
 * リンクを除いた行を見て、消える対訳に触れているかを答える述語。
 *
 * @see {@link namesAnyTranslation} / {@link describesAPair}
 */
export type MentionProbe = (lineWithoutLinks: string) => boolean;

/** 対訳を名指す綴りをすべて拾う（`en` 用。`X.ja.md` という名前は 1 つ残らず消える）。 */
export const namesAnyTranslation: MentionProbe = (line) =>
  [...line.matchAll(BARE_MENTION)].some(([target]) => target.endsWith(TRANSLATION_SUFFIX));

/**
 * 対訳ペアそのものを説明している行か（`ja` 用）。
 *
 * @remarks
 * `ja` では対訳が正本の名前へ改名されるので、他の文書への `other.ja.md` という言及は
 * 書き換えで生き残ります。生き残らないのは、同じ行が正本と対訳を並べている場合
 * （`env/README.md`, `env/README.ja.md` —— 畳めば同じ 1 ファイルを 2 回挙げることになる）と、
 * `*.ja.md` のように当たるファイルが 1 つも無くなるグロブです。
 */
export const describesAPair: MentionProbe = (line) => {
  const targets = [...line.matchAll(BARE_MENTION)].map(([target]) => target);
  const translations = targets.filter((target) => target.endsWith(TRANSLATION_SUFFIX));

  return translations.some(
    (target) => TRANSLATION_GLOB.test(target) || targets.includes(canonicalOf(target)),
  );
};

/** リポジトリ相対パスが走査対象か。 */
export function isScanTarget(relativePath: string): boolean {
  return (
    relativePath.endsWith(".md") &&
    !EXCLUDED_PREFIXES.some((prefix) => relativePath.startsWith(prefix))
  );
}

/** `<name>.ja.md` に対応する正本パス `<name>.md`。 */
export function canonicalOf(translation: string): string {
  return translation.endsWith(TRANSLATION_SUFFIX)
    ? `${translation.slice(0, -TRANSLATION_SUFFIX.length)}.md`
    : translation;
}

/**
 * `.ja.md` を起点に対訳ペアを組み立てる。
 *
 * @remarks
 * 正本が実在しない孤児の `.ja.md` も 1 件として返します。`en` では消し、`ja` では改名するだけで、
 * 正本の有無は結果を変えないためです。並びを固定するのは、撤去の出力とコミットの中身が
 * 走査順（ファイルシステムの都合）で揺れないようにするためです。
 */
export function listDocPairs(files: readonly string[]): DocPair[] {
  return files
    .filter((file) => file.endsWith(TRANSLATION_SUFFIX) && isScanTarget(file))
    .sort((a, b) => a.localeCompare(b))
    .map((translation) => ({ canonical: canonicalOf(translation), translation }));
}

/**
 * 本文中の `.ja.md` 参照を `.md` へ書き換える（`ja` 用）。
 *
 * @remarks
 * 改名後は対訳がすべて正本の名前になるため、対訳どうしのリンクはこの置換だけで生き続けます。
 * 置換の結果として自分自身を指すようになったリンクは {@link redactReferences} が別に落とします。
 */
export function rewriteTranslationLinks(content: string): string {
  return content.replace(/([^\s()[\]"'`|]{1,256})\.ja\.md/g, "$1.md");
}

/**
 * 正本のフロントマターを訳文へ移植する（`ja` 用）。訳文が既に持っていればそちらを正とする。
 *
 * @remarks
 * スキルのフロントマター（`name` / `description` / `allowed-tools`）は機械が読む定義で、対訳は
 * 本文しか持ちません。移植しないまま改名すると、日本語を選んだだけでスキルが 1 本残らず
 * 読み込まれなくなります。この差はリポジトリ全体で 84 スキル分あります。
 */
export function transplantFrontmatter(canonical: string, translation: string): string {
  const block = frontmatterOf(canonical);

  return block === null || frontmatterOf(translation) !== null
    ? translation
    : `${block}\n${translation.replace(/^\n+/, "")}`;
}

function frontmatterOf(content: string): string | null {
  return /^---\n[\s\S]*?\n---\n/.exec(content)?.[0] ?? null;
}

/**
 * 残る本文から、消えた側への参照を落とす。
 *
 * @param content 撤去後に残る本文。
 * @param file `content` の撤去後のパス（報告に使う）。
 * @param isRemoved リンク先が消えた側を指すかを答える述語。
 * @param declaredLines 落とすと宣言された行（前後の空白を除いた完全一致）。
 * @param namesAVanishedThing リンクを除いた行を見て、消える対訳に触れているかを答える述語。
 *   既定は「触れていない」。
 *
 * @remarks
 * リンクを剥がした残りが言語ラベルと区切り記号だけなら、その行は参照のためだけに在ったと見なして
 * 行ごと落とします。そうでない行は散文なので、宣言が無ければ書き換えず
 * {@link Redaction.undeclared} へ積みます。黙って剥がすと、規約を説明していた文が意味の通らない
 * 残骸になって残り、しかもそれは撤去の直後には誰も読み返しません。
 *
 * リンクになっていない言及を拾うかどうかは `namesAVanishedThing` が決めます。行単位で問うのは、
 * 同じ綴りが文脈によって嘘にも真にもなるためです。`ja` では `other.ja.md` は改名後の名前へ
 * 書き換えられて生き残りますが、同じ行が `other.md` と `other.ja.md` を並べていれば、それは
 * 対訳ペアを説明している行であり、畳んだ後には片方しか無くなります。
 */
export function redactReferences(
  content: string,
  file: string,
  isRemoved: (target: string) => boolean,
  declaredLines: ReadonlySet<string>,
  namesAVanishedThing: MentionProbe = () => false,
): Redaction {
  const undeclared: UndeclaredLine[] = [];
  const kept: string[] = [];
  let cutJustBefore = false;

  const cut = (): void => {
    cutJustBefore = true;
  };
  const keep = (line: string): void => {
    if (cutJustBefore && line.trim() === "" && (kept.at(-1) ?? "").trim() === "") {
      return;
    }

    kept.push(line);
    cutJustBefore = false;
  };

  content.split("\n").forEach((line, index) => {
    if (declaredLines.has(line.trim())) {
      cut();
      return;
    }

    const definition = REFERENCE_DEFINITION.exec(line);

    if (definition !== null) {
      // 定義行は行き先そのものなので、指す先が消えるなら行ごと落とす。残るなら手を触れない
      //（`.ja.md` の綴りは、この後の書き換えが正本の名前へ寄せる）。
      if (isRemoved(definition[1])) {
        cut();
      } else {
        keep(line);
      }

      return;
    }

    const stripped = line.replace(INLINE_LINK, (whole, inside: string) =>
      isRemoved(hrefOf(inside)) ? "" : whole,
    );

    if (stripped === line && !namesAVanishedThing(line.replace(INLINE_LINK, ""))) {
      keep(line);
      return;
    }

    if (stripped !== line && isNavigationOnly(stripped)) {
      cut();
      return;
    }

    if (isTranslationNotice(line)) {
      cut();
      return;
    }

    undeclared.push({ file, line: index + 1, text: line });
    keep(line);
  });

  return { content: kept.join("\n"), undeclared };
}

/**
 * 対訳の存在そのものを述べている注記か。
 *
 * @remarks
 * 「これは `X` の日本語訳です」「A Japanese reference translation is available at `X`」の類は、
 * 対訳が消えた後には述べる相手がいません。参照だけ剥がすと主語のない文が残るので、行ごと落とします。
 *
 * 判定を語彙に委ねているのは、この注記が 30 通り以上の文面で書かれているためです。文面を
 * 1 つずつ宣言すると、宣言のほうが本文より先に腐ります。片方（相手への参照）だけでは
 * 普通の相互参照と区別が付かないので、両方が同じ行に在ることを条件にします。
 *
 * 語彙は両言語で対称に保ちます。英語側の "translation" に当たる語を日本語側に持たないと、
 * `en` では静かに畳めた文が `ja` では 100 件超の「判断が要る散文」として積み上がります。
 */
export function isTranslationNotice(line: string): boolean {
  return NOTICE_PHRASES.some((phrase) => line.includes(phrase));
}

/**
 * 行き先を並べるだけの行か（リンクと飾りしか無いか）。
 *
 * @remarks
 * `[Controller README](...) | 日本語: [rest.ja.md](rest.ja.md)` のような導入行は、対訳への
 * 行き先が消えれば行として役目を終えます。生き残る行き先だけを残す繕い方もありますが、
 * 宙に浮いた 1 本のリンクが残るだけなので行ごと落とします。
 *
 * 散文には語があるのでここには当たりません。当たるのはリンク・区切り記号・言語ラベルだけで
 * 出来た行に限られます。
 */
export function isNavigationOnly(line: string): boolean {
  return isDecorationOnly(line.replace(INLINE_LINK, ""));
}

/** 言語ラベルと区切り記号しか残っていないか。 */
export function isDecorationOnly(line: string): boolean {
  const withoutLabels = LABEL_PHRASES.reduce((text, phrase) => text.split(phrase).join(""), line);

  return withoutLabels.replace(DECORATION_CHARS, "") === "";
}

/**
 * 参照が指すリポジトリ相対パスを解く。解けない形（外部 URL・散文中の略記）は `null`。
 *
 * @remarks
 * basename で突き合わせてはいけません。`SKILL.md` と `README.md` はこのリポジトリで最も多い
 * ファイル名で、他ディレクトリの同名ファイルへの言及まで自分自身への参照として拾います。
 * ja では `scaffold-test/SKILL.md` のような他スキルへの言及が 300 件近く巻き込まれました。
 */
export function resolveTarget(target: string, fromDir: string): string | null {
  const bare = target.replace(/[#?].*$/, "");

  if (bare === "" || /^[a-z][a-z0-9+.-]{0,32}:/i.test(bare)) {
    return null;
  }

  return path.posix.normalize(path.posix.join(fromDir, bare));
}
