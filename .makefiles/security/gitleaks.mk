## シークレットスキャン（gitleaks）
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: secret-scan ## ワーキングツリーのシークレットスキャンを実行
# -----CI内で実行するコマンド群-----
.PHONY: secret-scan-ci ## シークレットスキャンを実行(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
secret-scan:
	@docker compose run --rm go_tool_runner make secret-scan-ci

# -----CI内で実行するコマンド群-----
# --redact: 検出値をログ/PRコメントに出さない（漏洩二次被害の防止）
secret-scan-ci:
	gitleaks dir . --no-banner --redact
