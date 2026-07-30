## コミットメッセージ(commitlint)に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: commitlint ## コミットメッセージを commitlint で検証（node_tool_runner 経由。COMMIT_MSG_FILE 未指定時は git rev-parse --git-path COMMIT_EDITMSG）
# -----CI内で実行するコマンド群-----
.PHONY: commitlint-ci ## コミットメッセージを commitlint で検証(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
# worktree の .git はファイルでありメインの checkout 側を指すため、git が渡すメッセージファイル
# （COMMIT_EDITMSG / MERGE_MSG）は node_tool_runner のマウント範囲（.:/app）の外にある。
# 作業ツリー内へ写して相対パスで渡すことで、worktree でも通常の checkout でも同じ経路で検証する。
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
