## Markdownに対するLint/Fixコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: md-lint ## MarkdownのLintを実行
.PHONY: md-fix ## MarkdownのLint自動修正を実行
# -----CI内で実行するコマンド群-----
.PHONY: md-lint-ci ## MarkdownのLintを実行(CI用)
.PHONY: md-fix-ci ## MarkdownのLint自動修正を実行(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
md-lint:
	@docker compose run --rm node_tool_runner make md-lint-ci

md-fix:
	@docker compose run --rm node_tool_runner make md-fix-ci

# -----CI内で実行するコマンド群-----
md-lint-ci:
	markdownlint-cli2 "**/*.md" "#vendor/**" "#node_modules/**" "#.git/**"

md-fix-ci:
	markdownlint-cli2 --fix "**/*.md" "#vendor/**" "#node_modules/**" "#.git/**"
