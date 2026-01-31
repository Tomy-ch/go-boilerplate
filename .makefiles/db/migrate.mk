## DBマイグレーション関連のコマンド群
# -----Migrateターゲット-----
.PHONY: new-migrate-% ## 新しいマイグレーションファイルを生成します
.PHONY: db-migrate-up ## 全てのマイグレーションを最新まで適用
.PHONY: db-migrate-up-% ## 指定したバージョンまでのマイグレーションを適用
.PHONY: db-migrate-down ## 全てのマイグレーションを初期状態までダウングレード
.PHONY: db-migrate-down-% ## 指定したバージョンまでのマイグレーションをダウングレード
# -----LocalDBに対してのMigrateエイリアス-----
.PHONY: db-local-migrate-up ## localDB: 全てのマイグレーションを最新まで適用
.PHONY: db-local-migrate-up-% ## localDB: 指定したバージョンまでのマイグレーションを適用
.PHONY: db-local-migrate-down ## localDB: 全てのマイグレーションを初期状態までダウングレード
.PHONY: db-local-migrate-down-% ## localDB: 指定したバージョンまでのマイグレーションをダウングレード
# -----TestDBに対してのMigrateエイリアス-----
.PHONY: db-test-migrate-up ## testDB: 全てのマイグレーションを最新まで適用
.PHONY: db-test-migrate-up-% ## testDB: 指定したバージョンまでのマイグレーションを適用
.PHONY: db-test-migrate-down ## testDB: 全てのマイグレーションを初期状態までダウングレード
.PHONY: db-test-migrate-down-% ## testDB: 指定したバージョンまでのマイグレーションをダウングレード
# -----CI用ターゲット-----
.PHONY: db-migrate-ci-up ## 全てのマイグレーションを最新まで適用（CI用）
.PHONY: db-migrate-ci-up-% ## 指定したバージョンまでのマイグレーションを適用（CI用）
.PHONY: db-migrate-ci-down ## 全てのマイグレーションを初期状態までダウングレード（CI用）
.PHONY: db-migrate-ci-down-% ## 指定したバージョンまでのマイグレーションをダウングレード（CI用）

# -----migrateターゲット-----
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
	@echo "🧱 マイグレーション: 最新版までアップグレードします... (database=$(DB))"
	@docker compose run --rm go_tool_runner make db-migrate-ci-up
	@echo "✅ 完了：全マイグレーション適用されました。 (database=$(DB))"

db-migrate-up-%:
	@version=$*; \
	echo "🧱 マイグレーション: バージョン $$version 版までアップグレードします... (database=$(DB))"; \
	docker compose run --rm go_tool_runner make db-migrate-ci-up-$$version; \
	echo "✅ 完了：バージョン $$version まで適用されました。 (database=$(DB))"

db-migrate-down:
	@echo "💥 マイグレーション: 初期状態までダウングレードします... (database=$(DB))"
	@docker compose run --rm go_tool_runner make db-migrate-ci-down
	@echo "✅ 完了：全マイグレーションダウングレードされました。 (database=$(DB))"

db-migrate-down-%:
	@version=$*; \
	echo "💥 マイグレーション: バージョン $$version までダウングレードします... (database=$(DB))"; \
	docker compose run --rm go_tool_runner make db-migrate-ci-down-$$version; \
	echo "✅ 完了：バージョン $$version までダウングレードされました。 (database=$(DB))"

# -----LocalDBに対してのMigrateエイリアス-----
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

# -----TestDBに対してのMigrateエイリアス-----
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

# -----CI用ターゲット-----
db-migrate-ci-up:
	go run cmd/main.go migrate-up --database $(DB)

db-migrate-ci-up-%:
	go run cmd/main.go migrate-up --database $(DB) --version $*

db-migrate-ci-down-%:
	go run cmd/main.go migrate-down --database $(DB) --version $*

db-migrate-ci-down:
	go run cmd/main.go migrate-down --database $(DB)
