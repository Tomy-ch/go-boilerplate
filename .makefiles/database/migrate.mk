# 連番の重複・欠番の判定は scripts/migration-lint（テスト付き）が持つ。これは lefthook の
# pre-commit ゲートで、壊れ方が「何も検査しなくなる」方向に出るため、判定をシェルに置かない。
# 自前ツールなのでツールランナーは経由せずホストで実行する（cmd/db-slot と同じ扱い）。
MIGRATION_LINT := go run ./scripts/migration-lint

## DBマイグレーション関連のコマンド群
# -----Migrateターゲット-----
.PHONY: new-migrate-% ## 新しいマイグレーションファイルを生成します
.PHONY: check-migration-up-version ## up マイグレーションのバージョン重複をチェックします
.PHONY: check-migration-down-version ## down マイグレーションのバージョン重複をチェックします
.PHONY: check-migration-up-gap ## up マイグレーションのバージョンギャップをチェックします
.PHONY: check-migration-down-gap ## down マイグレーションのバージョンギャップをチェックします
.PHONY: db-migrate-up ## 全てのマイグレーションを最新まで適用
.PHONY: db-migrate-up-% ## 指定した段数だけマイグレーションを適用
.PHONY: db-migrate-down ## 全てのマイグレーションを初期状態までダウングレード
.PHONY: db-migrate-down-% ## 指定した段数だけマイグレーションをダウングレード
# -----LocalDBに対してのMigrateエイリアス-----
.PHONY: db-local-migrate-up ## localDB: 全てのマイグレーションを最新まで適用
.PHONY: db-local-migrate-up-% ## localDB: 指定した段数だけマイグレーションを適用
.PHONY: db-local-migrate-down ## localDB: 全てのマイグレーションを初期状態までダウングレード
.PHONY: db-local-migrate-down-% ## localDB: 指定した段数だけマイグレーションをダウングレード
# -----TestDBに対してのMigrateエイリアス-----
.PHONY: db-test-migrate-up ## testDB: 全てのマイグレーションを最新まで適用
.PHONY: db-test-migrate-up-% ## testDB: 指定した段数だけマイグレーションを適用
.PHONY: db-test-migrate-down ## testDB: 全てのマイグレーションを初期状態までダウングレード
.PHONY: db-test-migrate-down-% ## testDB: 指定した段数だけマイグレーションをダウングレード
# -----CI用ターゲット-----
.PHONY: db-migrate-ci-up ## 全てのマイグレーションを最新まで適用（CI用）
.PHONY: db-migrate-ci-up-% ## 指定した段数だけマイグレーションを適用（CI用）
.PHONY: db-migrate-ci-down ## 全てのマイグレーションを初期状態までダウングレード（CI用）
.PHONY: db-migrate-ci-down-% ## 指定した段数だけマイグレーションをダウングレード（CI用）

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
	@echo "✅ 新しいマイグレーションファイルを生成しました: database/migrations/<連番>_$*.up.sql / .down.sql"

check-migration-up-version:
	@$(MIGRATION_LINT) -kind up -check duplicate

check-migration-down-version:
	@$(MIGRATION_LINT) -kind down -check duplicate

check-migration-up-gap:
	@$(MIGRATION_LINT) -kind up -check gap

check-migration-down-gap:
	@$(MIGRATION_LINT) -kind down -check gap

# -------------------------------
# 汎用ターゲット（DB可変）
# 使い方:
#   make db-migrate-up DB=test
#   make db-migrate-up-2 DB=prd   # 現在位置から 2 段だけ適用（相対ステップ数）
#   make db-migrate-down DB=local
#   make db-seed DB=test
# -------------------------------
db-migrate-up: require-db-owner
	@echo "🧱 マイグレーション: 最新版までアップグレードします... (database=$(DB))"
	@docker compose run --rm go_tool_runner make db-migrate-ci-up DB=$(DB)
	@echo "✅ 完了：全マイグレーション適用されました。 (database=$(DB))"

db-migrate-up-%: require-db-owner
	@steps=$* && \
	echo "🧱 マイグレーション: 現在位置から $$steps 段アップグレードします... (database=$(DB))" && \
	docker compose run --rm go_tool_runner make db-migrate-ci-up-$$steps DB=$(DB) && \
	echo "✅ 完了：$$steps 段適用されました。 (database=$(DB))"

db-migrate-down: require-db-owner
	@echo "💥 マイグレーション: 初期状態までダウングレードします... (database=$(DB))"
	@docker compose run --rm go_tool_runner make db-migrate-ci-down DB=$(DB)
	@echo "✅ 完了：全マイグレーションダウングレードされました。 (database=$(DB))"

db-migrate-down-%: require-db-owner
	@steps=$* && \
	echo "💥 マイグレーション: 現在位置から $$steps 段ダウングレードします... (database=$(DB))" && \
	docker compose run --rm go_tool_runner make db-migrate-ci-down-$$steps DB=$(DB) && \
	echo "✅ 完了：$$steps 段ダウングレードされました。 (database=$(DB))"

# -----LocalDBに対してのMigrateエイリアス-----
# 例: make db-local-migrate-up, make db-local-seed
# 対象は所有している local 系データベース（主 checkout=local / 取得済み worktree=wt<N>_local）。
# local 直書きに戻すと、取得済み worktree からでも主 checkout のデータベースを触れてしまう。
# -------------------------------
db-local-migrate-up: DB=$(DB_LOCAL)
db-local-migrate-up: db-migrate-up

db-local-migrate-up-%: DB=$(DB_LOCAL)
db-local-migrate-up-%: db-migrate-up-%

db-local-migrate-down: DB=$(DB_LOCAL)
db-local-migrate-down: db-migrate-down

db-local-migrate-down-%: DB=$(DB_LOCAL)
db-local-migrate-down-%: db-migrate-down-%

# -----TestDBに対してのMigrateエイリアス-----
# 例: make db-test-migrate-up, make db-test-seed
# -------------------------------
db-test-migrate-up: DB=$(DB_TEST)
db-test-migrate-up: db-migrate-up

db-test-migrate-up-%: DB=$(DB_TEST)
db-test-migrate-up-%: db-migrate-up-%

db-test-migrate-down: DB=$(DB_TEST)
db-test-migrate-down: db-migrate-down

db-test-migrate-down-%: DB=$(DB_TEST)
db-test-migrate-down-%: db-migrate-down-%

# -----CI用ターゲット-----
db-migrate-ci-up:
	go run ./cmd/ migrate-up --database $(DB)

db-migrate-ci-up-%:
	go run ./cmd/ migrate-up --database $(DB) --steps $*

db-migrate-ci-down-%:
	go run ./cmd/ migrate-down --database $(DB) --steps $*

db-migrate-ci-down:
	go run ./cmd/ migrate-down --database $(DB)
