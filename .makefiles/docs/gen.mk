## ドキュメント関連のコマンド群
.PHONY: gen-portal-docs ## Portal用のドキュメントを生成する
.PHONY: gen-docs-json ## Portal用のドキュメントリンクのJSONを生成する
.PHONY: gen-portal-build ## Portalフロントエンド(JSX)をesbuildでバンドルする
.PHONY: gen-portal-docs-ci ## Portal用のドキュメントを生成する（CI用）
.PHONY: gen-docs-json-ci ## Portal用のドキュメントリンクのJSONを生成する（CI用）
.PHONY: gen-portal-build-ci ## Portalフロントエンドをesbuildでバンドルする（CI用）
.PHONY: gen-godoc ## godoc の静的HTMLを docs/godoc/ に生成する
.PHONY: gen-godoc-ci ## godoc の静的HTMLを docs/godoc/ に生成する（CI用）

GODOC_OUT := docs/godoc
# --disable-filter なしだと internal/ 配下が全て除外されるため必須。exclude は index からのみ除外される。
GODOC_EXCLUDE = $(shell go list ./... | grep -E '/(gen|mock)$$|^[^/]+/(cmd|scripts)(/|$$)' | tr '\n' ' ')

gen-docs-json:
	@echo "🔍 Portal用のドキュメントリンクのJSONを生成します..."
	docker compose run --rm node_tool_runner make gen-docs-json-ci
	@echo "✅ Portal用のドキュメントリンクのJSONの生成が完了しました。"

gen-portal-docs:
	@echo "🔍 Portal用のドキュメントの生成を開始します..."
	docker compose run --rm node_tool_runner make gen-portal-docs-ci
	@echo "✅ Portal用のドキュメントの生成が完了しました。"

gen-portal-build:
	@echo "🔍 Portalフロントエンドのバンドルを開始します..."
	docker compose run --rm node_tool_runner make gen-portal-build-ci
	@echo "✅ Portalフロントエンドのバンドルが完了しました。"

gen-docs-json-ci:
	node scripts/gen-docs-json.mjs

gen-portal-docs-ci:
	node scripts/gen-portal-docs.mjs

gen-portal-build-ci:
	node scripts/build-portal.mjs

gen-godoc:
	@echo "🔍 godoc の静的HTMLの生成を開始します..."
	docker compose run --rm go_tool_runner make gen-godoc-ci
	@echo "✅ godoc の静的HTMLの生成が完了しました（$(GODOC_OUT)/）。"

gen-godoc-ci:
	rm -rf $(GODOC_OUT)
	mkdir -p $(GODOC_OUT)
	godoc-static \
		--listen=127.0.0.1:9001 \
		--go-root="$(shell go env GOROOT)" \
		--disable-filter \
		--exclude="$(GODOC_EXCLUDE)" \
		--site-name="go-boilerplate godoc" \
		--destination=$(GODOC_OUT) \
		--zip="" \
		.
	# godoc-static が書き出すファイル一覧の mtime セル（CI 実行時刻ごとに変わり PR ノイズの原因）を空にして deterministic に
	find $(GODOC_OUT) -name 'index.html' -exec sed -i -E 's|<td align="left">[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9:.]+ \+0000 UTC</td>|<td align="left"></td>|g' {} +
