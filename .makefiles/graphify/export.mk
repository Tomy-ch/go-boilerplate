## Graphify knowledge graph
.PHONY: graphify-update ## コード差分だけを Graphify グラフへ反映（決定的・モデル不要）
.PHONY: graphify-update-ci ## CI/tool-runner 内で Graphify のコード差分反映を実行
.PHONY: graphify-export ## Graphify graph.json を決定的な nodes/edges/metadata JSON へ full export
.PHONY: graphify-export-ci ## CI/tool-runner 内で Graphify JSON の full export を実行
.PHONY: graphify-check ## 追跡中の Graphify 成果物とキャッシュの素性を検証（書き換えなし・CI/hook用）
.PHONY: graphify-pending ## 意味論抽出がどれだけ溜まっているかを測る（報告のみ）

# SPEC / BASE の意味と使いどころは .makefiles/README.md の `.makefiles/graphify` group を参照。
SPEC ?=
BASE ?=
# 目安であって門ではない（超えても失敗しない）。値の根拠は .makefiles/README.md を参照。
GRAPHIFY_PENDING_THRESHOLD ?= 3000

graphify-update:
	@docker compose run --rm python_tool_runner make graphify-update-ci

# 走査ルートは省略不可。graphify は graphify-out/.graphify_root から復元するが、あれは未追跡かつ
# マシン固有の絶対パスを持つ。
graphify-update-ci:
	@graphify update .

graphify-export:
	@docker compose run --rm go_tool_runner make graphify-export-ci

graphify-export-ci:
	@go run ./scripts/graphify-export graphify-out/graph.json graphify-out

# 検査対象は git の index でツールチェーンではないため、ホストで走らせる。
graphify-check:
	@go run ./scripts/graphify-check $(if $(SPEC),-spec $(SPEC)) $(if $(BASE),-base $(BASE))

# 抽出しないこのターゲットを graphify-extract と名付けないこと。意味論抽出は make から起動できず
# （アシスタントの `/graphify --update`）、正規の経路と手順は .makefiles/README.md にある。
graphify-pending:
	@go run ./scripts/graphify-pending -threshold $(GRAPHIFY_PENDING_THRESHOLD) $(GRAPHIFY_PENDING_ARGS)
