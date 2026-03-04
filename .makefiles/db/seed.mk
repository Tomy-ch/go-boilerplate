## DBに対するシードデータ投入のコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: db-seed ## データベースにシードデータを投入
# -----CI用ターゲット-----
.PHONY: db-seed-ci ## データベースにシードデータを投入（CI用）
# -----LocalDBに対してのシードデータ投入エイリアス-----
.PHONY: db-local-seed ## LocalDBに対してシードデータを投入
# -----TestDBに対してのシードデータ投入エイリアス-----
.PHONY: db-test-seed ## TestDBに対してシードデータを投入

# -----Dockerコンテナ内で実行するコマンド群-----
db-seed:
	@echo "🌱 データベースにシードデータを投入します... (database=$(DB))"
	@docker compose run --rm go_tool_runner make db-seed-ci DB=$(DB)
	@echo "✅ シードデータの投入が完了しました。 (database=$(DB))"

# -----CI用ターゲット-----
db-seed-ci:
	go run cmd/main.go db-seed --database $(DB)

# -----LocalDBに対してのシードデータ投入エイリアス-----
db-local-seed: DB=local
db-local-seed: db-seed

# -----TestDBに対してのシードデータ投入エイリアス-----
db-test-seed: DB=test
db-test-seed: db-seed
