// ボイラープレートでのみ成り立つ記述の宣言と、対象の走査判定。マーカー除去の機構だけは
// ../lib/markers.ts と共有する。
//
// このモジュールは撤去の成功後にツール自身と一緒に消える。初期化と同じく一度きりの操作で、
// 消えた後の利用者のリポジトリでは対象が存在しないため、残しても失敗しかできない。
//
// 走査の判定は remove-sample-api 側とほぼ同じ形をしているが、**共通化していない**。2 つの撤去は
// 起爆の契機が違い（こちらはセットアップ、あちらはサンプル削除）、どちらが先に走るかも順序が
// 決まっていない。共有モジュールへ寄せると、先に走ったほうがもう一方の足元を持って行きうる
// 依存が増える。判定は数行なので、重複を許して各ツールが自分の分を抱える。

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
 * 「マーカーを書いたのに除去されない」を静かに作るためです。除去されない条件は、ここと
 * 下の接頭辞に挙げた場所に居ることだけに限ります。
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
 * マーカー文字列を「データ・散文」として持つファイル。走査の対象から外す。
 *
 * @remarks
 * 空なのは、文書側にこの名前空間の例示が無いためです。マーカーの形を説明する箇所（集約先の
 * 規約表、セットアップ手順の `grep`）はいずれもバッククォートや引用符の内側にあり、コメント
 * 記号を伴わないので走査には当たりません。マーカーの形を入力として持つのはこのツール自身の
 * テストだけで、そちらは `SELF_DIR` が丸ごと外します。コメントとして例示を書いた瞬間に
 * ここへの登録が要ります（`remove-sample-api` 側は同じ理由で教材とフィクスチャを宣言）。
 */
export const MARKER_LITERAL_FILES: readonly string[] = [];

/**
 * このツール自身のディレクトリ。走査から外す。
 *
 * @remarks
 * 撤去の最後に `selfDestruct()` が丸ごと消すので、書き換えても意味がありません。むしろ害が
 * あります。宣言やテストはマーカーの形そのものを本文に持つため、走査すると「マーカーではない
 * もの」を刈り取るか、対応の取れない片割れとして除去全体を止めます。`remove-sample-api` 側で
 * 削除登録パスを外しているのと同じ理由です。
 */
export const SELF_DIR = "scripts/setup/remove-boilerplate-identity";

/**
 * マーカーではなくパスで消えるファイル / ディレクトリ。
 *
 * @remarks
 * 全体が上流限定であるものは、領域を囲うより丸ごと消すほうが安全です。散文から領域を切り出すと
 * 前後の文が修復対象になりますが、まるごとなら継ぎ目が生まれません。
 *
 * `premise-lint` がここに居るのは、あれが「上流である間だけ意味を持つ検査」だからです。守って
 * いる規則（`docs/rules.md` の *No premise the document will outlive*）は一般形なので fork が
 * 継承しますが、検査が探す言い回し（`this template` / `adopters` / `このテンプレート`）は上流
 * 固有の実例でしかありません。fork にはその前提が無いので、残しても永久に緑のままの検査が
 * 増えるだけで、赤くなったときに何を意味するのかも読めません。
 *
 * `marker-baseline` も同じです。あれが守るのは撤去マーカーを**書く側**で、書く場面は上流に
 * しかありません。fork が受け取るのはマーカーが解決し終えたツリーなので、見張る対象が居ません。
 *
 * `docs/plan` は上流が着手していないリリース線の要件です。ロードマップ本体と違い fork 側で
 * 書き換えて使える形をしておらず、中身は「このボイラープレートが次に何を作るか」でしかないので、
 * 領域を囲うのではなくディレクトリごと消します。ロードマップは残します——あちらは導入部の
 * 置換マーカーが fork 向けの文面を持っており、方向を記録する場所として引き継げます。
 */
export const BOILERPLATE_DELETE_PATHS: readonly string[] = [
  "docs/get-started/boilerplate-only-conventions.md",
  "docs/ja/get-started/boilerplate-only-conventions.ja.md",
  "docs/plan",
  "docs/ja/plan",
  "scripts/premise-lint",
  "scripts/marker-baseline",
];

/** 撤去後に残ってはいけない語。検査が的を外していないかの確認にも使う。 */
export const BOILERPLATE_PROSE_MARKERS: readonly string[] = ["boilerplate", "ボイラープレート"];

/** 丸ごと消えるパスの配下か。消すものを書き換えても意味が無い。 */
function isDeletedWhole(normalizedPath: string): boolean {
  return BOILERPLATE_DELETE_PATHS.some(
    (target) => normalizedPath === target || normalizedPath.startsWith(`${target}/`),
  );
}

/**
 * 走査対象か。ディレクトリ名の除外は列挙側が行うため、ここは接頭辞・自ディレクトリ・
 * 丸ごと消えるパス・literal を見る。
 *
 * @remarks
 * 丸ごと消えるパスを外すのは、除去が削除より先に走るためです。順序を入れ替えても `dryRun` では
 * 何も消えないので、走査には現れ続けます。`premise-lint` のテストはマーカーの形を入力として
 * 持つので、外さないと対応の取れない片割れとして除去全体が止まります（実際に止まりました）。
 */
export function isScanTarget(relativePath: string): boolean {
  const normalized = relativePath.split("\\").join("/");

  if (
    MARKER_LITERAL_FILES.includes(normalized) ||
    normalized.startsWith(`${SELF_DIR}/`) ||
    isDeletedWhole(normalized)
  ) {
    return false;
  }

  return !EXCLUDED_PATH_PREFIXES.some((prefix) => normalized.startsWith(prefix));
}

/** `<comment> boilerplate-only:` を含むか。宣言の陳腐化を検査する側が使う。 */
export function containsBoilerplateMarker(content: string): boolean {
  return new RegExp(`(?:\\/\\/|#|<!--)\\s*${BOILERPLATE_MARKER}:`).test(content);
}
