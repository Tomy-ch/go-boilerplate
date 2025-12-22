## Go言語のライブラリ関連のコマンド群
.PHONY: tidy-lib ## ローカルライブラリの依存関係更新

tidy-lib:
	go mod tidy
	go mod vendor
