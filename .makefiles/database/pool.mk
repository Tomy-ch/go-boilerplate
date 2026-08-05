## DB スロットプール（複数 worktree が単一共有 DB を per-worktree データベースで共有）
.PHONY: slot-acquire ## DB スロットを取得しスキーマを作り直す（共有 DB 上に per-worktree DB を貸与）
.PHONY: slot-free ## 保持中の DB スロットだけを解放する（worktree は残す。DB は warm 保持）
.PHONY: slot-release ## worktree を撤収する（app 停止+イメージ削除 → スロット解放 → worktree 削除）
.PHONY: slot-status ## DB スロットプールの占有状況を表示する
.PHONY: require-db-owner ## 自分が所有するデータベースがあることを検証する（DB を触るターゲットの前提）

# ---- 不変条件: データベース : worktree = 1 : 0..1 --------------------------------
# 1 つのデータベースを 2 箇所から触らせない。worktree 側はスロットを取っても取らなくてもよいが、
# 取らなかったときの所有データベースは「既定の local / test」ではなく「無し」である。
#
#   主 checkout               local / test / gen_schema
#   スロット取得済み worktree    wt<N>_local / wt<N>_test / gen_schema_wt<N>
#   スロット未取得の worktree    無し（DB を触るターゲットは require-db-owner で失敗する）
#
# 未取得の worktree を既定値へフォールバックさせると、主 checkout のデータベースを 2 箇所から
# 触ることになる。しかもフォールバックは黙って成功するため、別ブランチの migration が混ざった
# データベースでテストが緑になる・生成物が壊れる、という気づけない壊れ方をする。所有者が居ない
# なら既定値へ落とさず止める。
# ------------------------------------------------------------------------------

# スロット定義（db-slot が書き出す KEY=VALUE）。取得時だけ生成され、DB 名・ホスト公開ポート・
# app 層の compose プロジェクト名を上書きする。既定値は docker-compose.attach.yaml 側の
# ${VAR:-...}（ホスト公開ポート 8080/2010/2345/6060）と docker/compose.mk の APP_PROJECT_DEFAULT。
# ここでの -include はホスト実行の go test へ DB 名を渡すためのもので、app 層の compose 呼び出しは
# パース時では取りこぼすため LOAD_SLOT がレシピ内で読み直す（docker/compose.mk 参照）。
-include .gobp-db-slot
export DB_NAME_LOCAL
export DB_NAME_TEST
export SERVE_PROJECT

# git-dir と git-common-dir はリンク worktree でだけ食い違う（主 checkout はどちらも .git、git 管理外や
# ツールランナーのマウント内は両方空）。GIT_DIRS の実体は docker/compose.mk が持つ。
IS_LINKED_WORKTREE := $(if $(filter-out $(word 2,$(GIT_DIRS)),$(word 1,$(GIT_DIRS))),1,)

# 所有データベースの名前。local / test の所有者は主 checkout ただ 1 つで、未取得の worktree が
# ここへ落ちてこないことは require-db-owner が保証する。再帰展開なので、これらを参照する
# migrate.mk / seed.mk / gen.mk が本ファイルより前に include されていても解決順は問題にならない。
DB_LOCAL = $(if $(DB_NAME_LOCAL),$(DB_NAME_LOCAL),local)
DB_TEST = $(if $(DB_NAME_TEST),$(DB_NAME_TEST),test)

# 所有データベースを持たない状態（リンク worktree かつスロット未取得）を検出して止める。
# 主 checkout・ツールランナー内・CI は IS_LINKED_WORKTREE が空になるため素通りする。
require-db-owner:
	@test -z "$(IS_LINKED_WORKTREE)" || test -n "$(DB_NAME_LOCAL)" || { \
		echo "❌ この worktree は DB スロットを取得していないため、所有するデータベースがありません。"; \
		echo "   1 つのデータベースを複数の checkout から触らないよう、既定の local / test へは"; \
		echo "   フォールバックしません。"; \
		echo "   → make slot-acquire でスロットを取得してください（make slot-status で空きを確認）。"; \
		exit 1; }

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
