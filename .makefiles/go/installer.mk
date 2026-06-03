## Go言語のツールインストーラー
.PHONY: go-update ## mise.toml に記載された Go バージョンをインストール
.PHONY: install-tools ## host 開発用の Go ツール一式を mise.toml のバージョンでインストール
.PHONY: activate-tools ## lefthookのインストールを実行
.PHONY: sync-versions ## mise.toml の go/node/python を go.mod と Dockerfile FROM へ反映

go-update:
	@if ! command -v mise >/dev/null 2>&1; then \
		echo "❌ mise が見つかりません。docs/maintenance/go-upgrade.md を参照してセットアップしてください。"; \
		exit 1; \
	fi
	@echo "🔄 mise で Go ランタイムをインストールします..."
	@mise install go

install-tools:
	@if ! command -v mise >/dev/null 2>&1; then \
		echo "❌ mise が見つかりません。docs/maintenance/go-upgrade.md を参照してセットアップしてください。"; \
		exit 1; \
	fi
	@echo "🔄 host 用 Go ツール群を mise.toml のバージョンでインストールします..."
	@mise install aqua:golang.org/x/tools/gopls
	@mise install go:github.com/cweill/gotests/gotests
	@mise install go:github.com/josharian/impl
	@mise install aqua:go-delve/delve
	@mise install lefthook
	@mise install golangci-lint
	@mise reshim
	@echo "✅ Go tools installed successfully."

activate-tools:
	lefthook install

sync-versions:
	@go run ./scripts/sync-versions
