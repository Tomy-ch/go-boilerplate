## コミットメッセージ(commitlint)に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: commitlint ## コミットメッセージを commitlint で検証（node_tool_runner 経由。COMMIT_MSG_FILE 未指定時は git rev-parse --git-path COMMIT_EDITMSG）
# -----CI内で実行するコマンド群-----
.PHONY: commitlint-ci ## コミットメッセージを commitlint で検証(CI用)
.PHONY: commitlint-range-ci ## COMMITLINT_FROM..COMMITLINT_TO のコミット範囲を commitlint で検証(CI用)

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

# コミット範囲を検証する。commit-msg フックは分割コミット時にバイパスされ、最後に回す
# lefthook run pre-commit にも含まれない（フック名が違う）ため、そこを通り抜けたメッセージは
# この経路でしか検査されない。範囲が空なら参照解決が壊れているとみなして落とす。黙って
# 0 件を検査して緑を返すと、ゲートが外れたことと合格したことが見分けられなくなる。
# ローカル向けの node_tool_runner ラッパーは無い。コンテナのマウントは .:/app だけで
# worktree の gitdir はその外にあり、履歴はメッセージファイルのように写して渡せない。
commitlint-range-ci:
	@set -eu; \
	[ -n '$(COMMITLINT_FROM)' ] && [ -n '$(COMMITLINT_TO)' ] || \
		{ echo "❌ COMMITLINT_FROM と COMMITLINT_TO の指定が必要です"; exit 2; }; \
	count="$$(git rev-list --count '$(COMMITLINT_FROM)..$(COMMITLINT_TO)')"; \
	echo "🔍 commitlint range: $(COMMITLINT_FROM)..$(COMMITLINT_TO) （$$count 件）"; \
	[ "$$count" -gt 0 ] || \
		{ echo "❌ 検査対象が 0 件です（参照解決が壊れている可能性があります）"; exit 2; }; \
	commitlint --from '$(COMMITLINT_FROM)' --to '$(COMMITLINT_TO)'
