## リリースブランチの切り替えコマンド
# 手順の実体は scripts/release（テスト付き）。設計意図は release-tag.mk のコメントを参照。
# RELEASE_TOOL は release-tag.mk が定義する。本ファイルのほうが先に include されるが、
# 参照はレシピ内（実行時展開）なので解決順は問題にならない。

.PHONY: hotfix-patch ## hotfixブランチ(vX.Y.Z+1)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: branch-patch ## releaseブランチ(vX.Y.Z+1)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: branch-minor ## releaseブランチ(vX.Y+1.0)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: branch-major ## releaseブランチ(vX+1.0.0)を作成して、デフォルトブランチに設定(現在のタグ基準)

hotfix-patch:
	@$(RELEASE_TOOL) branch -bump patch -prefix hotfix

branch-patch:
	@$(RELEASE_TOOL) branch -bump patch -prefix release

branch-minor:
	@$(RELEASE_TOOL) branch -bump minor -prefix release

branch-major:
	@$(RELEASE_TOOL) branch -bump major -prefix release
