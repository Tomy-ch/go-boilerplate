## DBに対する生成コマンド群
# -----Dockerコンテナ内で実行するターゲット-----
.PHONY: db-schema ## スキーマの更新を実行
.PHONY: dump-schema ## スキーマのダンプを実行
# ----CI用ターゲット-----
.PHONY: dump-schema-ci ## スキーマのダンプを実行（CI用）

# -----Dockerコンテナ内で実行するターゲット-----
db-schema:
	@echo "🔄 スキーマの更新を実行します..."
	docker compose run --rm er_diagram_generator
	@echo "✅ スキーマの更新が完了しました。"

dump-schema:
	@echo "🔄 スキーマのダンプを実行します..."
	@docker compose run --rm go_tool_runner make dump-schema-ci work-dir="$(work-dir)"
	@echo "✅ スキーマのダンプが完了しました。"

# ----CI用ターゲット-----
dump-schema-ci:
	go run cmd/main.go dump-schema --work-dir=$(work-dir)
