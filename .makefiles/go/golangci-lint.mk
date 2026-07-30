## GolangCI-Lintのコマンド群
.PHONY: lint ## コードの静的解析
.PHONY: fix ## コードの自動修正

GOLANGCI_LINT := $(shell mise which golangci-lint 2>/dev/null || command -v golangci-lint 2>/dev/null || echo golangci-lint)

# .golangci-full.yaml は固定 timeout を持たないため、ローカル実行のハング防止ガードはここが持つ。
# 開発機での実測は単独 2.5m だが、worktree 並行開発でホストが埋まると十数分（観測値 17.5m）まで伸びる。
# 検査そのものは完走するので、ここは所要の見積もりではなく「応答しなくなった場合の脱出口」として置く。
# CI は 0（無効）を渡し、打ち切りをジョブの timeout-minutes に委ねる。
GOLANGCI_LINT_TIMEOUT ?= 60m

lint:
	@$(GOLANGCI_LINT) run --config .golangci-full.yaml --timeout $(GOLANGCI_LINT_TIMEOUT)

fix:
	@$(GOLANGCI_LINT) run --fix --config .golangci-full.yaml --timeout $(GOLANGCI_LINT_TIMEOUT)
