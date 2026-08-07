/** OpenAPI の `info.version` 行。桁は 2 で固定（`info:` 直下）。 */
const VERSION_LINE = /^ {2}version: .*$/m;

const SEMVER = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

/**
 * ref 名から刻むべき `info.version` を導く。
 *
 * @remarks
 * `release/vX.Y.Z` だけを対象にします。SemVer として読めない ref は「刻まない」であって
 * 「エラー」ではないため、`null` を返して呼び出し元に no-op を選ばせます。
 */
export function deriveVersion(ref: string): string | null {
  const matched = /^release\/v(.+)$/.exec(ref);

  if (!matched || !SEMVER.test(matched[1])) {
    return null;
  }

  return matched[1];
}

/**
 * spec 本文から現在の `info.version` を読む。
 *
 * @returns 該当行が無ければ `null`。
 */
export function readVersion(content: string): string | null {
  const line = VERSION_LINE.exec(content);

  return line ? line[0].replace(/^ {2}version: /, "") : null;
}

/**
 * spec 本文の `info.version` を差し替える。
 *
 * @remarks
 * 置換値は関数で渡します。バージョン文字列に `$&` のような置換パターンが含まれていても、
 * 文字列としてそのまま書き込むためです。
 */
export function replaceVersion(content: string, version: string): string {
  return content.replace(VERSION_LINE, () => `  version: ${version}`);
}

/**
 * ref と現在のファイル内容から、書き換えるべきかどうかを決める。
 *
 * @remarks
 * 「対象外の ref だから何もしない」と「既に同じ版だから何もしない」と「version 行が無い」は、
 * どれも書き込まない点では同じですが、最後だけは失敗です。入口で分岐を書くと、この 3 つを
 * 取り違えても誰も気づけません。
 *
 * 内容は `readContent` から遅延で受け取ります。対象外の ref では spec を読まずに終える必要があり
 * （spec が無い文脈からも呼ばれる）、その順序は呼び出し側の書き方ではなくここが持ちます。
 */
export type StampPlan =
  | { kind: "skip"; ref: string }
  | { kind: "unchanged"; version: string }
  | { kind: "missing" }
  | { kind: "write"; from: string; to: string; content: string };

export function planStamp(ref: string, readContent: () => string): StampPlan {
  const version = deriveVersion(ref);

  if (version === null) {
    return { kind: "skip", ref };
  }

  const content = readContent();
  const current = readVersion(content);

  if (current === null) {
    return { kind: "missing" };
  }

  if (current === version) {
    return { kind: "unchanged", version };
  }

  return { kind: "write", from: current, to: version, content: replaceVersion(content, version) };
}
