## Go言語のツールインストーラー
.PHONY: go-update ## .go-version に記載された Go バージョンをインストール（mise 優先 / goenv フォールバック）
.PHONY: install-tools ## goツールのインストール
.PHONY: activate-tools ## lefthookのインストールを実行

go-update:
	@if command -v mise >/dev/null 2>&1; then \
		echo "Installing Go via mise..."; \
		mise install; \
	elif command -v goenv >/dev/null 2>&1; then \
		echo "mise not found. Falling back to goenv (see docs/maintenance/go-upgrade.md for migration)..."; \
		anyenv update; \
		goenv install "$$(cat .go-version)"; \
	else \
		echo "Neither mise nor goenv is installed. See docs/maintenance/go-upgrade.md for setup instructions."; \
		exit 1; \
	fi

install-tools:
	@echo "Installing Go tools..."
	go install golang.org/x/tools/gopls@latest
	go install github.com/cweill/gotests/...@latest
	go install github.com/josharian/impl@latest
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/evilmartians/lefthook/v2@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@if command -v mise >/dev/null 2>&1; then \
		echo "mise is installed. Running 'mise reshim'..."; \
		mise reshim; \
	elif command -v goenv >/dev/null 2>&1; then \
		echo "goenv is installed. Running 'goenv rehash'..."; \
		goenv rehash; \
	else \
		echo "Neither mise nor goenv detected. Skipping shim regeneration."; \
	fi
	@grep -Fxq 'export PATH="$$HOME/go/bin:$$PATH"' $$HOME/.zprofile || \
		echo 'export PATH="$$HOME/go/bin:$$PATH"' >> $$HOME/.zprofile
	@echo "Go tools installed successfully."

activate-tools:
	lefthook install
