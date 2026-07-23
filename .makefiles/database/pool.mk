## DB スロットプール（複数 worktree が単一共有 DB を per-worktree データベースで共有）
.PHONY: db-acquire ## DB スロットを取得しスキーマを作り直す（共有 DB 上に per-worktree DB を貸与）
.PHONY: db-release ## 保持中の DB スロットを解放する（データベースは warm 保持）
.PHONY: db-pool-status ## DB スロットプールの占有状況を表示する

# スロット定義（db-pool が書き出す KEY=VALUE）。COMPOSE_PROJECT_NAME=gobp-shared は docker compose を
# 呼ぶ全ターゲットに効き DB ツーリングを共有 DB へ向ける（serve は server.mk が SERVE_PROJECT で上書き）。
# DB_NAME_LOCAL/TEST を読むのは db-acquire の reinit と host 実行の go test のみで、db-init/db-migrate-up は
# 今も DB=local/test 直書き（取得後に手で db-init すると共有 DB の local/test を触る点に注意）。
# 未取得なら従来動作（opt-in）。
-include .gobp-db-slot
export DB_NAME_LOCAL
export DB_NAME_TEST
export API_HOST_PORT
export MOCK_AUTH_HOST_PORT
export SERVE_PROJECT
ifneq ($(COMPOSE_PROJECT_NAME),)
export COMPOSE_PROJECT_NAME
endif

db-acquire:
	@go run ./cmd/ db-slot acquire
	@echo "🔄 取得したスロットのデータベースを作り直します..."
	@# DB 名は .gobp-db-slot を実行時に source して得る（$(...) の make 変数は parse 時評価で、
	@# 初回取得時はまだファイルが無く空になるため、シェル変数 $$DB_NAME_* を使う）。
	@# local / test は共通 prerequisite db-reinit を持つため 1 つの make 呼び出しにまとめると
	@# db-reinit が 1 度しか実行されない。個別の make 呼び出しに分け、各 worktree DB を作り直す。
	@set -a; . ./.gobp-db-slot; set +a; \
		$(MAKE) db-reinit DB=$$DB_NAME_LOCAL && $(MAKE) db-reinit DB=$$DB_NAME_TEST
	@go run ./cmd/ db-slot heartbeat
	@echo "✅ DB スロットを取得しました。make test は自 worktree DB(wt<N>_test)、make serve は共有 DB の wt<N>_local を使います。"

db-release:
	@go run ./cmd/ db-slot release

db-pool-status:
	@go run ./cmd/ db-slot status
