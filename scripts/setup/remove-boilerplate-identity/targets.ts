// ボイラープレートを名乗る痕跡の宣言。マーカー除去の機構は ../lib/markers.ts が持つ。
//
// このモジュールは撤去の成功後にツール自身と一緒に消える。初期化と同じく一度きりの操作で、
// 消えた後の利用者のリポジトリでは対象が存在しないため、残しても失敗しかできない。

/** テンプレート自身を語る行の在否を切り替えるマーカー名。 */
export const BOILERPLATE_MARKER = "boilerplate";

/**
 * マーカー行を落とす対象。
 *
 * @remarks
 * 対象は「テンプレートがテンプレートであることを語っている散文」に限ります。リポジトリ名や
 * モジュール名の `go-boilerplate` は `replace-module` / `replace-repository-reference` が
 * 置換済みなので、ここでは扱いません。
 *
 * `docs/get-started/setup-repository.md` は**ファイルごとは消しません**。インスタンス化手順と
 * 一緒に、運用中も読み返す内容（clamp される設定値のレビュー・除外 ADR への入口）が同居しており、
 * 後者は 14 のパッケージ README から参照されているためです。落とすのは手順の側だけです。
 */
export const BOILERPLATE_MARKER_FILES: readonly string[] = [
  "README.md",
  "README.ja.md",
  "docs/get-started/setup-repository.md",
  "docs/ja/get-started/setup-repository.ja.md",
  // ツール自身が消える以上、それを呼ぶ make ターゲットの宣言と説明も一緒に落とす。
  // 残すと `make help` に並び、叩けば失敗するだけのものになる。
  ".makefiles/github/operation/setup-repository.mk",
  ".makefiles/README.md",
  ".makefiles/README.ja.md",
];

/** 撤去後に残ってはいけない語。検査が的を外していないかの確認にも使う。 */
export const BOILERPLATE_PROSE_MARKERS: readonly string[] = ["boilerplate", "ボイラープレート"];
