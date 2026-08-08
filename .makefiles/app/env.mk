## 埋め込み用 env 材料化関連
.PHONY: materialize-env ## 埋め込み対象 env/.env を APP_ENV 環境の値で材料化する（CI/ビルド用）
.PHONY: restore-env ## 材料化で書き換えた env/.env を git 管理の内容へ復元する

# 材料化対象の環境（ci / dast / dev / stg / prd）。未指定なら ci
APP_ENV ?= ci

materialize-env:
	@cp env/.env.$(APP_ENV) env/.env

restore-env:
	@git restore env/.env
