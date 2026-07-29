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
# seed の ${AUTH_ISSUER} は投入先で発行されるトークンの iss と一致していなければならない。ツールランナーは
# env/.env（4000 固定）を読むため、スロット保持時のホスト公開ポートから導いた値（AUTH_ISSUER_SH）を渡し、
# スロットの有無に関わらず実機で認証済み経路を叩ける identity を投入する。
db-seed:
	@echo "🌱 データベースにシードデータを投入します... (database=$(DB))"
	@$(LOAD_SLOT); docker compose run --rm \
		-e AUTH_ISSUER=$(AUTH_ISSUER_SH) \
		go_tool_runner make db-seed-ci DB=$(DB)
	@echo "✅ シードデータの投入が完了しました。 (database=$(DB))"

# -----CI用ターゲット-----
# AUTH_ISSUER の注入は呼び出し元（db-seed / CI の env 材料化）が持つ。スロット保持中にこのターゲットを
# 単独で叩くと env/.env の既定値（4000）で投入され、mock 認証サーバーのポートと食い違う。
db-seed-ci:
	go run ./cmd/ db-seed --database $(DB)

# -----LocalDBに対してのシードデータ投入エイリアス-----
db-local-seed: DB=local
db-local-seed: db-seed

# -----TestDBに対してのシードデータ投入エイリアス-----
db-test-seed: DB=test
db-test-seed: db-seed
