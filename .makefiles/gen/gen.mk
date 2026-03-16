## 生成関連のコマンドの一括実行コマンド群
.PHONY: gen ## 各種ドキュメントやコードを生成します
.PHONY: gen-api ## API関連のドキュメントやコードを生成します
.PHONY: gen-docs ## ドキュメント関連の生成を行います
.PHONY: gen-query ## SQLCのコード生成を行う
.PHONY: gen-query-repo ## ドメイン用のSQLCのコード生成を行う
.PHONY: gen-query-qs ## クエリサービス用のSQLCのコード生成を行う
.PHONY: gen-query-sysq ## システムクエリ用のSQLCのコード生成を行う

gen:
	@echo "🔄 各種ドキュメントやコードの生成します..."
	@make gen-api
	@make gen-query
	@make gen-docs
	@echo "✅ 各種ドキュメントやコードの生成が完了しました。"

gen-api:
	@make gen-swagger
	@make gen-go-code

gen-docs:
	@make gen-api-docs
	@make gen-tools-meta
	@make gen-docs-json
	@make gen-db-schema
	@make gen-test-repo

gen-query:
	@echo "🔄 SQLCのコードを生成します..."
	@make dump-schema
	@make merge-dml
	@make gen-sqlc
	@make fmt
	@echo "✅ SQLCのコード生成が完了しました。"

gen-query-repo:
	@echo "🔄 ドメイン用のSQLCコードを生成します..."
	@make dump-schema
	@make merge-dml-repo
	@make gen-sqlc
	@echo "✅ ドメイン用のSQLCコード生成が完了しました。"

gen-query-qs:
	@echo "🔄 クエリサービス用のSQLCコードを生成します..."
	@make dump-schema
	@make merge-dml-qs
	@make gen-sqlc
	@echo "✅ クエリサービス用のSQLCコード生成が完了しました。"

gen-query-sysq:
	@echo "🔄 システムクエリ用のSQLCコードを生成します..."
	@make dump-schema
	@make merge-dml-sysq
	@make gen-sqlc
	@echo "✅ システムクエリ用のSQLCコード生成が完了しました。"
