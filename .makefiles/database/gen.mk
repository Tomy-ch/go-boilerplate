## DBに対する生成コマンド群
# -----Dockerコンテナ内で実行するターゲット-----
.PHONY: gen-db-schema ## スキーマの更新を実行
.PHONY: dump-schema ## スキーマのダンプを実行
.PHONY: db-ensure ## 指定DBを無ければ作成し、拡張(pg_trgm)を初期化する
# ----CI用ターゲット-----
.PHONY: gen-db-schema-ci ## DBスキーマの生成を実行（CI用）
.PHONY: dump-schema-ci ## スキーマのダンプを実行（CI用）

# -----Dockerコンテナ内で実行するターゲット-----
# 注: gen-db-schema(ローカル) と gen-db-schema-ci(CI) は同名だが別処理・別出力。
#   ローカル = er_diagram_generator(compose) → docs/er-diagram
#   CI       = schemaspy(raw docker)        → docs/db-schema
gen-db-schema:
	@echo "🔄 スキーマの更新を実行します..."
	docker compose run --rm er_diagram_generator
	@echo "✅ スキーマの更新が完了しました。"

gen-db-schema-ci:
	docker run --rm \
		--network host \
		-u $(shell id -u):$(shell id -g) \
		-v $(PWD)/docs/db-schema:/output \
		-v $(PWD)/docker/database/schemaspy/schemaspy-ci.properties:/schemaspy.properties \
		schemaspy/schemaspy:latest \
		-configFile /schemaspy.properties

# dump-schema(ローカル) は作業用データベースを避け、当該ブランチの migration だけから再構築した
# 使い捨て DB($(SCHEMA_GEN_DB)) をダンプする。所有者ごとに 1 つ持つ理由まで含め
# docs/maintenance/db-worktree-pool.md「schema-generation isolation」参照。
# ローカル専用のガードで、CI は fresh な postgres service に対し dump-schema-ci を直接呼ぶ。
SCHEMA_GEN_DB ?= $(if $(SLOT),gen_schema_wt$(SLOT),gen_schema)

dump-schema: require-db-owner
	@echo "🔄 スキーマのダンプを実行します（$(SCHEMA_GEN_DB) を migration から再構築してダンプ）..."
	@$(MAKE) db-ensure DB=$(SCHEMA_GEN_DB)
	@$(MAKE) db-drop-tables DB=$(SCHEMA_GEN_DB)
	@$(MAKE) db-migrate-up DB=$(SCHEMA_GEN_DB)
	@docker compose run --rm -e DB_NAME=$(SCHEMA_GEN_DB) go_tool_runner make dump-schema-ci work-dir="$(work-dir)"
	@echo "✅ スキーマのダンプが完了しました。"

# compose 初期化 SQL(docker/database/sql) と同じ拡張で使い捨てスキーマ DB をブートストラップする
# （dbslot の PgxAdmin.SetupDatabase と等価）。timezone は DB コンテナの TZ 由来のクラスタ既定を
# そのまま継承するため、ここでは設定しない。DB コンテナが起動している前提（gen-query と同じ）。
db-ensure: require-db-owner
	@docker compose exec -T database psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$(DB)'" | grep -q 1 \
		|| docker compose exec -T database psql -U postgres -c "CREATE DATABASE $(DB)"
	@docker compose exec -T database psql -U postgres -d $(DB) -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;"

# ----CI用ターゲット-----
dump-schema-ci:
	go run ./cmd/ dump-schema --work-dir=$(work-dir)
