## GitHub Actions ワークフローの状態操作
.PHONY: enable-workflows ## disabled_fork 状態のワークフローを一括で有効化します

# テンプレート由来のリポジトリでは、GitHub が全ワークフローを disabled_fork 状態で
# 作成する。この状態のワークフローは push にも schedule にも反応せず、しかも Actions タブを
# 開くまで気付きにくい。セキュリティ系のスキャナは「動いているつもりで一度も動いていない」が
# 最悪の失敗なので、リポジトリ初期化の一環でまとめて有効化する。
#
# 冪等：既に有効なものは列挙されないため、再実行しても何もしない。
enable-workflows:
	@if ! gh auth status >/dev/null 2>&1; then \
		echo "❌ gh が未認証です。先に make gh-login を実行してください。"; \
		exit 1; \
	fi
	@DISABLED=$$(gh workflow list --all --json id,name,state \
		-q '.[] | select(.state == "disabled_fork") | "\(.id)\t\(.name)"'); \
	if [ -z "$$DISABLED" ]; then \
		echo "🟡 disabled_fork 状態のワークフローはありません。"; \
	else \
		printf '%s\n' "$$DISABLED" | while IFS="$$(printf '\t')" read -r id name; do \
			printf '  有効化: %s\n' "$$name"; \
			gh workflow enable "$$id" || echo "  ⚠️ 有効化に失敗: $$name"; \
		done; \
		echo "✅ ワークフローの有効化が完了しました。"; \
	fi
