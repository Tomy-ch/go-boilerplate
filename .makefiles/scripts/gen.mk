## scriptsを用いた生成コマンド群
.PHONY: gen-ctxkey ## Contextに値を格納するためのコードを生成する(ctx-nameとctx-typeの指定が必要)

gen-ctxkey:
	@if [ -z "$(ctx-name)" ] || [ -z "$(ctx-type)" ]; then \
	echo "❌ ctx-nameとctx-typeの引数が必要です。以下のように指定してください："; \
	echo "   make gen-ctxkey ctx-name=UserID ctx-type=string"; \
	exit 1; \
	fi; \
	bash scripts/gen_ctxkey.sh $(ctx-name) $(ctx-type)
