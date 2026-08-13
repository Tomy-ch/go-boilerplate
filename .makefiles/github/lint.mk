## GitHub Actions に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: actions-lint ## GitHub Actions 定義(ワークフロー)のLintを実行（actionlint + composite action の run シェル検査 + Node 検査群）
.PHONY: actions-shellcheck ## composite action の run シェルを shellcheck で検査
.PHONY: actions-comment-secret-lint ## upsert-pr-comment を使うジョブへの secret 混入検査のみを実行
.PHONY: actions-comment-fence-lint ## PRコメント本文のフェンス検査(固定長フェンス/実装一致/補間span)のみを実行
.PHONY: actions-cutoff-lint ## ジョブ打ち切り時の振る舞い(timeout-minutes / PRコメントの always())検査のみを実行
.PHONY: actions-mise-pin-lint ## setup-mise の版 / digest / キャッシュキーの整合を検査
.PHONY: required-check-lint ## Ruleset の required context と本体 / guard workflow の対応を検査
# -----CI内で実行するコマンド群-----
.PHONY: actions-lint-ci ## GitHub Actions 定義のLintを実行(CI用)
.PHONY: actions-actionlint-ci ## actionlint でワークフロー定義をLint(CI用)
.PHONY: actions-shellcheck-ci ## composite action の run シェルを shellcheck で検査(CI用)
.PHONY: actions-comment-secret-lint-ci ## upsert-pr-comment を使うジョブへの secret 混入を検査(CI用)
.PHONY: actions-comment-fence-lint-ci ## PRコメント本文のフェンス(固定長フェンス/実装一致/補間span)を検査(CI用)
.PHONY: actions-node-lint-ci ## node で書かれた検査 4 種をまとめて実行(CI用)
.PHONY: actions-cutoff-lint-ci ## ジョブ打ち切り時の振る舞いを検査(CI用)
.PHONY: actions-mise-pin-lint-ci ## setup-mise の版 / digest / キャッシュキーの整合を検査(CI用)
.PHONY: required-check-lint-ci ## Ruleset の required context と本体 / guard workflow の対応を検査(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
# actionlint は Go ツール、secret / フェンス検査は node スクリプトなので tool-runner を跨ぐ。
actions-lint:
	@docker compose run --rm go_tool_runner make actions-actionlint-ci
	@docker compose run --rm go_tool_runner make actions-shellcheck-ci
	@docker compose run --rm node_tool_runner make actions-node-lint-ci

actions-shellcheck:
	@docker compose run --rm go_tool_runner make actions-shellcheck-ci

actions-comment-secret-lint:
	@docker compose run --rm node_tool_runner make actions-comment-secret-lint-ci

actions-comment-fence-lint:
	@docker compose run --rm node_tool_runner make actions-comment-fence-lint-ci

actions-cutoff-lint:
	@docker compose run --rm node_tool_runner make actions-cutoff-lint-ci

actions-mise-pin-lint:
	@docker compose run --rm node_tool_runner make actions-mise-pin-lint-ci

required-check-lint:
	@docker compose run --rm node_tool_runner make required-check-lint-ci

# -----CI内で実行するコマンド群-----
actions-lint-ci: actions-actionlint-ci actions-shellcheck-ci actions-node-lint-ci

# actionlint → shellcheck → node の順序は入れ替えない
# （理由は .makefiles/README.md の actions-lint-ci 行）。
actions-node-lint-ci: actions-comment-secret-lint-ci actions-comment-fence-lint-ci actions-cutoff-lint-ci actions-mise-pin-lint-ci

actions-actionlint-ci:
	actionlint

actions-shellcheck-ci:
	go run ./scripts/actions-shellcheck

# actionlint は「upsert-pr-comment を使うジョブに secret を渡すな」という規約を表現できない。
actions-comment-secret-lint-ci:
	$(TSX) scripts/pr-comment-secret-lint

# actionlint は「PR コメント本文のフェンスを固定長にするな」「素通し経路で span へ値を補間するな」
# という規約を表現できない。
actions-comment-fence-lint-ci:
	$(TSX) scripts/pr-comment-fence-lint

# actionlint は「打ち切られたジョブでも PR に結果を残せ」「全ジョブに timeout-minutes を置け」という規約を表現できない。
actions-cutoff-lint-ci:
	$(TSX) scripts/actions-cutoff-lint

actions-mise-pin-lint-ci:
	$(TSX) scripts/actions-mise-pin-lint

required-check-lint-ci:
	$(TSX) scripts/required-check-lint
