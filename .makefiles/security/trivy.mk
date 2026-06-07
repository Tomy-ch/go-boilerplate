## Trivyによる脆弱性スキャンコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: trivy-fs ## 依存ライブラリの脆弱性を Trivy fs でスキャン
# -----CI内で実行するコマンド群-----
.PHONY: trivy-fs-ci ## 依存ライブラリの脆弱性を Trivy fs でスキャン(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
trivy-fs:
	@docker compose run --rm go_tool_runner make trivy-fs-ci

# -----CI内で実行するコマンド群-----
trivy-fs-ci:
	trivy fs \
		--scanners vuln \
		--pkg-types library \
		--skip-dirs vendor \
		--ignore-unfixed \
		--severity CRITICAL,HIGH,MEDIUM \
		--skip-version-check \
		.
