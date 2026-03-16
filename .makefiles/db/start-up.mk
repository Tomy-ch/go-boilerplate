## DB関連のコマンドの一括実行
.PHONY: db-init ## DBの初期化を行う(マイグレーション、シードデータ投入、スキーマ更新)
.PHONY: db-init-local ## LocalDBの初期化を行う(マイグレーション、シードデータ投入)
.PHONY: db-init-test ## TestDBの初期化を行う(マイグレーション、シードデータ投入)

db-init:
	@echo "🔄 DB初期化関連のコマンドを実行します..."
	@make db-init-local
	@make db-init-test
	@make gen-db-schema
	@echo "✅ DB初期化関連のコマンドが完了しました。"

db-init-local:
	@echo "🔄 localDBを初期化します..."
	@make db-local-migrate-down
	@make db-local-migrate-up
	@make db-local-seed
	@echo "✅ localDBの初期化が完了しました。"

db-init-test:
	@echo "🔄 testDBを初期化します..."
	@make db-test-migrate-down
	@make db-test-migrate-up
	@make db-test-seed
	@echo "✅ testDBの初期化が完了しました。"
