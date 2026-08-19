// サンプルAPI削除に固有の判定。削除対象の宣言（SAMPLE_DOMAINS / MARKER_LITERAL_FILES /
// BUILD_STEPS）は sample-manifest.ts、マーカー除去の機構は ../lib/markers.ts を参照。

import { type StripResult, stripMarkers } from "../lib/markers";
import { EXCLUDED_PATH_PREFIXES, MARKER_LITERAL_FILES, SAMPLE_DOMAINS } from "./sample-manifest";

/** サンプルAPI の在否で行を切り替えるマーカー名。 */
export const SAMPLE_MARKER = "sample-api";

/** サンプルAPI のマーカー行を除去する。 */
export function stripSampleMarkers(content: string): StripResult {
  return stripMarkers(content, SAMPLE_MARKER);
}

/** 削除登録されたパスの配下か。丸ごと消えるものを書き換えても意味が無い。 */
function isRegisteredForDeletion(normalizedPath: string): boolean {
  return Object.values(SAMPLE_DOMAINS)
    .flatMap((domain) => domain.paths)
    .some((registered) => normalizedPath === registered || normalizedPath.startsWith(`${registered}/`));
}

/**
 * 走査対象か。ディレクトリ名の除外は列挙側が行うため、ここは接頭辞・削除登録・literal 宣言を見る。
 *
 * @remarks
 * 削除登録されたパスを外すのは、`dryRun` と本番で結果を揃えるためです。本番では削除が先に走れば
 * 走査から自然に消えますが、`dryRun` では何も消えないので、外さないと「本番では起きない失敗」を
 * 予行演習だけが報告することになります。削除ツール自身のディレクトリがまさにそれで、マーカーの
 * 形を持つフィクスチャを抱えています。
 *
 * remove-boilerplate-identity 側にも同型の判定があります。共通化していない理由は
 * `sample-manifest.ts` の `EXCLUDED_DIRECTORIES` に書いたとおりで、2 つの撤去の起爆順序が
 * 決まっていないためです。
 */
export function isScanTarget(relativePath: string): boolean {
  const normalized = relativePath.replaceAll("\\", "/");

  if (MARKER_LITERAL_FILES.includes(normalized) || isRegisteredForDeletion(normalized)) {
    return false;
  }

  return !EXCLUDED_PATH_PREFIXES.some((prefix) => normalized.startsWith(prefix));
}

/** `<comment> sample-api:` を含むか。 */
export function containsSampleMarker(content: string): boolean {
  return new RegExp(String.raw`(?:\/\/|#|<!--)\s*${SAMPLE_MARKER}:`).test(content);
}

/**
 * 削除対象が ROOT_DIR の内側に収まっているかを判定する。
 *
 * @remarks
 * manifest への追記ミス（`..` を含むパス・空文字・絶対パス）で ROOT_DIR の外や ROOT_DIR 自体を
 * 消してしまわないための安全策です。dry-run でも必ず通し、削除の前に検証します。
 * 区切り文字を足してから前方一致を見るのは、`/repo` に対する `/repo-backup` のような
 * 「接頭辞は一致するが別のディレクトリ」を内側と誤判定しないためです。
 */
export function isWithinRoot(absolutePath: string, rootDir: string, separator: string): boolean {
  const rootWithSeparator = rootDir.endsWith(separator) ? rootDir : rootDir + separator;

  return absolutePath !== rootDir && absolutePath.startsWith(rootWithSeparator);
}

/** 初期化ツールの検証器。まだ残っていれば `setup/lib` を使い続けている。 */
export const SETUP_VERIFIER_DIR = "verify-setup";

/** `setup/` 配下の共有モジュール。使う側が全て消えたときだけ道連れにする。 */
export const SETUP_SHARED_DIR = "lib";

/**
 * `setup/lib` を使い続ける、サンプル削除以外のツール（`setup/` からの相対）。
 *
 * @remarks
 * どれも独立した任意手順で、実行順は利用者が決めます。1 つでも残っているうちに `lib` を
 * 消すと、まだ実行していない手順が実行できなくなるため、在否を見てから判断します。
 */
export const SETUP_SHARED_DIR_USERS: readonly string[] = [SETUP_VERIFIER_DIR];

/**
 * サンプル削除ツール自身の撤去に、共有モジュールを含めるか。
 *
 * @remarks
 * `setup/lib` は他の任意手順のツールとも共有です。それらがまだ残っていれば `lib` も要ります。
 * 逆順のときは残った側が同じ規則で `lib` を持っていくため、どちらの順序でも残骸が出ません。
 */
export function sharedModuleTargets(anyUserExists: boolean): string[] {
  return anyUserExists ? [] : [SETUP_SHARED_DIR];
}
