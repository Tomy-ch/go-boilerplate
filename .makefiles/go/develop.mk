## 開発環境系(生成系は別ファイルに分離)
.PHONY: start ## 開発環境の実行
.PHONY: install ## goツールのインストール
.PHONY: tidy-lib ## ローカルライブラリの依存関係更新
.PHONY: fmt ## フォーマットの適用

export PJ_DIR=$(shell pwd)

start:
	go run internal/main.go

tidy-lib:
	go mod tidy
	go mod vendor

fmt:
	golines --max-len=80 --base-formatter=gofumpt -w .

install:
	go install mvdan.cc/gofumpt@latest
	go install github.com/segmentio/golines@latest
	go install github.com/evilmartians/lefthook@latest
	go install go.uber.org/mock/mockgen@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zprofile
