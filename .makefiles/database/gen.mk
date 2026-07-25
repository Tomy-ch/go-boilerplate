## DBに対する生成コマンド群
# -----Dockerコンテナ内で実行するターゲット-----
.PHONY: gen-db-schema ## スキーマの更新を実行
.PHONY: dump-schema ## スキーマのダンプを実行
.PHONY: db-ensure ## 指定DBを無ければ作成し、拡張(pg_trgm)とtimezoneを初期化する
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

# dump-schema(ローカル) は共有 local DB を避け、当該ブランチの migration だけから再構築した
# 専用 DB($(SCHEMA_GEN_DB)) をダンプする。これにより並行する別 worktree の migration が生成物
# (schema.gen.sql / models.gen.go 等) へ混入しない（#657）。ローカル専用のガードであり、CI は
# fresh な postgres service を migrate 済みで dump-schema-ci を直接呼ぶため本経路は通らない。
SCHEMA_GEN_DB ?= gen_schema

dump-schema:
	@echo "🔄 スキーマのダンプを実行します（$(SCHEMA_GEN_DB) を migration から再構築してダンプ）..."
	@$(MAKE) db-ensure DB=$(SCHEMA_GEN_DB)
	@$(MAKE) db-drop-tables DB=$(SCHEMA_GEN_DB)
	@$(MAKE) db-migrate-up DB=$(SCHEMA_GEN_DB)
	@docker compose run --rm -e DB_NAME=$(SCHEMA_GEN_DB) go_tool_runner make dump-schema-ci work-dir="$(work-dir)"
	@echo "✅ スキーマのダンプが完了しました。"

# db-ensure: 指定 DB を無ければ作成し、拡張(pg_trgm)と timezone を初期化する。
# compose 初期化 SQL(docker/database/sql) と同じ拡張・timezone で使い捨てスキーマ DB をブートストラップ
# する（dbslot の PgxAdmin.SetupDatabase と等価）。DB コンテナが起動している前提（gen-query と同じ）。
db-ensure:
	@docker compose exec -T database psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$(DB)'" | grep -q 1 \
		|| docker compose exec -T database psql -U postgres -c "CREATE DATABASE $(DB)"
	@docker compose exec -T database psql -U postgres -c "ALTER DATABASE $(DB) SET timezone TO 'Asia/Tokyo';"
	@docker compose exec -T database psql -U postgres -d $(DB) -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;"

# ----CI用ターゲット-----
dump-schema-ci:
	go run ./cmd/ dump-schema --work-dir=$(work-dir)
