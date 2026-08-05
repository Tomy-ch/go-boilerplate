# ホスト負荷の配分（並行 worktree 前提の CPU シェアと、ゲートの委譲先）
#
# 設計速度が実装速度を上回ると窓（worktree）を増やして並行実装することになる。このとき各窓が
# ホスト全体を前提にした並列度でゲートを回すと CPU が飽和し、「触っていないテストが落ちる」
# 「lint が 17.5m かかる」「docker がデーモン障害に見える」といった、作業内容と無関係な事故に
# 化ける。失われるのは時間ではなく、ゲートの失敗を作業の失敗と区別できなくなることそのもの。
#
# 対策は 2 段構え。ホストの CPU を窓の数で割った share を重いゲートの並列度に使い（low）、
# それでも足りない窓数では重い検証をローカルで粘らず CI へ委ねる（ci-first）。
#
# 窓の数は git worktree の数で数える。go build も docker も要らず make のパース時に即決まるため、
# ゲート自身の起動を遅らせない。スロットのリースではなく worktree を数えるのは、スロット取得が
# opt-in で「窓はあるがスロットは取っていない」状態が普通にあるためで、CPU を食うのは窓のほう。

# GOBP_LOAD は auto / full / low / ci-first。auto は窓の数から決める。
GOBP_LOAD ?= auto

# 帯域の境目。1〜2 窓ならホストはまだ余裕があり、分割すると単独作業がただ遅くなる。
GOBP_LOW_THRESHOLD ?= 3
GOBP_CI_FIRST_THRESHOLD ?= 5

GOBP_WINDOWS := $(shell git worktree list 2>/dev/null | grep -c . || echo 1)
GOBP_CPUS := $(shell sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)

GOBP_LOAD_RESOLVED := $(strip $(if $(filter auto,$(GOBP_LOAD)), \
	$(shell \
		if [ "$(GOBP_WINDOWS)" -ge "$(GOBP_CI_FIRST_THRESHOLD)" ]; then echo ci-first; \
		elif [ "$(GOBP_WINDOWS)" -ge "$(GOBP_LOW_THRESHOLD)" ]; then echo low; \
		else echo full; fi), \
	$(GOBP_LOAD)))

# 絞る側の帯（low / ci-first）をまとめて判定する。ci-first でもローカルに残す軽いゲートは
# 走るため、絞り自体は両方に効かせる。
GOBP_THROTTLED := $(filter low ci-first,$(GOBP_LOAD_RESOLVED))

# 1 窓あたりの CPU share。最低 1 は保証する（0 を渡すとツールによっては「無制限」と解釈され、
# 絞る意図が反転する）。
GOBP_SHARE := $(strip $(if $(GOBP_THROTTLED), \
	$(shell s=$$(( $(GOBP_CPUS) / $(GOBP_WINDOWS) )); [ "$$s" -lt 1 ] && s=1; echo $$s), \
	$(GOBP_CPUS)))

# 絞る帯では他の窓の対話操作（git / docker / エディタ）を待たせないよう優先度も下げる。
# ゲートはバックグラウンド作業で、人間の操作より待てる。
GOBP_NICE := $(if $(GOBP_THROTTLED),nice -n 10,)

# 各ツールへ渡す並列度。full では従来どおりフラグを足さず、ツールの既定（golangci-lint は
# 設定ファイルの concurrency、go test は GOMAXPROCS）に委ねる。
GOLANGCI_CONCURRENCY_FLAG := $(if $(GOBP_THROTTLED),--concurrency $(GOBP_SHARE),)
GO_TEST_P_FLAG := $(if $(GOBP_THROTTLED),-p $(GOBP_SHARE),)

# go test は -p でパッケージの同時実行数を絞れるが、各テストバイナリ内部の並列度
# （t.Parallel() の上限＝GOMAXPROCS）は絞れない。-race はここが特に重いため両方を絞る。
GO_TEST_LOAD_ENV := $(if $(GOBP_THROTTLED),GOMAXPROCS=$(GOBP_SHARE),)

.PHONY: load-status ## 現在の負荷モードと CPU シェアを表示する
.PHONY: gate-go ## pre-commit の Go ゲート（負荷帯に応じて並列/逐次/CI委譲を切り替える）
.PHONY: gate-go-push ## pre-push の Go ゲート（負荷帯に応じて並列/逐次/CI委譲を切り替える）
.PHONY: gate-fix ## 自動フォーマットの委譲先（毎回走る経路から呼ぶ。ci-first では実行しない）
.PHONY: gate-heavy-skip ## 重いゲートを CI へ委ねる帯かを返す（exit 0 = 委譲する）

