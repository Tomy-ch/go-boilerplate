## Closed Loop（AI 開発フィードバック）のコマンド群
#
# 打刻は .agents/closed-loop/marks.sh が hook / スキル / git フックから行う。ここは読む側と送る側。
#
# closed-loop-report だけがコンテナ実行で、send / weekly はホスト実行になっている。理由は見る先が
# 違うため。report が読む打刻はリポジトリ内（マウント済み）にあるが、send が読むトランスクリプトは
# 利用者のホーム配下にあり、weekly が叩く gh は利用者の認証を要る。どちらもコンテナからは届かない。
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: closed-loop-report ## 打刻された開発の窓のフェーズ区間と異常を報告する
# -----ホストで実行するコマンド群-----
.PHONY: closed-loop-send ## 閉じたが未送出の窓を Feedback Issue へ送る
.PHONY: closed-loop-send-dry ## 送出せずに、送る内容だけを表示する
.PHONY: closed-loop-weekly ## 期間内の Feedback Issue を集計し検討課題を並べる（FROM= TO= で期間指定）
# -----CI内で実行するコマンド群-----
.PHONY: closed-loop-report-ci ## 開発の窓を報告する（CI用）

# -----Dockerコンテナ内で実行するコマンド群-----
closed-loop-report:
	@docker compose run --rm node_tool_runner make closed-loop-report-ci

# -----ホストで実行するコマンド群-----
closed-loop-send:
	@.agents/closed-loop/send.sh

closed-loop-send-dry:
	@.agents/closed-loop/send.sh --dry-run

closed-loop-weekly:
	@scripts/node_modules/.bin/tsx scripts/closed-loop/weekly \
		$(if $(FROM),--from $(FROM)) $(if $(TO),--to $(TO))

# -----CI内で実行するコマンド群-----
closed-loop-report-ci:
	$(PNPM_SCRIPTS) tsx scripts/closed-loop
