## データベース関連のコマンド群
.PHONY: sql-lint ## SQLマイグレーションファイルのLintを実行
.PHONY: sql-fix ## SQLマイグレーションファイルの自動修正を実行
.PHONY: db-schema ## スキーマの更新を実行
.PHONY: new-migrate-% ## 新しいマイグレーションファイルを生成します
.PHONY: db-migrate-up ## 全てのマイグレーションを最新まで適用
.PHONY: db-migrate-up-% ## 指定したバージョンまでのマイグレーションを適用
.PHONY: db-migrate-down ## 全てのマイグレーションを初期状態までダウングレード
.PHONY: db-migrate-down-% ## 指定したバージョンまでのマイグレーションをダウングレード
.PHONY: db-seed ## データベースにシードデータを投入
.PHONY: db-init ## DBの初期化を行う(マイグレーション、シードデータ投入、スキーマ更新)
.PHONY: db-logs-delete ## DBのログを削除

sql-lint:
	docker compose run --rm python_tool_runner sqlfluff lint database/migrations/ --config docker/database/.sqlfluff

sql-fix:
	docker compose run --rm python_tool_runner sqlfluff fix database/migrations/ --config docker/database/.sqlfluff

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
	@echo "✅ 新しいマイグレーションファイルが生成されました: database/migrations/$$file_name.up-down.sql"

db-migrate-up:
	@echo "🔄 マイグレーション: 最新版までアップグレードします..."
	docker compose run --rm go_tool_runner go run cmd/main.go migrate-up
	@echo "✅ 完了：全マイグレーション適用されました。"

db-migrate-up-%:
	@version=$*; \
	echo "🔄 マイグレーション: バージョン $$version 版までアップグレードします..."; \
	docker compose run --rm go_tool_runner go run cmd/main.go migrate-up $$version; \
	echo "✅ 完了：バージョン $$version まで適用されました。"

db-migrate-down:
	@echo "🔄 マイグレーション: 初期状態までダウングレードします..."
	docker compose run --rm go_tool_runner go run cmd/main.go migrate-down
	@echo "✅ 完了：全マイグレーションダウングレードされました。"

db-migrate-down-%:
	@version=$*; \
	echo "🔄 マイグレーション: バージョン $$version までダウングレードします..."; \
	docker compose run --rm go_tool_runner go run cmd/main.go migrate-down $$version; \
	echo "✅ 完了：バージョン $$version までダウングレードされました。"

db-seed:
	@echo "🔄 データベースにシードデータを投入します..."
	docker compose run --rm go_tool_runner go run cmd/main.go db-seed
	@echo "✅ シードデータの投入が完了しました。"

db-init:
	@echo "🔄 マイグレーション関連のコマンドを実行します..."
	@make db-migrate-down
	@make db-migrate-up
	@make db-seed
	@make db-schema
	@echo "✅ マイグレーション関連のコマンドが完了しました。"

db-logs-delete:
	@rm -rf docker/database/logs/*
