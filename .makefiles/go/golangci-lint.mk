## GolangCI-Lintのコマンド群
.PHONY: lint ## コードの静的解析
.PHONY: fix ## コードの自動修正

# mise which で binary path を make-parse 時に解決することで、host shell に mise activate が
# 入っていない環境でも mise.toml で pin したバージョンが確実に呼ばれる。mise が無ければ
# PATH 上の golangci-lint にフォールバックし、それも無ければ bare 名前のままにする
# （結果として `command not found` が出るが、エラーメッセージとしては明示的になる）。
GOLANGCI_LINT := $(shell mise which golangci-lint 2>/dev/null || command -v golangci-lint 2>/dev/null || echo golangci-lint)

lint:
	@$(GOLANGCI_LINT) run --config .golangci-full.yaml

fix:
	@$(GOLANGCI_LINT) run --fix --config .golangci-full.yaml
