## 自動生成系

.PHONY: go-gen ## go generateの実行
.PHONY: gen-ctxkey ## Contextに値を格納するためのコードを生成する(nameとtypeを指定が必要)

gen-ctxkey:
	@if [ -z "$(name)" ] || [ -z "$(type)" ]; then \
	echo "❌ nameとtypeの引数が必要です。以下のように指定してください："; \
	echo "   make gen-ctxkey name=UserID type=string"; \
	exit 1; \
	fi; \
	bash scripts/gen_ctxkey.sh $(name) $(type)

go-gen:
	@go generate ./...
