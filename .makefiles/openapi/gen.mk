## OpenAPI関連の生成コマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: gen-bundle-oapi ## OpenAPIをバンドルしてOpenAPIファイルを一つにまとめます
.PHONY: gen-api-docs ## OpenAPIに基づき、APIドキュメントを生成します
# -----CI用ターゲット-----
.PHONY: gen-bundle-oapi-ci ## OpenAPIをバンドルしてOpenAPIファイルを一つにまとめます（CI用）
.PHONY: gen-api-docs-ci ## OpenAPIに基づき、APIドキュメントを生成します（CI用）

# -----Dockerコンテナ内で実行するコマンド群-----
gen-bundle-oapi:
	@docker compose run --rm node_tool_runner make gen-bundle-oapi-ci

gen-api-docs:
	@docker compose run --rm node_tool_runner make gen-api-docs-ci

# -----CI用ターゲット-----
gen-bundle-oapi-ci:
	redocly bundle openapi/openapi.yaml -o openapi/openapi.gen.yaml

gen-api-docs-ci:
	redocly build-docs openapi/openapi.yaml --output /app/docs/openapi/index.html
