## 生成関連のコマンドの一括実行コマンド群
.PHONY: gen ## 各種ドキュメントやコードを生成します
.PHONY: gen-api ## API関連のドキュメントやコードを生成します
.PHONY: gen-docs ## ドキュメント関連の生成を行います
.PHONY: gen-all-docs ## ドキュメント関連の生成を全て行います
.PHONY: gen-query ## SQLCのコード生成を行う
.PHONY: gen-query-repo ## ドメイン用のSQLCのコード生成を行う
.PHONY: gen-query-qs ## クエリサービス用のSQLCのコード生成を行う
.PHONY: gen-query-sysq ## システムクエリ用のSQLCのコード生成を行う

gen:
	@echo "🔄 各種ドキュメントやコードの生成します..."
	@$(MAKE) gen-api
	@$(MAKE) gen-query
	@$(MAKE) gen-docs
	@echo "✅ 各種ドキュメントやコードの生成が完了しました。"

gen-api:
	@$(MAKE) gen-bundle-oapi
	@$(MAKE) gen-api-docs
	@$(MAKE) gen-go-code

gen-docs:
	@$(MAKE) gen-portal-docs
	@$(MAKE) gen-docs-json

gen-all-docs:
	@$(MAKE) gen-api-docs
	@$(MAKE) gen-docs
	@$(MAKE) gen-db-schema
	@$(MAKE) gen-test-repo

gen-query:
	@echo "🔄 SQLCのコードを生成します..."
	@$(MAKE) dump-schema
	@$(MAKE) merge-dml
	@$(MAKE) gen-sqlc
	@$(MAKE) fmt
	@echo "✅ SQLCのコード生成が完了しました。"

gen-query-repo:
	@echo "🔄 ドメイン用のSQLCコードを生成します..."
	@$(MAKE) dump-schema
	@$(MAKE) merge-dml-repo
	@$(MAKE) gen-sqlc
	@$(MAKE) fmt
	@echo "✅ ドメイン用のSQLCコード生成が完了しました。"

gen-query-qs:
	@echo "🔄 クエリサービス用のSQLCコードを生成します..."
	@$(MAKE) dump-schema
	@$(MAKE) merge-dml-qs
	@$(MAKE) gen-sqlc
	@$(MAKE) fmt
	@echo "✅ クエリサービス用のSQLCコード生成が完了しました。"

gen-query-sysq:
	@echo "🔄 システムクエリ用のSQLCコードを生成します..."
	@$(MAKE) dump-schema
	@$(MAKE) merge-dml-sysq
	@$(MAKE) gen-sqlc
	@$(MAKE) fmt
	@echo "✅ システムクエリ用のSQLCコード生成が完了しました。"
