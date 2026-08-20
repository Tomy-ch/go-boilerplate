## Graphify knowledge graph
.PHONY: graphify-update ## コード差分だけを Graphify グラフへ反映（決定的・モデル不要）
.PHONY: graphify-update-ci ## CI/tool-runner 内で Graphify のコード差分反映を実行
.PHONY: graphify-export ## Graphify graph.json を決定的な nodes/edges/metadata JSON へ full export
.PHONY: graphify-export-ci ## CI/tool-runner 内で Graphify JSON の full export を実行
.PHONY: graphify-check ## 追跡中の Graphify 成果物とキャッシュの素性を検証（書き換えなし・CI/hook用）
.PHONY: graphify-pending ## 意味論抽出がどれだけ溜まっているかを測る（報告のみ）

# 抽出プロンプトの実体は graphify のパッケージ内にあり、リポジトリからは見えない。
# SPEC を与えたときだけ .agents/graphify/spec-pin.toml と突き合わせる。
SPEC ?=
# BASE を与えると、その ref からの差分に graphify-out/ が含まれていないかを検査する。
BASE ?=
# 意味論抽出をまとめて回す判断の目安（変更行数）。門ではないので超えても失敗にはしない。
# 意味論コーパスは約 92,500 行なので、3000 行は約 3% にあたる。低く置くと ADR を 1 本
# 書き直しただけで鳴り、鳴っても回せない（モデルが要る）ため警告として無視されるようになる。
# それより下の、行数は小さいが影響の大きい書き換えは、件数と内訳が毎回出るのでそこで拾い、
# 必要なら graphify-sync を workflow_dispatch して測り直す。
GRAPHIFY_PENDING_THRESHOLD ?= 3000

# graphify update が見るのは manifest.json の ast_hash だけで、semantic_hash は内容が変わった
# ファイルについて据え置かれる。だから決定的な半分をいくら回しても、意味論側の未処理は
# 消えずに積み上がる（graphify の save_manifest が kind ごとに打刻を分けている）。
#
# 走査ルートを引数で明示する。省略すると graphify は graphify-out/.graphify_root から復元しようと
# するが、あれはマシン固有の絶対パスを持つため追跡しておらず、他の checkout には存在しない。
graphify-update:
	@docker compose run --rm python_tool_runner make graphify-update-ci

graphify-update-ci:
	@graphify update .

graphify-export:
	@docker compose run --rm go_tool_runner make graphify-export-ci

graphify-export-ci:
	@go run ./scripts/graphify-export graphify-out/graph.json graphify-out

# 検査対象は git の index であってツールチェーンではないため、コンテナを起こさずホストで走らせる
# （pin-images-check / egress-check と同じ形）。
graphify-check:
	@go run ./scripts/graphify-check $(if $(SPEC),-spec $(SPEC)) $(if $(BASE),-base $(BASE))

# 意味論抽出そのものはここに無い。あれはアシスタントに `/graphify --update` を実行させるもので、
# make から起動できるコマンドではないため、抽出しないターゲットを graphify-extract と名付けない。
# 正規の経路は GitHub Actions の Graphify Extract（workflow_dispatch）で、手元で回す場合の手順は
# .makefiles/README.md に書いてある。このターゲットが持つのは判断材料だけ。
graphify-pending:
	@go run ./scripts/graphify-pending -threshold $(GRAPHIFY_PENDING_THRESHOLD) $(GRAPHIFY_PENDING_ARGS)
