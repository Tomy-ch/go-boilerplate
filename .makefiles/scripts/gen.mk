## scriptsを用いた生成コマンド群
.PHONY: gen-ctxkey ## Contextに値を格納するためのコードを生成する(nameとtypeを指定が必要)

gen-ctxkey:
	@if [ -z "$(ctxkey-name)" ] || [ -z "$(ctxkey-type)" ]; then \
	echo "❌ nameとtypeの引数が必要です。以下のように指定してください："; \
	echo "   make gen-ctxkey name=UserID type=string"; \
	exit 1; \
	fi; \
	bash scripts/gen_ctxkey.sh $(ctxkey-name) $(ctxkey-type)
