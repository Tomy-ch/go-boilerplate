## ツールのバージョン管理
.PHONY: sync-tools ## ツールのバージョンを tools.yaml と同期する

sync-tools:
	@echo "🔧 ツールのバージョンを tools.yaml と同期中..."
	@docker compose run --rm node_tool_runner node scripts/replace-tools-version.cjs
	@echo "✅ ツールのバージョンの同期が完了しました。再度 make install-tools を実行してツールをインストールしてください。"
