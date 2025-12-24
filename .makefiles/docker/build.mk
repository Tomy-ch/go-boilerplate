## Dockerを用いた開発環境関連のコマンド群
.PHONY: serve ## 開発環境の起動
.PHONY: serve-build ## ビルド実行後に、開発環境を起動する
.PHONY: tools ## 開発ツールの起動
.PHONY: tools-rebuild ## 開発ツールコンテナの再ビルド
.PHONY: smoke ## Smoke Test環境の起動

serve:
	@echo "🔄 開発環境を起動します。"
	@docker compose --profile development up -d
	@echo "✅ 開発環境の起動が完了しました。"

serve-build:
	@echo "🧰 ビルド後、開発環境を起動します。"
	@docker compose --profile development up -d --build
	@echo "✅ 開発環境の起動が完了しました。"

tools:
	@echo "🔄 開発ツールを起動します。"
	@docker compose --profile tools up -d --build
	@echo "✅ 開発ツールの起動が完了しました。"

tools-rebuild:
	@echo "🔄 開発ツールコンテナを再ビルドします。"
	@docker compose build --no-cache --pull go_tool_runner
	@docker compose build --no-cache --pull node_tool_runner
	@docker compose build --no-cache --pull python_tool_runner
	@echo "✅ 開発ツールコンテナの再ビルドが完了しました。"

smoke:
	@echo "🔄 Smoke Test環境を起動します。"
	@docker compose --profile smoke up --build -d smoke_server
	@echo "✅ Smoke Test環境の起動が完了しました。"
