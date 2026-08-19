## Closed Loop（AI 開発フィードバック）の集計コマンド群
#
# 打刻は .agents/closed-loop/marks.sh が hook / スキル / git フックから行い、ここは読む側だけ。
# 判定を持たない入口なので、ローカル用と CI 用の 2 段は他のツールと同じ形にしてある。
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: closed-loop-report ## 打刻された開発の窓のフェーズ区間と異常を報告する
# -----CI内で実行するコマンド群-----
.PHONY: closed-loop-report-ci ## 開発の窓を報告する（CI用）

# -----Dockerコンテナ内で実行するコマンド群-----
closed-loop-report:
	@docker compose run --rm node_tool_runner make closed-loop-report-ci

# -----CI内で実行するコマンド群-----
closed-loop-report-ci:
	$(PNPM_SCRIPTS) tsx scripts/closed-loop
