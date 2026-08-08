// サンプルAPI削除に固有の判定。削除対象の宣言（SAMPLE_DOMAINS / MARKER_FILES / BUILD_STEPS）は
// sample-manifest.ts、マーカー除去の機構は ../lib/markers.ts を参照。

import { type StripResult, stripMarkers } from "../lib/markers";

/** サンプルAPI の在否で行を切り替えるマーカー名。 */
export const SAMPLE_MARKER = "sample-api";

/** サンプルAPI のマーカー行を除去する。 */
export function stripSampleMarkers(content: string): StripResult {
  return stripMarkers(content, SAMPLE_MARKER);
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
export const SETUP_SHARED_DIR_USERS: readonly string[] = [SETUP_VERIFIER_DIR, "remove-dast-setting"];

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
