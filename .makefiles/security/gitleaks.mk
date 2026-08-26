## シークレットスキャン（gitleaks）
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: secret-scan ## ワーキングツリーのシークレットスキャンを実行
# -----CI内で実行するコマンド群-----
.PHONY: secret-scan-ci ## シークレットスキャンを実行(CI用)
.PHONY: secret-scan-history-ci ## コミット履歴全体のシークレットスキャンを実行(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
secret-scan:
	@docker compose run --rm go_tool_runner make secret-scan-ci

# -----CI内で実行するコマンド群-----
# --redact: 検出値をログ/PRコメントに出さない（漏洩二次被害の防止）
# --no-color: 非TTY（CI / lefthook ログ）で ANSI エスケープが化けないよう色付けを無効化する
secret-scan-ci:
	gitleaks dir . --no-banner --redact --no-color

# git モードは履歴全体を走査する（dir との違いは .makefiles/README.md の
# secret-scan-history-ci 行）。secret-scan.yaml の週次実行が使う。
secret-scan-history-ci:
	gitleaks git . --no-banner --redact --no-color
