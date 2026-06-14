## Dockerfile に対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: docker-lint ## Dockerfile の Lint を実行
# -----CI内で実行するコマンド群-----
.PHONY: docker-lint-ci ## Dockerfile の Lint を実行(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
docker-lint:
	@docker compose run --rm go_tool_runner make docker-lint-ci

# -----CI内で実行するコマンド群-----
docker-lint-ci:
	hadolint docker/*/Dockerfile
