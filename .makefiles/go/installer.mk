## Go言語のツールインストーラー
.PHONY: go-update ## goenvの更新を実行
.PHONY: install-tools ## goツールのインストール
.PHONY: activate-tools ## lefthookのインストールを実行

go-update:
	@anyenv update
	@goenv install "$$(cat .go-version)"

install-tools:
	@echo "Installing Go tools..."
	go install golang.org/x/tools/gopls@latest
	go install github.com/cweill/gotests/...@latest
	go install github.com/josharian/impl@latest
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/evilmartians/lefthook/v2@latest
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

activate-tools:
	lefthook install
