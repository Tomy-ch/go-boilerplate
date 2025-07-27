## 開発環境系(生成系は別ファイルに分離)
.PHONY: start ## 開発環境の実行
.PHONY: install ## goツールのインストール
.PHONY: tidy-lib ## ローカルライブラリの依存関係更新
.PHONY: fix ## コードの自動修正
.PHONY: lint ## コードの静的解析
.PHONY: test ## CI用のテスト実行
.PHONY: test-repo ## テストの実行とテストレポートの生成

export PJ_DIR=$(shell pwd)
TGT_PKGS := $(shell go list ./... | grep -v '/gen')

start:
	docker compose --profile development up -d --build

tidy-lib:
	go mod tidy
	go mod vendor

fix:
	@golangci-lint run --fix --config .golangci-full.yaml

lint:
	@golangci-lint run --config .golangci-full.yaml

test-repo:
	@go test $(TGT_PKGS) -coverprofile=coverage/coverage.out -covermode=atomic
	@go tool cover -html=coverage/coverage.out -o coverage/test-result.html
	@rm -f coverage/coverage.out

test:
	@go test $(TGT_PKGS) -cover -count=1

install:
	@echo "Installing Go tools..."
	go install golang.org/x/tools/gopls@latest
	go install github.com/cweill/gotests/...@latest
	go install github.com/josharian/impl@latest
	go install github.com/haya14busa/goplay/cmd/goplay@latest
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/evilmartians/lefthook@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	goenv rehash
	@grep -Fxq 'export PATH="$$HOME/go/bin:$$PATH"' $$HOME/.zprofile || \
		echo 'export PATH="$$HOME/go/bin:$$PATH"' >> $$HOME/.zprofile
	@echo "Go tools installed successfully."
