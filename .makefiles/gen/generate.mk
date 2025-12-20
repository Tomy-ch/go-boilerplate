## 自動生成系

.PHONY: gen ## 各種ドキュメントやコードを生成します
.PHONY: gen-api ## API関連のドキュメントやコードを生成します
.PHONY: gen-doc ## ドキュメント関連の生成を行います
.PHONY: gen-swagger ## OpenAPIをバインドルして生成します
.PHONY: gen-go-code ## Goコードを生成します
.PHONY: gen-redoc ## RedocでOpenAPIドキュメントを生成します
.PHONY: gen-ctxkey ## Contextに値を格納するためのコードを生成する(nameとtypeを指定が必要)
.PHONY: gen-sqlc ## SQLCのコード生成を行う
.PHONY: gen-sqlc-repo ## ドメイン用のSQLCのコード生成を行う
.PHONY: gen-sqlc-qs ## クエリサービス用のSQLCのコード生成を行う
.PHONY: gen-sqlc-sysq ## システムクエリ用のSQLCのコード生成を行う
.PHONY: gen-tools-meta ## 生成ツールのバージョン情報を出力する

SQLC_OUT := /app/internal/infrastructure/rdb/sqlc/gen

gen-ctxkey:
	@if [ -z "$(name)" ] || [ -z "$(type)" ]; then \
	echo "❌ nameとtypeの引数が必要です。以下のように指定してください："; \
	echo "   make gen-ctxkey name=UserID type=string"; \
	exit 1; \
	fi; \
	bash scripts/gen_ctxkey.sh $(name) $(type)

gen:
	@echo "🔄 各種ドキュメントやコードの生成します..."
	@make gen-api
	@make gen-sqlc
	@make gen-doc
	@echo "✅ 各種ドキュメントやコードの生成が完了しました。"

gen-go-code:
	@docker compose run --rm go_tool_runner go generate ./...

gen-swagger:
	@docker compose run --rm node_tool_runner swagger-cli bundle openapi/openapi.yaml --type yaml -o openapi/openapi.gen.yaml

gen-api:
	@make gen-swagger
	@make gen-go-code

gen-doc:
	@make gen-redoc
	@make gen-tools-meta

gen-redoc:
	docker compose run --rm node_tool_runner redocly build-docs openapi/openapi.yaml --output /app/docs/openapi/index.html

gen-sqlc:
	@echo "🔄 SQLCのコードを生成します..."
	@make dump-schema
	@make merge-dml
	@make exc-sqlc
	@make fmt
	@echo "✅ SQLCのコード生成が完了しました。"

gen-sqlc-repo:
	@echo "🔄 ドメイン用のSQLCコードを生成します..."
	@make dump-schema
	@make merge-dml-repo
	@make exc-sqlc
	@echo "✅ ドメイン用のSQLCコード生成が完了しました。"

gen-sqlc-qs:
	@echo "🔄 クエリサービス用のSQLCコードを生成します..."
	@make dump-schema
	@make merge-dml-qs
	@make exc-sqlc
	@echo "✅ クエリサービス用のSQLCコード生成が完了しました。"

gen-sqlc-sysq:
	@echo "🔄 システムクエリ用のSQLCコードを生成します..."
	@make dump-schema
	@make merge-dml-sysq
	@make exc-sqlc
	@echo "✅ システムクエリ用のSQLCコード生成が完了しました。"

exc-sqlc:
	@docker compose run --rm go_tool_runner sh -lc '\
		rm -f $(SQLC_OUT)/*.gen.sql.go && \
		cd /app && sqlc generate -f sqlc.yaml'

gen-tools-meta:
	@echo "🔍 生成ツールのバージョン情報を出力します..."
	docker compose run --rm go_tool_runner sh scripts/gen_generator_versions.sh go
	docker compose run --rm node_tool_runner sh scripts/gen_generator_versions.sh node
	docker compose run --rm python_tool_runner sh scripts/gen_generator_versions.sh python
	@echo "✅ 生成ツールのバージョン情報の出力が完了しました。"
