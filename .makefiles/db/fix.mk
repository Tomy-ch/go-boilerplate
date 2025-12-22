## DBに対する修正コマンド群
# -----Dockerコンテナ内で実行するCI用ターゲット-----
.PHONY: fix-collation ## データベースのコラテーションを修正
# ----CI用ターゲット-----
.PHONY: fix-collation-ci ## データベースのコラテーションを修正（CI用）

# -----Dockerコンテナ内で実行するCI用ターゲット-----
fix-collation:
	@echo "🔄 データベースのコラテーションを修正します..."
	@docker compose run --rm go_tool_runner make fix-collation-ci
	@echo "✅ データベースのコラテーション修正が完了しました。"

# ----CI用ターゲット-----
fix-collation-ci:
	go run cmd/main.go fix-collation
