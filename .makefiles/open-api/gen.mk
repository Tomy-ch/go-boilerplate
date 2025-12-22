## OpenAPI関連の生成コマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: gen-api-docs ## OpenAPIに基づき、APIドキュメントを生成します
.PHONY: gen-swagger ## OpenAPIをバインドルしてOpenAPIファイルを一つにまとめます
# -----CI用ターゲット-----
.PHONY: gen-swagger-ci ## OpenAPIをバインドルしてOpenAPIファイルを一つにまとめます（CI用）
.PHONY: gen-api-docs-ci ## OpenAPIに基づき、APIドキュメントを生成します（CI用）

# -----Dockerコンテナ内で実行するコマンド群-----
gen-swagger:
	@docker compose run --rm node_tool_runner make gen-swagger-ci

gen-api-docs:
	@docker compose run --rm node_tool_runner make gen-api-docs-ci

# -----CI用ターゲット-----
gen-swagger-ci:
	swagger-cli bundle openapi/openapi.yaml --type yaml -o openapi/openapi.gen.yaml

gen-api-docs-ci:
	redocly build-docs openapi/openapi.yaml --output /app/docs/openapi/index.html
