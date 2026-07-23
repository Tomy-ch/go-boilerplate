## DB スロットプール（複数 worktree の DB を per-port スロットで共有）
.PHONY: db-acquire ## DB スロットを取得しスキーマを作り直す（プールから per-port を貸与）
.PHONY: db-release ## 保持中の DB スロットを解放する（コンテナは warm 保持）
.PHONY: db-pool-status ## DB スロットプールの占有状況を表示する

# worktree が保持中のスロット定義（scripts/db-pool が書き出す KEY=VALUE）。
# 存在すれば DB_HOST_PORT / COMPOSE_PROJECT_NAME を取り込み、docker compose（ホスト公開ポート・
# プロジェクト分離）と host 実行の go test（接続ポート）へ伝播させる。未取得なら既定（ホスト 5432 /
# ディレクトリ由来プロジェクト）で従来どおり動作する。
# 注: コンテナ内部の接続ポート DB_PORT（既定 5432、env/.env 由来）は export しない。export すると
# go_tool_runner 内のアプリが内部 5432 ではなくホスト公開ポートへ繋ぎに行き接続不能になる。
-include .gobp-db-slot
export DB_HOST_PORT
ifneq ($(COMPOSE_PROJECT_NAME),)
export COMPOSE_PROJECT_NAME
endif

db-acquire:
	@bash scripts/db-pool/pool.sh acquire
	@echo "🔄 取得したスロットのスキーマを作り直します..."
	@set -a; . ./.gobp-db-slot; set +a; $(MAKE) db-local-reinit db-test-reinit
	@bash scripts/db-pool/pool.sh heartbeat
	@echo "✅ DB スロットを取得しました。以降の make（test / db-init など）は同一スロットを使います。"

db-release:
	@bash scripts/db-pool/pool.sh release

db-pool-status:
	@bash scripts/db-pool/pool.sh status
