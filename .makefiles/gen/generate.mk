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
	docker compose run --rm go_tool_runner go generate ./...

gen-swagger:
	docker compose run --rm node_tool_runner swagger-cli bundle openapi/openapi.yaml --type yaml -o openapi/openapi.gen.yaml

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
	@make gen-sqlc-repo
	@make gen-sqlc-qs
	@make gen-sqlc-sysq
	@echo "✅ SQLCのコード生成が完了しました。"

gen-sqlc-repo:
	@echo "🔄 ドメイン用のSQLCのコード生成を行います..."; \
	docker compose run --rm go_tool_runner go run cmd/main.go gen-sqlc --type=repository; \
	echo "✅ ドメイン用のSQLCのコード生成が完了しました。"
	@make fmt

gen-sqlc-qs:
	@echo "🔄 クエリサービス用のSQLCのコード生成を行います..."; \
	docker compose run --rm go_tool_runner go run cmd/main.go gen-sqlc --type=query_service; \
	echo "✅ クエリサービス用のSQLCのコード生成が完了しました。"
	@make fmt

gen-sqlc-sysq:
	@echo "🔄 システムクエリ用のSQLCのコード生成を行います..."; \
	docker compose run --rm go_tool_runner go run cmd/main.go gen-sqlc --type=system_query; \
	echo "✅ システムクエリ用のSQLCのコード生成が完了しました。"
	@make fmt

gen-tools-meta:
	@echo "🔍 生成ツールのバージョン情報を出力します..."
	docker compose run --rm go_tool_runner sh scripts/gen_generator_versions.sh go
	docker compose run --rm node_tool_runner sh scripts/gen_generator_versions.sh node
	docker compose run --rm python_tool_runner sh scripts/gen_generator_versions.sh python
	@echo "✅ 生成ツールのバージョン情報の出力が完了しました。"
