## DBに対する生成コマンド群
# -----Dockerコンテナ内で実行するターゲット-----
.PHONY: gen-db-schema ## スキーマの更新を実行
.PHONY: dump-schema ## スキーマのダンプを実行
# ----CI用ターゲット-----
.PHONY: gen-db-schema-ci ## DBスキーマの生成を実行（CI用）
.PHONY: dump-schema-ci ## スキーマのダンプを実行（CI用）

# -----Dockerコンテナ内で実行するターゲット-----
gen-db-schema:
	@echo "🔄 スキーマの更新を実行します..."
	docker compose run --rm er_diagram_generator
	@echo "✅ スキーマの更新が完了しました。"

gen-db-schema-ci:
	docker run --rm \
		--network host \
		-u $(id -u):$(id -g) \
		-v $(PWD)/docs/db-schema:/output \
		-v $(PWD)/docker/database/schemaspy/schemaspy-ci.properties:/schemaspy.properties \
		schemaspy/schemaspy:latest \
		-configFile /schemaspy.properties

dump-schema:
	@echo "🔄 スキーマのダンプを実行します..."
	@docker compose run --rm go_tool_runner make dump-schema-ci work-dir="$(work-dir)"
	@echo "✅ スキーマのダンプが完了しました。"

# ----CI用ターゲット-----
dump-schema-ci:
	go run cmd/main.go dump-schema --work-dir=$(work-dir)
