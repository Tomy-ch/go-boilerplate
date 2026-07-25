## DB スロットプール（複数 worktree が単一共有 DB を per-worktree データベースで共有）
.PHONY: slot-acquire ## DB スロットを取得しスキーマを作り直す（共有 DB 上に per-worktree DB を貸与）
.PHONY: slot-free ## 保持中の DB スロットだけを解放する（worktree は残す。DB は warm 保持）
.PHONY: slot-release ## worktree を撤収する（app 停止+イメージ削除 → スロット解放 → worktree 削除）
.PHONY: slot-status ## DB スロットプールの占有状況を表示する

# スロット定義（db-slot が書き出す KEY=VALUE）。既定値は docker-compose.attach.yaml 側の
# ${VAR:-...}（DB 名 local/test・ホスト公開ポート 8080/4000/2345/6060）と docker/compose.mk の
# APP_PROJECT_DEFAULT が持ち、スロット取得時だけこのファイルが上書きする。未取得でも既定のまま
# 動くため、スロット取得は並列作業のための opt-in に留まる。
# ここでの -include はホスト実行の go test へ DB 名を渡すためのもので、app 層の compose 呼び出しは
# パース時では取りこぼすため LOAD_SLOT がレシピ内で読み直す（docker/compose.mk 参照）。
# DB_NAME_LOCAL/TEST を読むのは slot-acquire の reinit と host 実行の go test のみで、db-init/db-migrate-up は
# 今も DB=local/test 直書き（取得後に手で db-init すると共有 DB の local/test を触る点に注意）。
-include .gobp-db-slot
export DB_NAME_LOCAL
export DB_NAME_TEST
export SERVE_PROJECT

slot-acquire:
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

slot-free:
	@go run ./cmd/ db-slot release

# 撤収は docker → slot → git の順で行う。slot-free は .gobp-db-slot を消し、そこで
# SERVE_PROJECT が失われて APP_PROJECT が gobp-app-<dir> へフォールバックするため、
# app の停止・イメージ削除より先に解放すると別プロジェクトを対象にしてしまう。
# git worktree remove は cwd ごと消すので必ず最後に置く（後続レシピは cwd 消失で失敗する）。
# 未コミット・未追跡ファイルがあれば git 自身が拒否するため --force は付けない。
slot-release:
	@test "$$(git rev-parse --git-dir)" != "$$(git rev-parse --git-common-dir)" \
		|| { echo "❌ 主 checkout では実行できません（撤収対象の worktree で実行してください）"; exit 1; }
	@$(COMPOSE_APP) down --rmi local
	@if [ -f .gobp-db-slot ]; then $(MAKE) slot-free; fi
	@echo "🧹 worktree を削除します: $(CURDIR)"
	@git worktree remove $(CURDIR)

slot-status:
	@go run ./cmd/ db-slot status
