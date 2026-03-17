## ドキュメント関連のコマンド群
.PHONY: gen-tools-meta ## 生成ツールのバージョン情報を出力する
.PHONY: gen-docs-json ## Portal用のドキュメントリンクのJSONを生成する
.PHONY: gen-docs-json-ci ## Portal用のドキュメントリンクのJSONを生成する（CI用）

gen-tools-meta:
	@echo "🔍 生成ツールのバージョン情報を出力します..."
	docker compose run --rm go_tool_runner sh scripts/gen_tools_version.sh go
	docker compose run --rm node_tool_runner sh scripts/gen_tools_version.sh node
	docker compose run --rm python_tool_runner sh scripts/gen_tools_version.sh python
	@echo "✅ 生成ツールのバージョン情報の出力が完了しました。"

gen-docs-json-ci:
	node scripts/generate-docs-json.js

gen-docs-json:
	@echo "🔍 Portal用のドキュメントリンクのJSONを生成します..."
	docker compose run --rm node_tool_runner make gen-docs-json-ci
	@echo "✅ Portal用のドキュメントリンクのJSONの生成が完了しました。"
