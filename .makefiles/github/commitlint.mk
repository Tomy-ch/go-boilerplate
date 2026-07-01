## コミットメッセージ(commitlint)に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: commitlint ## コミットメッセージを commitlint で検証（node_tool_runner 経由。COMMIT_MSG_FILE 未指定時は .git/COMMIT_EDITMSG）
# -----CI内で実行するコマンド群-----
.PHONY: commitlint-ci ## コミットメッセージを commitlint で検証(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
commitlint:
	@docker compose run --rm node_tool_runner make commitlint-ci COMMIT_MSG_FILE=$(COMMIT_MSG_FILE)

# -----CI内で実行するコマンド群-----
commitlint-ci:
	commitlint --edit $(COMMIT_MSG_FILE)
