## GolangCI-Lintのコマンド群
.PHONY: lint ## コードの静的解析
.PHONY: fix ## コードの自動修正

lint:
	@golangci-lint run --config .golangci-full.yaml

fix:
	@golangci-lint run --fix --config .golangci-full.yaml
