// ボイラープレートでのみ成り立つ記述の宣言。マーカー除去の機構は ../lib/markers.ts が持つ。
//
// このモジュールは撤去の成功後にツール自身と一緒に消える。初期化と同じく一度きりの操作で、
// 消えた後の利用者のリポジトリでは対象が存在しないため、残しても失敗しかできない。

/**
 * 上流のボイラープレートである間だけ成り立つ記述の在否を切り替えるマーカー名。
 *
 * @remarks
 * かつては素の `boilerplate` と `boilerplate-only` が併存していました。前者はリポジトリが自分を
 * ボイラープレートと名乗る散文を、後者はボイラープレートでのみ成り立つ規約を指す、という
 * 分け方でしたが、境界は書き手の主観にしかなく、どちらの名前で書いても除去は同じ契機
 * （セットアップ）で起きます。2 つあること自体が「どちらで書くべきか」という誤りうる判断を
 * 毎回作っていたため、`boilerplate-only` へ寄せました。
 */
export const BOILERPLATE_MARKER = "boilerplate-only";

/**
 * 走査から外すディレクトリ名。
 *
 * @remarks
 * 依存の取得物と VCS の内部です。走査対象を拡張子や名前で絞り込まないのは、絞り込みが
 * 「マーカーを書いたのに除去されない」を静かに作るためです。除去されない条件は、ここに
 * 挙げた取得物の中に居ることだけに限ります。
 */
export const EXCLUDED_DIRECTORIES: ReadonlySet<string> = new Set([
  ".git",
  "node_modules",
  "vendor",
]);

/**
 * 走査から外す相対パス接頭辞。
 *
 * @remarks
 * いずれも生成物です。マーカーを持っていても再生成で戻るため、ここで除去しても意味がなく、
 * 生成前後で内容が食い違う分だけ有害です。
 */
export const EXCLUDED_PATH_PREFIXES: readonly string[] = [
  "docs/portal/guides/",
  "docs/coverage/",
  "docs/db-schema/",
  "docs/godoc/",
  "graphify-out/",
  "tmp/",
];

/**
 * マーカーではなくパスで消えるファイル。
 *
 * @remarks
 * 全体が上流限定であるものは、領域を囲うより丸ごと消すほうが安全です。散文から領域を切り出すと
 * 前後の文が修復対象になりますが、ファイルごとなら継ぎ目が生まれません。
 */
export const BOILERPLATE_DELETE_FILES: readonly string[] = [
  "docs/get-started/boilerplate-only-conventions.md",
  "docs/ja/get-started/boilerplate-only-conventions.ja.md",
];

/** 撤去後に残ってはいけない語。検査が的を外していないかの確認にも使う。 */
export const BOILERPLATE_PROSE_MARKERS: readonly string[] = ["boilerplate", "ボイラープレート"];

/** 走査対象か。ディレクトリ名の除外は列挙側が行うため、ここは接頭辞だけを見る。 */
export function isScanTarget(relativePath: string): boolean {
  const normalized = relativePath.split("\\").join("/");

  return !EXCLUDED_PATH_PREFIXES.some((prefix) => normalized.startsWith(prefix));
}
