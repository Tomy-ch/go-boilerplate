## データベース関連のコマンド群
.PHONY: db-schema ## スキーマの更新を実行
.PHONY: new-migrate-% ## 新しいマイグレーションファイルを生成します
.PHONY: db-migrate-up ## 全てのマイグレーションを最新まで適用
.PHONY: db-migrate-up-% ## 指定したバージョンまでのマイグレーションを適用
.PHONY: db-migrate-down ## 全てのマイグレーションを初期状態までダウングレード
.PHONY: db-migrate-down-% ## 指定したバージョンまでのマイグレーションをダウングレード
.PHONY: db-seed ## データベースにシードデータを投入
.PHONY: db-init ## DBの初期化を行う(マイグレーション、シードデータ投入、スキーマ更新)
.PHONY: db-logs-delete ## DBのログを削除

# 対象DB（local / test / prd など）。未指定なら local
DB ?= local

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

# -------------------------------
# 汎用ターゲット（DB可変）
# 使い方:
#   make db-migrate-up DB=test
#   make db-migrate-up-123 DB=prd
#   make db-migrate-down DB=local
#   make db-seed DB=test
# -------------------------------
db-migrate-up:
	@echo "🔄 マイグレーション: 最新版までアップグレードします... (database=$(DB))"
	docker compose run --rm go_tool_runner go run cmd/main.go migrate-up --database $(DB)
	@echo "✅ 完了：全マイグレーション適用されました。 (database=$(DB))"

db-migrate-up-%:
	@version=$*; \
	echo "🔄 マイグレーション: バージョン $$version 版までアップグレードします... (database=$(DB))"; \
	docker compose run --rm go_tool_runner go run cmd/main.go migrate-up --database $(DB) --version $$version; \
	echo "✅ 完了：バージョン $$version まで適用されました。 (database=$(DB))"

db-migrate-down:
	@echo "🔄 マイグレーション: 初期状態までダウングレードします... (database=$(DB))"
	docker compose run --rm go_tool_runner go run cmd/main.go migrate-down --database $(DB)
	@echo "✅ 完了：全マイグレーションダウングレードされました。 (database=$(DB))"

db-migrate-down-%:
	@version=$*; \
	echo "🔄 マイグレーション: バージョン $$version までダウングレードします... (database=$(DB))"; \
	docker compose run --rm go_tool_runner go run cmd/main.go migrate-down --database $(DB) --version $$version; \
	echo "✅ 完了：バージョン $$version までダウングレードされました。 (database=$(DB))"

db-seed:
	@echo "🔄 データベースにシードデータを投入します... (database=$(DB))"
	docker compose run --rm go_tool_runner go run cmd/main.go db-seed --database $(DB)
	@echo "✅ シードデータの投入が完了しました。 (database=$(DB))"

# -------------------------------
# localDBに対してのエイリアス（local固定）
# 例: make db-local-migrate-up, make db-local-seed
# -------------------------------
db-local-migrate-up: DB=local
db-local-migrate-up: db-migrate-up

db-local-migrate-up-%: DB=local
db-local-migrate-up-%: db-migrate-up-%

db-local-migrate-down: DB=local
db-local-migrate-down: db-migrate-down

db-local-migrate-down-%: DB=local
db-local-migrate-down-%: db-migrate-down-%

db-local-seed: DB=local
db-local-seed: db-seed

db-init-local:
	@echo "🔄 localDBを初期化します..."
	@make db-local-migrate-down
	@make db-local-migrate-up
	@make db-local-seed
	@echo "✅ localDBの初期化が完了しました。"

# -------------------------------
# testDBに対してのエイリアス（test固定）
# 例: make db-test-migrate-up, make db-test-seed
# -------------------------------
db-test-migrate-up: DB=test
db-test-migrate-up: db-migrate-up

db-test-migrate-up-%: DB=test
db-test-migrate-up-%: db-migrate-up-%

db-test-migrate-down: DB=test
db-test-migrate-down: db-migrate-down

db-test-migrate-down-%: DB=test
db-test-migrate-down-%: db-migrate-down-%

db-test-seed: DB=test
db-test-seed: db-seed

db-init-test:
	@echo "🔄 testDBを初期化します..."
	@make db-test-migrate-down
	@make db-test-migrate-up
	@make db-test-seed
	@echo "✅ testDBの初期化が完了しました。"

db-init:
	@echo "🔄 DB初期化関連のコマンドを実行します..."
	@make db-init-local
	@make db-init-test
	@make db-schema
	@echo "✅ DB初期化関連のコマンドが完了しました。"

db-logs-delete:
	@rm -rf docker/database/logs/*
