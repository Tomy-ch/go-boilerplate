## リリースタグの設定とリリースノートの設定コマンド
# 最新タグの解決・次バージョンの算出・production の同期・タグと GitHub Release の作成は
# scripts/release（テスト付き）が持つ。取り消しの効かない操作を含み、分岐を実地で確かめるには
# 本当にリリースするしかないため、手順と中止条件をシェルに置かない。
# git / gh はホストの認証情報を使うのでツールランナーは経由しない（cmd/db-slot と同じ扱い）。

.PHONY: tag-patch ## リリースタグ(vX.Y.Z+1)を作成
.PHONY: tag-minor ## リリースタグ(vX.Y+1.0)を作成
.PHONY: tag-major ## リリースタグ(vX+1.0.0)を作成

RELEASE_TOOL := go run ./scripts/release

tag-patch:
	@$(RELEASE_TOOL) tag -bump patch

tag-minor:
	@$(RELEASE_TOOL) tag -bump minor

tag-major:
	@$(RELEASE_TOOL) tag -bump major
