## DB スロットプール（複数 worktree の DB を per-port スロットで共有）
.PHONY: db-acquire ## DB スロットを取得しスキーマを作り直す（プールから per-port を貸与）
.PHONY: db-release ## 保持中の DB スロットを解放する（コンテナは warm 保持）
.PHONY: db-pool-status ## DB スロットプールの占有状況を表示する

# worktree が保持中のスロット定義（scripts/db-pool が書き出す KEY=VALUE）。
# 存在すれば各 *_HOST_PORT / COMPOSE_PROJECT_NAME を取り込み、docker compose（ホスト公開ポート・
# プロジェクト分離、make serve の API/mock_auth ポート含む）と host 実行の go test（接続ポート）へ
# 伝播させる。未取得なら既定（ホスト 5432 / 8080 / 4000 / ディレクトリ由来プロジェクト）で従来動作。
# 注: コンテナ内部の接続ポート（DB_PORT=5432 等、env/.env 由来）は export しない。export すると
# コンテナ内アプリが内部ポートではなくホスト公開ポートへ繋ぎに行き接続不能になる。
-include .gobp-db-slot
export DB_HOST_PORT
export API_HOST_PORT
export MOCK_AUTH_HOST_PORT
ifneq ($(COMPOSE_PROJECT_NAME),)
export COMPOSE_PROJECT_NAME
endif

db-acquire:
	@bash scripts/db-pool/pool.sh acquire
	@echo "🔄 取得したスロットのスキーマを作り直します..."
	@# local と test は共通 prerequisite db-reinit を持つため 1 つの make 呼び出しにまとめると
	@# db-reinit が 1 度しか実行されず test 側がスキップされる。個別の make 呼び出しに分ける。
	@set -a; . ./.gobp-db-slot; set +a; $(MAKE) db-local-reinit && $(MAKE) db-test-reinit
	@bash scripts/db-pool/pool.sh heartbeat
	@echo "✅ DB スロットを取得しました。以降の make（test / db-init など）は同一スロットを使います。"

db-release:
	@bash scripts/db-pool/pool.sh release

db-pool-status:
	@bash scripts/db-pool/pool.sh status
