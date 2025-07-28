## 開発環境系(生成系は別ファイルに分離)
.PHONY: serve ## 開発環境の起動
.PHONY: serve-build ## ビルド実行後に、開発環境を起動する
.PHONY: tools ## 開発ツールの起動
.PHONY: tools-build ## ビルド実行後に、開発ツールを起動する
.PHONY: install ## goツールのインストール
.PHONY: tidy-lib ## ローカルライブラリの依存関係更新
.PHONY: fix ## コードの自動修正
.PHONY: lint ## コードの静的解析
.PHONY: test ## CI用のテスト実行
.PHONY: test-repo ## テストの実行とテストレポートの生成
.PHONY: del-db-logs ## DBのログを削除

TGT_PKGS := $(shell go list ./... | grep -v '/gen')

serve:
	@echo "🔄 開発環境を起動します。"
	@make del-db-logs
	@docker compose --profile development up -d
	@echo "✅ 開発環境の起動が完了しました。"

serve-build:
	@echo "🧰 ビルド後、開発環境を起動します。"
	@make del-db-logs
	@docker compose --profile development up -d --build
	@echo "✅ 開発環境の起動が完了しました。"

tools:
	@echo "🔄 開発ツールを起動します。"
	@docker compose --profile tools up -d
	@echo "✅ 開発ツールの起動が完了しました。"

tools-build:
	@echo "🧰 ビルド後、開発ツールを起動します。"
	@docker compose --profile tools up -d --build
	@echo "✅ 開発ツールの起動が完了しました。"

tidy-lib:
	go mod tidy
	go mod vendor

fix:
	@golangci-lint run --fix --config .golangci-full.yaml

lint:
	@golangci-lint run --config .golangci-full.yaml

test-repo:
	@go test $(TGT_PKGS) -coverprofile=docs/coverage/coverage.out -covermode=atomic
	@go tool cover -html=docs/coverage/coverage.out -o docs/coverage/test-result.html
	@rm -f docs/coverage/coverage.out

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
	@if command -v goenv >/dev/null 2>&1; then \
		echo "goenv is installed. Running 'goenv rehash'..."; \
		goenv rehash; \
	else \
		echo "goenv is not installed. Skipping 'goenv rehash'."; \
	fi
	@grep -Fxq 'export PATH="$$HOME/go/bin:$$PATH"' $$HOME/.zprofile || \
		echo 'export PATH="$$HOME/go/bin:$$PATH"' >> $$HOME/.zprofile
	@echo "Go tools installed successfully."

del-db-logs:
	@rm -rf docker/database/logs/*
