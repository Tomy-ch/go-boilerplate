## GolangCI-Lintのコマンド群
.PHONY: lint ## コードの静的解析
.PHONY: fix ## コードの自動修正

GOLANGCI_LINT := $(shell mise which golangci-lint 2>/dev/null || command -v golangci-lint 2>/dev/null || echo golangci-lint)

lint:
	@$(GOLANGCI_LINT) run --config .golangci-full.yaml

fix:
	@$(GOLANGCI_LINT) run --fix --config .golangci-full.yaml
