// DAST（OWASP ZAP）のサンプル設定一式の撤去対象を宣言する。マーカー除去の機構は ../lib/markers.ts が持つ。
//
// 撤去は「するか、しないか」の二択で、有効/無効を切り替える設定は用意しない。切り替えを持てば、
// 無効にしたまま存在し続ける設定というもう 1 つの状態が生まれ、それは誰にも読まれないまま腐る。
// 撤去後に中身を参照したくなったときは git の履歴から辿れる。

/** DAST 設定の在否で行を切り替えるマーカー名。 */
export const DAST_MARKER = "dast";

/**
 * 丸ごと削除するパス（リポジトリルートからの相対）。
 *
 * @remarks
 * 撤去ツール自身も含みます。一度きりの操作なので、残しても二度目は何も消せません。
 */
export const DAST_PATHS: readonly string[] = [
  ".github/workflows/dast.yaml",
  ".github/zap",
  "scripts/setup/remove-dast-setting",
];

/**
 * DAST 行が残る行と混在するため、行単位でマーカー除去するファイル。
 *
 * @remarks
 * ワークフロー一覧（対訳ペア）とセットアップ手順（対訳ペア）に加えて、このツールの dry-run を
 * 叩く CI 定義を含みます。ツールが消えた後も検査だけが残ると、存在しないスクリプトを実行しよう
 * として落ちます。
 */
export const DAST_MARKER_FILES: readonly string[] = [
  ".github/workflows/README.md",
  ".github/workflows/README.ja.md",
  ".github/workflows/setup-scripts-check.yaml",
  "docs/get-started/setup-repository.md",
  "docs/ja/get-started/setup-repository.ja.md",
];

/** GitHub Actions の pin lockfile。 */
export const ACTION_PIN_LOCKFILE = ".github/actions-pin.toml";

/** DAST が使う唯一の外部 action。撤去後は lockfile から参照されなくなる。 */
export const DAST_ACTION_PIN_KEY = "zaproxy/action-api-scan";

/**
 * pin lockfile から指定 action のエントリ行を落とす。書き換え不要なら `null` を返す。
 *
 * @remarks
 * ここだけマーカーではなくキーで消すのは、lockfile を `make pin-actions-resolve` が
 * 毎回まるごと書き直すためです。行末にマーカーコメントを置いても次の resolve で消え、
 * 撤去は静かに取りこぼします。
 *
 * 落とし忘れると `make pin-actions-check` が「参照されていないエントリ」で落ちます。
 * 撤去した利用者の側で初めて赤くなるので、こちらでは気づけません。
 */
export function stripActionPin(content: string, actionKey: string): string | null {
  const lines = content.split("\n");
  const kept = lines.filter((line) => !line.startsWith(`"${actionKey}@`));

  return kept.length === lines.length ? null : kept.join("\n");
}

/**
 * 削除対象として受け付けてよい相対パスかを判定する。
 *
 * @remarks
 * 宣言の書き間違い（空文字・絶対パス・`..` を含むパス）が、そのままリポジトリ外や
 * リポジトリ自体の削除にならないための門です。dry-run でも必ず通し、削除の前に弾きます。
 */
export function isRemovablePath(relativePath: string): boolean {
  if (relativePath === "" || relativePath.startsWith("/")) {
    return false;
  }

  return !relativePath.split("/").includes("..");
}