# lefthook の skip 条件から呼ぶ述語。終了コードだけが意味を持つ（0 = スキップ）。
# 判定を make 側に置くことで、フックの各 command と make ターゲットが同じ 1 つの
# 負荷帯定義を見る（.lefthook.yaml へ閾値を書き写さない）。
gate-heavy-skip:
	@test "$(GOBP_LOAD_RESOLVED)" = "ci-first"

load-status:
	@echo "load      : $(GOBP_LOAD_RESOLVED)  (GOBP_LOAD=$(GOBP_LOAD))"
	@echo "windows   : $(GOBP_WINDOWS) worktree  (low >= $(GOBP_LOW_THRESHOLD), ci-first >= $(GOBP_CI_FIRST_THRESHOLD))"
	@echo "cpus      : $(GOBP_CPUS)  ->  share $(GOBP_SHARE) / 窓"
	@echo "golangci  : $(if $(GOLANGCI_CONCURRENCY_FLAG),$(GOLANGCI_CONCURRENCY_FLAG),設定ファイルの concurrency に委譲)"
	@echo "go test   : $(if $(GO_TEST_P_FLAG),$(GO_TEST_P_FLAG) $(GO_TEST_LOAD_ENV),既定（ホスト全体）)"
	@echo "nice      : $(if $(GOBP_NICE),$(GOBP_NICE),なし)"
	@echo ""
ifeq ($(GOBP_LOAD_RESOLVED),ci-first)
	@echo "💡 窓が多いため CI-first です。重いゲート（lint / test）はローカルで走らせず、"
	@echo "   push して CI で検証します。手元に残るのは commitlint / secret-scan / pin 検査など、"
	@echo "   push 後では手遅れになる軽いゲートだけです。"
	@echo "   一時的に手元で回すなら: make lint GOBP_LOAD=low"
else ifeq ($(GOBP_LOAD_RESOLVED),low)
	@echo "💡 窓が多いため低負荷モードです。重いゲートは CPU share $(GOBP_SHARE) に絞り、逐次で走ります。"
	@echo "   さらに窓を増やすなら CI へ委ねる方が速く確実です: GOBP_LOAD=ci-first"
else
	@echo "💡 窓が少ないためホスト全体を使います（従来どおり）。"
endif

# pre-commit / pre-push の Go ゲート。lefthook は commands を parallel で走らせるため、
# 重いゲートを個別の command として並べると「窓の数 × フックの同時実行数」で負荷が乗算される。
# Go 系だけを 1 つの command へ束ね、並列度の判断をここへ集約する。
#
# full  : 従来どおり並列（1 窓ならホスト全体を使い切るのが最速）
# low   : 逐次（share で絞ったうえで、同時に走るゲートを 1 つに保つ）
# ci-first: 実行せず CI へ委ねる
define run_go_gates
	@if [ "$(GOBP_LOAD_RESOLVED)" = "ci-first" ]; then \
		echo "⏭  重い Go ゲート($(1))は CI へ委譲します（窓 $(GOBP_WINDOWS) 個 / GOBP_LOAD=$(GOBP_LOAD_RESOLVED)）。"; \
		echo "   手元で回すなら: make $(1) GOBP_LOAD=low"; \
	elif [ -n "$(GOBP_THROTTLED)" ]; then \
		echo "🐌 低負荷モード（share $(GOBP_SHARE)）で $(1) を逐次実行します。"; \
		$(MAKE) $(1); \
	else \
		$(MAKE) -j$(words $(1)) $(1); \
	fi
endef

gate-go:
	$(call run_go_gates,lint test-cached)

gate-go-push:
	$(call run_go_gates,test test-scripts)

# 自動フォーマットの委譲先。`/commit` のように人手を介さず毎回走る経路はこちらを呼ぶ。
# `make fix` を直接叩いたときは帯に関わらず実行する（明示したコマンドは書いたとおりに動く）。
#
# fix は lint と同じ full config を回すため、委譲しないとここだけが帯をすり抜ける。
# 委譲すると CI が赤くなってから直すことになり往復は増えるが、目的は負荷の低減であり、
# フォーマットのずれは CI の lint が確実に捕まえる（見落として壊れる類の指摘ではない）。
gate-fix:
	@if [ "$(GOBP_LOAD_RESOLVED)" = "ci-first" ]; then \
		echo "⏭  自動フォーマット(fix)は委譲します（窓 $(GOBP_WINDOWS) 個 / GOBP_LOAD=$(GOBP_LOAD_RESOLVED)）。"; \
		echo "   フォーマットのずれは CI の lint が指摘します。手元で直すなら: make fix GOBP_LOAD=low"; \
	else \
		$(MAKE) fix; \
	fi
