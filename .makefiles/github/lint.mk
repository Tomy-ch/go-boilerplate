## GitHub Actions に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: actions-lint ## GitHub Actions 定義(ワークフロー)のLintを実行（actionlint + PRコメント本文の secret / フェンス検査）
.PHONY: actions-comment-secret-lint ## upsert-pr-comment を使うジョブへの secret 混入検査のみを実行
.PHONY: actions-comment-fence-lint ## PRコメント本文の固定長フェンス検査のみを実行
# -----CI内で実行するコマンド群-----
.PHONY: actions-lint-ci ## GitHub Actions 定義のLintを実行(CI用)
.PHONY: actions-actionlint-ci ## actionlint でワークフロー定義をLint(CI用)
.PHONY: actions-comment-secret-lint-ci ## upsert-pr-comment を使うジョブへの secret 混入を検査(CI用)
.PHONY: actions-comment-fence-lint-ci ## PRコメント本文の固定長フェンスを検査(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
# actionlint は Go ツール、secret / フェンス検査は node スクリプトなので tool-runner を跨ぐ。
actions-lint:
	@docker compose run --rm go_tool_runner make actions-actionlint-ci
	@docker compose run --rm node_tool_runner make actions-comment-secret-lint-ci
	@docker compose run --rm node_tool_runner make actions-comment-fence-lint-ci

actions-comment-secret-lint:
	@docker compose run --rm node_tool_runner make actions-comment-secret-lint-ci

actions-comment-fence-lint:
	@docker compose run --rm node_tool_runner make actions-comment-fence-lint-ci

# -----CI内で実行するコマンド群-----
actions-lint-ci: actions-actionlint-ci actions-comment-secret-lint-ci actions-comment-fence-lint-ci

actions-actionlint-ci:
	actionlint

# actionlint は「upsert-pr-comment を使うジョブに secret を渡すな」という規約を表現できない。
actions-comment-secret-lint-ci:
	node scripts/pr-comment-secret-lint.mjs

# actionlint は「PR コメント本文のフェンスを固定長にするな」という規約を表現できない。
actions-comment-fence-lint-ci:
	node scripts/pr-comment-fence-lint.mjs
