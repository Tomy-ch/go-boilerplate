## GitHub Actions に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: actions-lint ## GitHub Actions 定義(ワークフロー)のLintを実行（actionlint + composite action の run シェル検査 + PRコメント本文の secret 検査）
.PHONY: actions-shellcheck ## composite action の run シェルを shellcheck で検査
.PHONY: actions-comment-secret-lint ## upsert-pr-comment を使うジョブへの secret 混入検査のみを実行
# -----CI内で実行するコマンド群-----
.PHONY: actions-lint-ci ## GitHub Actions 定義のLintを実行(CI用)
.PHONY: actions-actionlint-ci ## actionlint でワークフロー定義をLint(CI用)
.PHONY: actions-shellcheck-ci ## composite action の run シェルを shellcheck で検査(CI用)
.PHONY: actions-comment-secret-lint-ci ## upsert-pr-comment を使うジョブへの secret 混入を検査(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
# actionlint は Go ツール、secret 検査は node スクリプトなので tool-runner を跨ぐ。
actions-lint:
	@docker compose run --rm go_tool_runner make actions-actionlint-ci
	@docker compose run --rm go_tool_runner make actions-shellcheck-ci
	@docker compose run --rm node_tool_runner make actions-comment-secret-lint-ci

actions-shellcheck:
	@docker compose run --rm go_tool_runner make actions-shellcheck-ci

actions-comment-secret-lint:
	@docker compose run --rm node_tool_runner make actions-comment-secret-lint-ci

# -----CI内で実行するコマンド群-----
actions-lint-ci: actions-actionlint-ci actions-shellcheck-ci actions-comment-secret-lint-ci

actions-actionlint-ci:
	actionlint

actions-shellcheck-ci:
	go run ./scripts/actions-shellcheck

# actionlint は「upsert-pr-comment を使うジョブに secret を渡すな」という規約を表現できない。
actions-comment-secret-lint-ci:
	node scripts/pr-comment-secret-lint.mjs
