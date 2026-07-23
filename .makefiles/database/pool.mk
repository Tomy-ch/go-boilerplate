## DB スロットプール（複数 worktree が単一共有 DB を per-worktree データベースで共有）
.PHONY: db-acquire ## DB スロットを取得しスキーマを作り直す（共有 DB 上に per-worktree DB を貸与）
.PHONY: db-release ## 保持中の DB スロットを解放する（データベースは warm 保持）
.PHONY: db-pool-status ## DB スロットプールの占有状況を表示する

# worktree が保持中のスロット定義（scripts/db-pool が書き出す KEY=VALUE）。存在すれば取り込む。
# - COMPOSE_PROJECT_NAME=共有 DB プロジェクト（gobp-shared）。export は make プロセス全体に効くため、
#   docker compose を呼ぶ全ターゲット（DB ツーリング migrate/seed/psql/gen だけでなく lint/gen 等の
#   tool_runner 系も）がこのプロジェクトに紐づく。tool_runner はソースを焼き込まないため実害はないが、
#   DB ツーリングが共有 DB コンテナ（ホスト 5432）を確実に指すのが本来の目的。serve は server.mk が
#   SERVE_PROJECT で上書きし、app コンテナだけ worktree 毎に分離する。
# - DB_NAME_LOCAL / DB_NAME_TEST = この worktree のデータベース名（wt<N>_local / wt<N>_test）。
#   実際にこれを読むのは db-acquire の reinit（$$DB_NAME_*）と host 実行の go test（DB_NAME_TEST）のみ。
#   db-init / db-migrate-up 等の既存 DB ターゲットは今も DB=local・test を直書きするため DB 名を切り替えない
#   （＝プール取得後に手で db-init すると共有 DB の local/test を触る点に注意。作り直しは db-acquire を使う）。
# 未取得なら既定（既定プロジェクト / DB 名 local・test / ホスト 8080・4000）で従来動作（opt-in）。
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
	@go run ./cmd/ db-pool acquire
	@echo "🔄 取得したスロットのデータベースを作り直します..."
	@# DB 名は .gobp-db-slot を実行時に source して得る（$(...) の make 変数は parse 時評価で、
	@# 初回取得時はまだファイルが無く空になるため、シェル変数 $$DB_NAME_* を使う）。
	@# local / test は共通 prerequisite db-reinit を持つため 1 つの make 呼び出しにまとめると
	@# db-reinit が 1 度しか実行されない。個別の make 呼び出しに分け、各 worktree DB を作り直す。
	@set -a; . ./.gobp-db-slot; set +a; \
		$(MAKE) db-reinit DB=$$DB_NAME_LOCAL && $(MAKE) db-reinit DB=$$DB_NAME_TEST
	@go run ./cmd/ db-pool heartbeat
	@echo "✅ DB スロットを取得しました。make test は自 worktree DB(wt<N>_test)、make serve は共有 DB の wt<N>_local を使います。"

db-release:
	@go run ./cmd/ db-pool release

db-pool-status:
	@go run ./cmd/ db-pool status
