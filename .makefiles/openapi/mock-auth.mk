## mock-auth-server（疑似 OIDC Provider）の OpenAPI 生成コマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: gen-mock-auth-oapi ## mock-auth-server の OpenAPI をバンドルし zod スキーマを生成します
.PHONY: gen-mock-auth-oapi-docs ## mock-auth-server の OpenAPI から Redoc HTML を生成します
.PHONY: lint-mock-auth-oapi ## mock-auth-server の OpenAPI 定義を redocly lint で検証します
# -----CI用ターゲット-----
.PHONY: gen-mock-auth-oapi-ci ## mock-auth-server の OpenAPI をバンドルし zod スキーマを生成します（CI用）
.PHONY: gen-mock-auth-oapi-docs-ci ## mock-auth-server の OpenAPI から Redoc HTML を生成します（CI用）
.PHONY: lint-mock-auth-oapi-ci ## mock-auth-server の OpenAPI 定義を redocly lint で検証します（CI用）

# -----Dockerコンテナ内で実行するコマンド群-----
gen-mock-auth-oapi:
	@docker compose run --rm node_tool_runner make gen-mock-auth-oapi-ci

gen-mock-auth-oapi-docs:
	@docker compose run --rm node_tool_runner make gen-mock-auth-oapi-docs-ci

lint-mock-auth-oapi:
	@docker compose run --rm node_tool_runner make lint-mock-auth-oapi-ci

# -----CI用ターゲット-----
# orval は provider の devDeps ではなく node_tools 同梱のため、scripts の bin を PATH へ前置して解決する。
gen-mock-auth-oapi-ci:
	cd mock-auth-server && PATH="/app/scripts/node_modules/.bin:$$PATH" npm run gen

# Redoc は redocly（mise グローバル）で生成する。出力は repo の docs/openapi/mock-auth-server/ 配下。
gen-mock-auth-oapi-docs-ci:
	cd mock-auth-server && npm run gen:docs

lint-mock-auth-oapi-ci:
	cd mock-auth-server && PATH="/app/scripts/node_modules/.bin:$$PATH" npm run lint:oapi
