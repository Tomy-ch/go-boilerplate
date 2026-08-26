## ベースブランチ（フィーチャーブランチの分岐元）の解決
# -----ホスト上で実行するコマンド群-----
.PHONY: base-branch ## 最新のリリースライン(release/vX.Y.Z)のブランチ名を1行で出力する

# -----ホスト上で実行するコマンド群-----
# 解決は scripts/base-branch（テスト付き）が持つ。origin/HEAD も GitHub のデフォルトブランチも
# 古い答えを黙って返す理由と「最新」の定義は .makefiles/README.md の base-branch 行、および
# scripts/base-branch のパッケージコメント。
#
# git はホストの認証情報を使うのでツールランナーは経由しない（scripts/release と同じ扱い）。
#
# 出力はブランチ名 1 行だけ。コマンド置換でそのまま受けられるよう、@ で余計な行を出さない。
base-branch:
	@go run ./scripts/base-branch
