## GolangCI-Lintのコマンド群
.PHONY: lint ## コードの静的解析
.PHONY: fix ## コードの自動修正

GOLANGCI_LINT := $(shell mise which golangci-lint 2>/dev/null || command -v golangci-lint 2>/dev/null || echo golangci-lint)

# .golangci-full.yaml の run.timeout は CI ランナーの実測に合わせた値で、複数 worktree を抱えた開発機では
# 全パッケージのスキャンがそれを超え、issue が 0 件でも "Timeout exceeded" で失敗する。検出内容は同じなので、
# ローカルの入口だけ余裕のある上限へ引き上げる（CLI 指定が config の run.timeout より優先される）。
GOLANGCI_LINT_TIMEOUT ?= 20m

lint:
	@$(GOLANGCI_LINT) run --config .golangci-full.yaml --timeout $(GOLANGCI_LINT_TIMEOUT)

fix:
	@$(GOLANGCI_LINT) run --fix --config .golangci-full.yaml --timeout $(GOLANGCI_LINT_TIMEOUT)
