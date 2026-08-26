## Python ツールの lockfile 生成コマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: py-lock ## python/*.in から hash 付き lockfile(python/*.txt)を再生成
# -----CI内で実行するコマンド群-----
.PHONY: py-lock-ci ## hash 付き lockfile を再生成(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
py-lock:
	@docker compose run --rm python_tool_runner make py-lock-ci

# -----CI内で実行するコマンド群-----
# 解決は宣言している Python ランタイムに対して行う。走らせた環境の Python に任せると、
# 環境マーカーの評価が実行者ごとに変わり、同じ *.in から違う lockfile が出る。
# バージョンは mise ではなく mise.toml から直に読む。この recipe は tool-runner の中でも
# 走り、そこでの mise は build 時にコピーした mise.toml を信頼しているだけで、
# 実行時にマウントされるリポジトリのものを読めるとは限らない。
py-lock-ci:
	@python_version="$$(sed -n 's/^python = "\(.*\)"$$/\1/p' mise.toml)"; \
	test -n "$$python_version" || { echo "❌ mise.toml から python のバージョンを読めません"; exit 1; }; \
	for req in python/*.in; do \
		uv pip compile "$$req" --generate-hashes --universal \
			--python-version "$$python_version" -o "$${req%.in}.txt" || exit 1; \
	done
