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

# dir モードは作業ツリーのスナップショットしか見ないため、コミットして後で消した
# シークレットを取りこぼす。git モードはコミット履歴全体を走査するので、マージ済みの
# 履歴に埋もれたシークレットを定期実行で検知できる（secret-scan.yaml の週次実行が使用）。
secret-scan-history-ci:
	gitleaks git . --no-banner --redact --no-color
