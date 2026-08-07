## MockAuthサーバー関連

.PHONY: reset-mock-auth-users ## MockAuthサーバーの固定ユーザーを中立な既定へリセットする
.PHONY: reset-mock-auth-users-ci ## MockAuthサーバーの固定ユーザーを中立な既定へリセットする（CI用）

reset-mock-auth-users:
	@docker compose run --rm node_tool_runner make reset-mock-auth-users-ci

reset-mock-auth-users-ci:
	$(TSX) scripts/reset-mock-auth-users.ts
