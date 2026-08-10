## Markdownに対するLint/Fixコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: md-lint ## MarkdownのLintを実行（markdownlint + mermaid 構文検証 + スキル定義の意味検査）
.PHONY: md-fix ## MarkdownのLint自動修正を実行
.PHONY: md-mermaid-lint ## Mermaid 図の構文Lintのみを実行
.PHONY: md-skill-lint ## スキル/エージェント定義(.claude/** と .codex/** の対応)の意味検査のみを実行
.PHONY: md-doc-ref-lint ## ADR 参照と対訳ペアの実在性を検査
.PHONY: md-doc-ref-fix ## ADR 参照へ canonical slug を補う
.PHONY: md-premise-lint ## fork 後も残る文書に失効予定の前提が書かれていないかのみを検査 # boilerplate-only:line
# -----CI内で実行するコマンド群-----
.PHONY: md-lint-ci ## MarkdownのLintを実行(CI用)
.PHONY: md-fix-ci ## MarkdownのLint自動修正を実行(CI用)
.PHONY: md-markdownlint-ci ## markdownlint-cli2 で Markdown 体裁を Lint(CI用)
.PHONY: md-mermaid-lint-ci ## Mermaid 図の構文Lintを実行(CI用)
.PHONY: md-skill-lint-ci ## スキル/エージェント定義(.claude/** と .codex/** の対応)の意味検査を実行(CI用)
.PHONY: md-premise-lint-ci ## fork 後も残る文書に失効予定の前提が書かれていないかを検査(CI用) # boilerplate-only:line

# -----Dockerコンテナ内で実行するコマンド群-----
md-lint:
	@docker compose run --rm node_tool_runner make md-lint-ci

md-fix:
	@docker compose run --rm node_tool_runner make md-fix-ci

md-mermaid-lint:
	@docker compose run --rm node_tool_runner make md-mermaid-lint-ci

md-skill-lint:
	@docker compose run --rm node_tool_runner make md-skill-lint-ci

md-doc-ref-lint:
	@docker compose run --rm node_tool_runner make md-doc-ref-lint-ci

md-doc-ref-fix:
	@docker compose run --rm node_tool_runner make md-doc-ref-fix-ci

# boilerplate-only:begin
md-premise-lint:
	@docker compose run --rm node_tool_runner make md-premise-lint-ci
# boilerplate-only:end

# -----CI内で実行するコマンド群-----
# 除外は .markdownlint-cli2.yaml の ignores: に集約する（エディタ拡張と同じ結果にするため）。
MD_GLOBS := "**/*.md"

# markdownlint（体裁）・mermaid（図の構文）・skill-lint（.claude/** の意味と .codex/** との対応）の3段を通す。
# 上流ではこれに premise-lint（fork 後も残る文書に失効予定の前提が無いこと）が加わる。 # boilerplate-only:line
# boilerplate-only:replace-begin
md-lint-ci: md-markdownlint-ci md-mermaid-lint-ci md-skill-lint-ci md-doc-ref-lint-ci md-premise-lint-ci
# boilerplate-only:replace-with
# = md-lint-ci: md-markdownlint-ci md-mermaid-lint-ci md-skill-lint-ci
# boilerplate-only:replace-end

md-markdownlint-ci:
	markdownlint-cli2 $(MD_GLOBS)

# Markdown 内の ```mermaid フェンスを実 mermaid パーサで構文検証する（markdownlint は図の中身を見ない）。
md-mermaid-lint-ci:
	$(TSX) scripts/mermaid-lint

# スキル / エージェント定義が参照する make ターゲット・パスの実在性、対訳ペアの構造同期、
# および .claude/** と .codex/** の存在対応を検証する。
md-skill-lint-ci:
	$(TSX) scripts/skill-lint

md-doc-ref-lint-ci:
	$(TSX) scripts/doc-ref-lint

md-doc-ref-fix-ci:
	$(TSX) scripts/doc-ref-lint --write

# boilerplate-only:begin
# docs/rules.md の「No premise the document will outlive」を機械化したもの。前提を書いてよいのは
# fork 時に破棄・書き換えされる文書（README / docs/get-started/**）とマーカーで囲った領域だけ。
# 規則自体は fork が継承するが、検査が探す言い回しは上流固有の実例なので、検査ごと撤去される。
md-premise-lint-ci:
	$(TSX) scripts/premise-lint
# boilerplate-only:end

md-fix-ci:
	markdownlint-cli2 --fix $(MD_GLOBS)
