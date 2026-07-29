## OpenAPI関連の生成コマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: gen-bundle-oapi ## OpenAPIをバンドルしてOpenAPIファイルを一つにまとめます
.PHONY: gen-api-docs ## OpenAPIに基づき、APIドキュメントを生成します
.PHONY: lint-oapi ## OpenAPI定義を redocly lint で検証します
# -----CI用ターゲット-----
.PHONY: gen-bundle-oapi-ci ## OpenAPIをバンドルしてOpenAPIファイルを一つにまとめます（CI用）
.PHONY: gen-api-docs-ci ## OpenAPIに基づき、APIドキュメントを生成します（CI用）
.PHONY: lint-oapi-ci ## OpenAPI定義を redocly lint で検証します（CI用）
.PHONY: lint-oapi-security-ci ## OpenAPI定義を OWASP API Security ルールセットで検証します（CI用）

# -----Dockerコンテナ内で実行するコマンド群-----
gen-bundle-oapi:
	@docker compose run --rm node_tool_runner make gen-bundle-oapi-ci

gen-api-docs:
	@docker compose run --rm node_tool_runner make gen-api-docs-ci

lint-oapi:
	@docker compose run --rm node_tool_runner make lint-oapi-ci

# -----CI用ターゲット-----
gen-bundle-oapi-ci:
	redocly bundle openapi/openapi.yaml -o openapi/openapi.gen.yaml

gen-api-docs-ci:
	redocly build-docs openapi/openapi.yaml --output docs/openapi/index.html

lint-oapi-ci:
	redocly lint openapi/openapi.yaml

# OWASP API Security ルールセットによる検証。redocly lint（規約・命名・メタデータ）とは担当が
# 異なり、指摘は重複しない。
#
# node_tool_runner を経由しないのは、spectral の ruleset が npm パッケージで、Spectral が
# extends を .spectral.yaml の位置から node の解決規則で探すため。コンテナは依存を
# /app/scripts/node_modules へ置き、CI とホストは docker/tools/node_modules へ置くので、
# 両方で成立する単一の extends パスが書けない。CI 専用スキャナ（zizmor / osv-scanner /
# trufflehog）と同じく、コンテナを介さず直接実行する方式へ寄せる。
#
# 事前に `cd docker/tools && npm ci --ignore-scripts` が必要。
lint-oapi-security-ci:
	./docker/tools/node_modules/.bin/spectral lint openapi/openapi.yaml --ruleset .spectral.yaml
