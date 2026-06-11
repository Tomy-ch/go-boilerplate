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
MD_GLOBS := "**/*.md" "#vendor/**" "#node_modules/**" "#.git/**" "#docs/portal/guides/**" "#docs/coverage/**" "#docs/db-schema/**" "#AGENTS.md"

md-lint-ci:
	markdownlint-cli2 $(MD_GLOBS)

md-fix-ci:
	markdownlint-cli2 --fix $(MD_GLOBS)
