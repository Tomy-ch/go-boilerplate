## Go言語のライブラリ関連のコマンド群
.PHONY: tidy-lib ## ローカルライブラリの依存関係更新
.PHONY: vendor-sync ## vendor/ が go.mod からずれていれば再生成する

tidy-lib:
	go mod tidy
	go mod vendor

vendor-sync:
	@go list -mod=vendor ./cmd/ > /dev/null 2>&1 || go mod vendor
