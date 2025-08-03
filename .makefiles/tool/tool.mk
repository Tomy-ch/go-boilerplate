## 開発ツール系
.PHONY: go-update ## goenvの更新を実行
.PHONY: db-schema ## スキーマの更新を実行
.PHONY: new-migrate-% ## 新しいマイグレーションファイルを生成します

go-update:
	@anyenv update
	@goenv install "$(cat .go-version)"

db-schema:
	@echo "🔄 スキーマの更新を実行します..."
	docker compose run --rm er_diagram_generator
	@echo "✅ スキーマの更新が完了しました。"

new-migrate-%:
	@echo "🔄 新しいマイグレーションファイルを生成します..."
	@file_name=$* && \
	if [ -z "$$file_name" ]; then \
		echo "❌ マイグレーションファイル名を指定してください。"; \
		echo "   例: make new-migrate-create_users_table"; \
		exit 1; \
	fi && \
	docker compose run --rm go_tool_runner migrate create -ext sql -dir database/migrations -seq "$$file_name"
	@echo "✅ 新しいマイグレーションファイルが生成されました: database/migrations/$$file_name.up.sql, database/migrations/$$file_name.down.sql"
