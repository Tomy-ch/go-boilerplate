## アプリケーションサーバー関連
.PHONY: serve ## 開発環境の起動
.PHONY: serve-build ## キャッシュを利用したビルド後、開発環境を起動する
.PHONY: serve-build-clean ## キャッシュ無効・base image を pull したクリーンビルド後、開発環境を起動する
.PHONY: tools ## 開発ツールの起動
.PHONY: tools-build ## 開発ツールコンテナをビルドする（キャッシュ利用・起動はしない）
.PHONY: tools-build-clean ## 開発ツールコンテナをクリーンビルドする（--no-cache --pull・起動はしない）
.PHONY: smoke ## Smoke Test環境の起動

serve:
	@echo "🔄 開発環境を起動します。"
	@docker compose --profile development up -d
	@echo "✅ 開発環境の起動が完了しました。"

serve-build:
	@echo "🧰 ビルド後、開発環境を起動します。"
	@docker compose --profile development up -d --build
	@echo "✅ 開発環境の起動が完了しました。"

serve-build-clean:
	@echo "🧹 クリーンビルド後、開発環境を起動します（--no-cache --pull）。"
	@docker compose --profile development build --no-cache --pull
	@docker compose --profile development up -d
	@echo "✅ 開発環境の起動が完了しました。"

tools:
	@echo "🔄 開発ツールを起動します。"
	@docker compose --profile tools up -d --build
	@echo "✅ 開発ツールの起動が完了しました。"

tools-build:
	@echo "🧰 開発ツールコンテナをビルドします。"
	@docker compose build go_tool_runner node_tool_runner python_tool_runner
	@echo "✅ 開発ツールコンテナのビルドが完了しました。"

tools-build-clean:
	@echo "🧹 開発ツールコンテナをクリーンビルドします（--no-cache --pull）。"
	@docker compose build --no-cache --pull go_tool_runner node_tool_runner python_tool_runner
	@echo "✅ 開発ツールコンテナのクリーンビルドが完了しました。"

smoke:
	@echo "🔄 Smoke Test環境を起動します。"
	@docker compose --profile smoke up --build -d smoke_server
	@echo "✅ Smoke Test環境の起動が完了しました。"
