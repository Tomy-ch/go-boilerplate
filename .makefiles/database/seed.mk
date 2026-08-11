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
# seed の ${AUTH_ISSUER} をトークンの iss と一致させる理由は
# docs/maintenance/db-worktree-pool.md「persisted data that follows the shifted ports」参照。
# ツールランナーは env/.env（2010 固定）を読むため、db-slot が導いた値を明示的に渡す。
db-seed: require-db-owner
	@echo "🌱 データベースにシードデータを投入します... (database=$(DB))"
	@$(DB_SLOT_ENV); docker compose run --rm \
		-e AUTH_ISSUER="$$AUTH_ISSUER" \
		go_tool_runner make db-seed-ci DB=$(DB)
	@echo "✅ シードデータの投入が完了しました。 (database=$(DB))"

# -----CI用ターゲット-----
# AUTH_ISSUER の注入は呼び出し元（db-seed / CI の env 材料化）が持つ。スロット保持中にこのターゲットを
# 単独で叩くと env/.env の既定値（2010）で投入され、mock 認証サーバーのポートと食い違う。
db-seed-ci:
	go run ./cmd/ db-seed --database $(DB)

# -----LocalDBに対してのシードデータ投入エイリアス-----
db-local-seed: DB=$(DB_LOCAL)
db-local-seed: db-seed

# -----TestDBに対してのシードデータ投入エイリアス-----
db-test-seed: DB=$(DB_TEST)
db-test-seed: db-seed
