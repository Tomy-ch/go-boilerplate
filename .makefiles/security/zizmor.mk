## GitHub Actions 定義の静的解析（zizmor）
# -----ホストで直接実行するコマンド群-----
.PHONY: actions-zizmor ## workflows / composite action の定義を zizmor で検査（オフライン監査のみ・high でゲート）
# -----CI内で実行するコマンド群-----
.PHONY: actions-zizmor-sarif-ci ## zizmor の全所見を SARIF で標準出力へ書き出す(CI用)
.PHONY: actions-zizmor-gate-ci ## zizmor の high 所見でゲート(CI用・オンライン監査込み)

# zizmor は osv-scanner / trufflehog / spectral と同じく、tool-runner コンテナを介さず
# ホストの mise で直接実行する。バージョンの正本は mise.toml。
ZIZMOR := $(shell mise which zizmor 2>/dev/null || command -v zizmor 2>/dev/null || echo zizmor)

# 検査対象は `.`。zizmor が workflows と composite action の両方を収集し、除外設定は
# .github/zizmor.yml がリポジトリルートから自動検出される。
ZIZMOR_TARGET := .

# -q: ファイルごとの INFO 行を落とし、所見だけを残す。
ZIZMOR_FLAGS := -q --no-progress

# high のみをゲート対象にする方針は CI（zizmor.yaml）と共通。medium 以下は SARIF から
# code scanning へ流れるので、落とさなくても見失わない。
ZIZMOR_GATE_SEVERITY := high

# -----ホストで直接実行するコマンド群-----
# --offline: hook をネットワークと GH_TOKEN から切り離す。オンライン監査（impostor-commit
# 等）は GitHub API を引くため hook では走らないが、run: の式展開を見る template-injection は
# オフライン監査なのでここで捕まる。オンライン監査は CI 側の actions-zizmor-gate-ci が担う。
actions-zizmor:
	@$(ZIZMOR) $(ZIZMOR_FLAGS) --offline --min-severity $(ZIZMOR_GATE_SEVERITY) $(ZIZMOR_TARGET)

# -----CI内で実行するコマンド群-----
# SARIF は severity で絞らない。ゲートが high だけを見る一方で、code scanning には全所見を
# 残すことで「落とさないが記録はする」層を成立させる。
#
# trivy 系と同じく、make 自身がレシピ行を stdout へエコーするため、機械可読な出力を
# 取るときは必ず `make -s` で呼ぶこと。
actions-zizmor-sarif-ci:
	@$(ZIZMOR) --no-progress --format sarif $(ZIZMOR_TARGET)

actions-zizmor-gate-ci:
	@$(ZIZMOR) $(ZIZMOR_FLAGS) --format plain --min-severity $(ZIZMOR_GATE_SEVERITY) $(ZIZMOR_TARGET)
