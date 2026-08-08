## OpenAPI関連の生成コマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: gen-bundle-oapi ## OpenAPIをバンドルしてOpenAPIファイルを一つにまとめます
.PHONY: gen-api-docs ## OpenAPIに基づき、APIドキュメントを生成します
.PHONY: lint-oapi ## OpenAPI定義を redocly lint で検証します
.PHONY: stamp-openapi-version ## リリースブランチ名(REF=release/vX.Y.Z)から info.version を書き換えます
# -----CI用ターゲット-----
.PHONY: stamp-openapi-version-ci ## リリースブランチ名から info.version を書き換えます（CI用）
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

# REF 未指定なら環境変数 GITHUB_REF_NAME を使う。コンテナへは環境変数が渡らないため、
# 呼び出し側が読んだ値を make 変数として内側へ引き継ぐ。
stamp-openapi-version:
	@docker compose run --rm node_tool_runner make stamp-openapi-version-ci REF="$(or $(REF),$(GITHUB_REF_NAME))"

# -----CI用ターゲット-----
gen-bundle-oapi-ci:
	redocly bundle openapi/openapi.yaml -o openapi/openapi.gen.yaml

gen-api-docs-ci:
	redocly build-docs openapi/openapi.yaml --output docs/openapi/index.html

lint-oapi-ci:
	redocly lint openapi/openapi.yaml

# release/vX.Y.Z 以外の ref は no-op（スキップして正常終了）。
stamp-openapi-version-ci:
	$(TSX) scripts/stamp-openapi-version $(REF)

# OWASP API Security ルールセットによる検証。redocly lint（規約・命名・メタデータ）とは担当が
# 異なり、指摘は重複しない。
#
# spec だけを見る検査のためにツールランナーのイメージを起こさない。コンテナを介さず直接実行する
# スキャナ（zizmor / osv-scanner / trufflehog）と同じ方式に寄せている。
#
# 事前に `pnpm install --dir scripts --frozen-lockfile` が必要。
lint-oapi-security-ci:
	$(PNPM_SCRIPTS) lint:openapi-security
