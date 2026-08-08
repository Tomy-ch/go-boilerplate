// 撤去の純粋ロジック。ファイル I/O と git は index.ts / git-commit.ts が持つ。
//
// lockfile と egress の孤児判定は製品名を持たず「参照数を数える」1 本に寄せてある。README の
// 書き換えは完全一致に限り、一致しなければ投げる。どちらも「消したつもりで消えていない」を
// 静かに作らないための選択である。

/** 撤去後に残るファイルの本文と照合するための入力。 */
export type PinReferenceInput = {
  key: string;
  survivingContents: readonly string[];
};

/**
 * lockfile キーから `owner/repo` を取り出す。
 *
 * `uses:` はサブパス付き（`snyk/actions/setup`）でも書けるため、参照の有無はタグを外した
 * `owner/repo` で数える。
 */
export function repoOf(key: string): string {
  const at = key.lastIndexOf("@");

  return at === -1 ? key : key.slice(0, at);
}

/** 撤去後に残る本文のどれかが当該 action を参照していれば true。 */
export function isPinKeyReferenced({ key, survivingContents }: PinReferenceInput): boolean {
  const repo = repoOf(key);

  return survivingContents.some((content) => content.includes(`uses: ${repo}`));
}

/** lockfile から指定キーの行を落とす。 */
export function removePinEntries(content: string, keys: readonly string[]): string {
  if (keys.length === 0) {
    return content;
  }

  const targets = new Set(keys);

  return content
    .split("\n")
    .filter((line) => {
      const match = /^"([^"]+)"\s*=/.exec(line);

      return match === null || !targets.has(match[1]);
    })
    .join("\n");
}

/** egress SSOT から `[job."<key>"]` セクションを、次のセクション見出しまで本文ごと落とす。 */
export function removeEgressSections(content: string, jobKeys: readonly string[]): string {
  if (jobKeys.length === 0) {
    return content;
  }

  const targets = new Set(jobKeys.map((key) => `[job."${key}"]`));
  const kept: string[] = [];
  let dropping = false;

  for (const line of content.split("\n")) {
    if (/^\[(class|job)\./.test(line)) {
      dropping = targets.has(line.trim());
    }

    if (!dropping) {
      kept.push(line);
    }
  }

  return kept.join("\n");
}

/** 宣言した文字列が本文に無い（README が動いた）ことを表す。 */
export class MissingDeclarationError extends Error {
  constructor(file: string, kind: string, needle: string) {
    super(`${file}: 宣言した${kind}が見つかりません（README が動いた可能性があります）: ${needle}`);
    this.name = "MissingDeclarationError";
  }
}

/** 宣言した見出しが Markdown の見出しの形をしていないことを表す。 */
export class MalformedHeadingError extends Error {
  constructor(file: string, heading: string) {
    super(`${file}: 宣言した見出しが Markdown の見出しではありません: ${heading}`);
    this.name = "MalformedHeadingError";
  }
}

/**
 * 完全一致した最初の 1 箇所を取り除く。
 *
 * @throws {MissingDeclarationError} 一致が無い場合。
 */
export function removeExact(content: string, needle: string, file: string, kind: string): string {
  const at = content.indexOf(needle);

  if (at === -1) {
    throw new MissingDeclarationError(file, kind, needle.trim().slice(0, 80));
  }

  return content.slice(0, at) + content.slice(at + needle.length);
}

/**
 * 見出しと本文を、次の同レベル以上の見出しの直前まで落とす。
 *
 * 見出しの手前の空行は残す。それは直前の要素との区切りであって消える節の持ち物ではなく、
 * 巻き込むと隣り合う 2 節を続けて消したときに区切りが尽きて markdownlint の MD022 を割る。
 *
 * 見出しの形をしていない文字列は、本文に同じ行があっても受け付けない。level を 0 に落として
 * 続けると、どの見出しも境界と見なされず宣言した行から本文の終端までを黙って落とすためである。
 *
 * @throws {MalformedHeadingError} 宣言が見出しの形をしていない場合。
 * @throws {MissingDeclarationError} 見出しが無い場合。
 */
export function removeSection(content: string, heading: string, file: string): string {
  const level = /^(#+)\s/.exec(heading)?.[1].length;

  if (level === undefined) {
    throw new MalformedHeadingError(file, heading);
  }

  const lines = content.split("\n");
  const start = lines.findIndex((line) => line === heading);

  if (start === -1) {
    throw new MissingDeclarationError(file, "見出し", heading);
  }

  let end = lines.length;

  for (let i = start + 1; i < lines.length; i += 1) {
    const match = /^(#+)\s/.exec(lines[i]);

    if (match !== null && match[1].length <= level) {
      end = i;
      break;
    }
  }

  return [...lines.slice(0, start), ...lines.slice(end)].join("\n");
}
