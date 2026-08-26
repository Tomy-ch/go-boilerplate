## コミットメッセージ(commitlint)に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: commitlint ## コミットメッセージを commitlint で検証（node_tool_runner 経由。COMMIT_MSG_FILE 未指定時は git rev-parse --git-path COMMIT_EDITMSG）
# -----CI内で実行するコマンド群-----
.PHONY: commitlint-ci ## コミットメッセージを commitlint で検証(CI用)
.PHONY: commitlint-range-ci ## COMMITLINT_FROM..COMMITLINT_TO のコミット範囲を commitlint で検証(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
# メッセージファイルを tmp/ へ写して相対パスで渡す理由（worktree ではマウント範囲 .:/app の外に
# あるため）は .makefiles/README.md の commitlint 行を参照。
commitlint:
	@set -e; \
	src='$(COMMIT_MSG_FILE)'; \
	[ -n "$$src" ] || src="$$(git rev-parse --git-path COMMIT_EDITMSG)"; \
	mkdir -p tmp; \
	msg="$$(mktemp tmp/commit-msg.XXXXXX)"; \
	trap 'rm -f "$$msg"' EXIT; \
	cp "$$src" "$$msg"; \
	docker compose run --rm node_tool_runner make commitlint-ci COMMIT_MSG_FILE="$$msg"

# -----CI内で実行するコマンド群-----
commitlint-ci:
	commitlint --edit $(COMMIT_MSG_FILE)

# commit-msg フックがバイパスされたメッセージに唯一届く検査経路。空範囲は落とす
# （0 件を緑で返すと、ゲートが外れたことと合格したことが区別できなくなる）。
# node_tool_runner ラッパーを持たない理由は .makefiles/README.md の commitlint-range-ci 行。
commitlint-range-ci:
	@set -eu; \
	[ -n '$(COMMITLINT_FROM)' ] && [ -n '$(COMMITLINT_TO)' ] || \
		{ echo "❌ COMMITLINT_FROM と COMMITLINT_TO の指定が必要です"; exit 2; }; \
	count="$$(git rev-list --count '$(COMMITLINT_FROM)..$(COMMITLINT_TO)')"; \
	echo "🔍 commitlint range: $(COMMITLINT_FROM)..$(COMMITLINT_TO) （$$count 件）"; \
	[ "$$count" -gt 0 ] || \
		{ echo "❌ 検査対象が 0 件です（参照解決が壊れている可能性があります）"; exit 2; }; \
	commitlint --from '$(COMMITLINT_FROM)' --to '$(COMMITLINT_TO)'
