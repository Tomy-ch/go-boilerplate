## 開発環境系
.PHONY: serve ## 開発環境の起動
.PHONY: serve-build ## ビルド実行後に、開発環境を起動する
.PHONY: tools ## 開発ツールの起動
.PHONY: install ## goツールのインストール
.PHONY: tidy-lib ## ローカルライブラリの依存関係更新
.PHONY: fmt ## コードをフォーマット
.PHONY: fix ## コードの自動修正
.PHONY: lint ## コードの静的解析
.PHONY: test ## CI用のテスト実行
.PHONY: test-repo ## テストの実行とテストレポートの生成
.PHONY: go-update ## goenvの更新を実行

TGT_PKGS := $(shell go list ./... | grep -Ev '/(gen|cli|cmd|mock|apperror)')
COVER_PKGS := $(shell go list ./... | grep -Ev '/(gen|cli|cmd|mock|apperror)' | tr '\n' ',' | sed 's/,$$//')

go-update:
	@anyenv update
	@goenv install "$(cat .go-version)"

serve:
	@echo "🔄 開発環境を起動します。"
	@make db-logs-delete
	@docker compose --profile development up -d
	@echo "✅ 開発環境の起動が完了しました。"

serve-build:
	@echo "🧰 ビルド後、開発環境を起動します。"
	@make db-logs-delete
	@docker compose --profile development up -d --build
	@echo "✅ 開発環境の起動が完了しました。"

smoke:
	@echo "🔄 Smoke Test環境を起動します。"
	@docker compose --profile smoke up --build -d smoke_server
	@echo "✅ Smoke Test環境の起動が完了しました。"

tools:
	@echo "🔄 開発ツールを起動します。"
	@docker compose --profile tools up -d --build
	@echo "✅ 開発ツールの起動が完了しました。"

tidy-lib:
	go mod tidy
	go mod vendor

fmt:
	@echo "🔄 コードのフォーマットを実行します..."
	@go fmt ./...
	@echo "✅ コードのフォーマットが完了しました。"

fix:
	@golangci-lint run --fix --config .golangci-full.yaml

lint:
	@golangci-lint run --config .golangci-full.yaml

test-repo:
	@echo "🔄 テストを実行し、レポートを生成します..."
	@touch docs/coverage/coverage.out
	@go test $(TGT_PKGS) -coverpkg=$(COVER_PKGS) -coverprofile=docs/coverage/coverage.out -covermode=atomic  >/dev/null 2>&1
	@go tool cover -html=docs/coverage/coverage.out -o docs/coverage/test-result.html
	@rm -f docs/coverage/coverage.out
	@echo "✅ テストレポートの生成が完了しました。"

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
