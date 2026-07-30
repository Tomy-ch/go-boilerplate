## GitHub Actions 定義の静的解析（zizmor）
# -----ホストで直接実行するコマンド群-----
.PHONY: actions-zizmor ## workflows / composite action の定義を zizmor で検査（オフライン監査のみ・high でゲート）
# -----CI内で実行するコマンド群-----
.PHONY: actions-zizmor-sarif-ci ## zizmor の全所見を SARIF で標準出力へ書き出す(CI用)
.PHONY: actions-zizmor-gate-ci ## zizmor の high 所見でゲート(CI用・オンライン監査込み)

# tool-runner コンテナではなくホストで実行する。docs/rules.md「Toolchain Execution Rules」の
# 既定はコンテナ経由だが、zizmor は上流が musl ビルドを配布しておらず（リリース資産は
# unknown-linux-gnu / apple-darwin / pc-windows-msvc のみ）、alpine ベースの go_tool_runner に
# は載せられない。専用の glibc ランナーを増やすほどの検査ではないため、golangci-lint と同じく
# ホストの mise で解決し、provisioning を make install-tools が担う形に揃える。
# バージョンの正本は mise.toml。
ZIZMOR := $(shell mise which zizmor 2>/dev/null || command -v zizmor 2>/dev/null || echo zizmor)

# 検査対象は `.`。zizmor が workflows と composite action の両方を収集し、除外設定は
# .github/zizmor.yml がリポジトリルートから自動検出される。
ZIZMOR_TARGET := .

# -q: ファイルごとの INFO 行を落とし、所見だけを残す。
ZIZMOR_FLAGS := -q --no-progress

# high のみをゲート対象にする方針は hook と CI で共通。この変数がその単一の出所であり、
# medium 以下は SARIF から code scanning へ流れるので落とさなくても見失わない。
ZIZMOR_GATE_SEVERITY := high

# make はレシピの終了コードを自分の 2 へ丸めるため、呼び出し側へ届くのは「0 か否か」だけで、
# zizmor 固有のコード（所見ありの 14 など）は失われる。ここを呼ぶ側は二値で判定すること。

# -----ホストで直接実行するコマンド群-----
# --offline: hook をネットワークと GH_TOKEN から切り離す。オンライン監査（impostor-commit
# 等）は GitHub API を引くため hook では走らないが、run: の式展開を見る template-injection は
# オフライン監査なのでここで捕まる。オンライン監査は CI 側の actions-zizmor-gate-ci が担う。
#
# 未インストール時は解決に失敗して素の `zizmor` へ落ちる。そのままでは shell の
# command not found になり、フックの落ちた理由が読み取れないので導線を出す。
actions-zizmor:
	@command -v $(ZIZMOR) >/dev/null 2>&1 || { \
		echo "❌ zizmor が見つかりません。\`make install-tools\` を実行してください。"; \
		exit 1; \
	}
	@$(ZIZMOR) $(ZIZMOR_FLAGS) --offline --min-severity $(ZIZMOR_GATE_SEVERITY) $(ZIZMOR_TARGET)

# -----CI内で実行するコマンド群-----
# SARIF は severity で絞らない。ゲートが high だけを見る一方で、code scanning には全所見を
# 残すことで「落とさないが記録はする」層を成立させる。
#
# 出力は純粋な SARIF でなければ code scanning が受け取れない。レシピの `@` がエコーを
# 抑えているので現状それは満たされるが、呼び出し側は `make -s` も併せて渡すこと。`@` が
# 一つ落ちただけで JSON の先頭にコマンド文字列が混ざる壊れ方をするため。
actions-zizmor-sarif-ci:
	@$(ZIZMOR) --no-progress --format sarif $(ZIZMOR_TARGET)

actions-zizmor-gate-ci:
	@$(ZIZMOR) $(ZIZMOR_FLAGS) --format plain --min-severity $(ZIZMOR_GATE_SEVERITY) $(ZIZMOR_TARGET)
