## Markdownに対するLint/Fixコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: md-lint ## MarkdownのLintを実行（markdownlint + mermaid 構文検証）
.PHONY: md-fix ## MarkdownのLint自動修正を実行
.PHONY: md-mermaid-lint ## Mermaid 図の構文Lintのみを実行
# -----CI内で実行するコマンド群-----
.PHONY: md-lint-ci ## MarkdownのLintを実行(CI用)
.PHONY: md-fix-ci ## MarkdownのLint自動修正を実行(CI用)
.PHONY: md-markdownlint-ci
.PHONY: md-mermaid-lint-ci ## Mermaid 図の構文Lintを実行(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
md-lint:
	@docker compose run --rm node_tool_runner make md-lint-ci

md-fix:
	@docker compose run --rm node_tool_runner make md-fix-ci

md-mermaid-lint:
	@docker compose run --rm node_tool_runner make md-mermaid-lint-ci

# -----CI内で実行するコマンド群-----
# markdownlint-cli2 の ignore 記法は "#glob"。Make 変数代入では # がコメント開始になるため \# でエスケープする。
MD_GLOBS := "**/*.md" "\#vendor/**" "\#**/node_modules/**" "\#.git/**" "\#docs/portal/guides/**" "\#docs/coverage/**" "\#docs/db-schema/**" "\#AGENTS.md"

# markdownlint（体裁）と mermaid（図の構文）の両方を通す。どちらかが落ちれば md-lint-ci も落ちる。
md-lint-ci: md-markdownlint-ci md-mermaid-lint-ci

md-markdownlint-ci:
	markdownlint-cli2 $(MD_GLOBS)

# Markdown 内の ```mermaid フェンスを実 mermaid パーサで構文検証する（markdownlint は図の中身を見ない）。
md-mermaid-lint-ci:
	node scripts/mermaid-lint.mjs

md-fix-ci:
	markdownlint-cli2 --fix $(MD_GLOBS)
