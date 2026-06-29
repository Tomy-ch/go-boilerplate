## DB関連のコマンドの一括実行
.PHONY: db-init ## DBの初期化を行う(マイグレーション、シードデータ投入)
.PHONY: db-init-local ## LocalDBの初期化を行う(マイグレーション、シードデータ投入)
.PHONY: db-init-test ## TestDBの初期化を行う(マイグレーション、シードデータ投入)

db-init:
	@echo "🔄 DB初期化関連のコマンドを実行します..."
	@$(MAKE) db-init-local
	@$(MAKE) db-init-test
	@echo "✅ DB初期化関連のコマンドが完了しました。"

db-init-local:
	@echo "🔄 localDBを初期化します..."
	@$(MAKE) db-local-migrate-down
	@$(MAKE) db-local-migrate-up
	@$(MAKE) db-local-seed
	@echo "✅ localDBの初期化が完了しました。"

db-init-test:
	@echo "🔄 testDBを初期化します..."
	@$(MAKE) db-test-migrate-down
	@$(MAKE) db-test-migrate-up
	@$(MAKE) db-test-seed
	@echo "✅ testDBの初期化が完了しました。"
