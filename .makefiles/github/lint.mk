## GitHub Actions に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: actions-lint ## GitHub Actions 定義(ワークフロー)のLintを実行
# -----CI内で実行するコマンド群-----
.PHONY: actions-lint-ci ## GitHub Actions 定義のLintを実行(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
actions-lint:
	@docker compose run --rm go_tool_runner make actions-lint-ci

# -----CI内で実行するコマンド群-----
actions-lint-ci:
	actionlint
