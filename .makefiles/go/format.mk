.PHONY: fmt ## 折り返しフォーマッター

fmt:
	golines --max-len=80 --base-formatter=gofumpt -w .
