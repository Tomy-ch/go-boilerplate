## SQLCの生成コマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: gen-sqlc ## SQLCのコード生成を行います
.PHONY: remove-generated-sqlc ## 既存の生成済みSQLCコードを削除します
.PHONY: sqlc-generate ## SQLCのコード生成を実行します
# -----CI用ターゲット-----
.PHONY: remove-generated-sqlc-ci ## 既存の生成済みSQLCコードを削除します（CI用）
.PHONY: sqlc-generate-ci ## SQLCのコード生成を実行します（CI用）

# -----Dockerコンテナ内で実行するコマンド群-----
gen-sqlc:
	@$(MAKE) remove-generated-sqlc
	@$(MAKE) sqlc-generate

remove-generated-sqlc:
	@docker compose run --rm go_tool_runner make remove-generated-sqlc-ci

sqlc-generate:
	@docker compose run --rm go_tool_runner make sqlc-generate-ci

# -----CI用ターゲット-----
remove-generated-sqlc-ci:
	rm -f $(SQLC_OUT)/*.gen.sql.go

sqlc-generate-ci:
	sqlc generate -f sqlc.yaml
