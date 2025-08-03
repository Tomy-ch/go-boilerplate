## 自動生成系

.PHONY: gen ## 各種ドキュメントやコードを生成します
.PHONY: gen-swagger ## OpenAPIをバインドルして生成します
.PHONY: gen-golang-code ## Goコードを生成します
.PHONY: gen-redoc ## RedocでOpenAPIドキュメントを生成します
.PHONY: gen-ctxkey ## Contextに値を格納するためのコードを生成する(nameとtypeを指定が必要)

gen-ctxkey:
	@if [ -z "$(name)" ] || [ -z "$(type)" ]; then \
	echo "❌ nameとtypeの引数が必要です。以下のように指定してください："; \
	echo "   make gen-ctxkey name=UserID type=string"; \
	exit 1; \
	fi; \
	bash scripts/gen_ctxkey.sh $(name) $(type)

gen:
	@echo "🔄 各種ドキュメントやコードの生成します..."
	@make gen-swagger
	@make gen-redoc
	@make gen-golang-code
	@echo "✅ 各種ドキュメントやコードの生成が完了しました。"

gen-golang-code:
	docker compose run --rm go_tool_runner go generate ./...

gen-swagger:
	docker compose run --rm node_tool_runner swagger-cli bundle openapi/openapi.yaml --type yaml -o openapi/openapi.gen.yaml

gen-redoc:
	docker compose run --rm node_tool_runner redocly build-docs openapi/openapi.yaml --output /app/docs/openapi/index.html
