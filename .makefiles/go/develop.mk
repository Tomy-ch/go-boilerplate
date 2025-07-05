.PHONY: tidy-lib ## ローカルライブラリの依存更新

tidy-lib:
	go mod tidy
	go mod vendor
