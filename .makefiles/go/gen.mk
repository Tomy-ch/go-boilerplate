## Go言語のコード生成関連のコマンド群
# -----Dockerコンテナ内で実行するCI用ターゲット-----
.PHONY: gen-go-code ## Goコードの生成を実行
# -----CI用ターゲット-----
.PHONY: gen-go-code-ci ## Goコードの生成を実行（CI用）

# -----Dockerコンテナ内で実行するCI用ターゲット-----
gen-go-code:
	@docker compose run --rm go_tool_runner make gen-go-code-ci

# -----CI用ターゲット-----
gen-go-code-ci:
	go generate ./...
