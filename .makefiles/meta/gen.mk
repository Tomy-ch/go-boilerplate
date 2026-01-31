## ドキュメント関連のコマンド群
.PHONY: gen-tools-meta ## 生成ツールのバージョン情報を出力する

gen-tools-meta:
	@echo "🔍 生成ツールのバージョン情報を出力します..."
	docker compose run --rm go_tool_runner sh scripts/gen_generator_versions.sh go
	docker compose run --rm node_tool_runner sh scripts/gen_generator_versions.sh node
	docker compose run --rm python_tool_runner sh scripts/gen_generator_versions.sh python
	@echo "✅ 生成ツールのバージョン情報の出力が完了しました。"
