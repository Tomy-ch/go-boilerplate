## 自動生成系

.PHONY: gen ## 各種ドキュメントやコードの生成します
.PHONY: gen-build ## ビルド実行後に、各種生成を実行する
.PHONY: gen-ctxkey ## Contextに値を格納するためのコードを生成する(nameとtypeを指定が必要)

gen-ctxkey:
	@if [ -z "$(name)" ] || [ -z "$(type)" ]; then \
	echo "❌ nameとtypeの引数が必要です。以下のように指定してください："; \
	echo "   make gen-ctxkey name=UserID type=string"; \
	exit 1; \
	fi; \
	bash scripts/gen_ctxkey.sh $(name) $(type)

gen:
	@echo "🔄 各種ドキュメントやコードの生成します..."
	@docker compose --profile generate up -d
	@echo "✅ 各種ドキュメントやコードの生成が完了しました。"

gen-build:
	@echo "🧰 ビルド後、各種生成を開始します。"
	@docker compose --profile generate up -d --build
	@echo "✅ 生成が完了しました。"

