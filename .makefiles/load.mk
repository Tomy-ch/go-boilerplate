# ホスト負荷の配分（並行 worktree 前提の CPU シェア）
#
# 設計速度が実装速度を上回ると窓（worktree）を増やして並行実装することになる。このとき各窓が
# ホスト全体を前提にした並列度でゲートを回すと、CPU が飽和して「触っていないテストが落ちる」
# 「lint が 17.5m かかる」「docker がデーモン障害に見える」といった、原因が作業内容と無関係な
# 事故に化ける。ゲートの失敗が作業の失敗と区別できなくなるのが本当の損失で、所要時間ではない。
#
# そこでホストの CPU を窓の数で割った share を全ての重いターゲットの並列度に使う。窓が 1 つなら
# share はホスト全体に等しく、従来と同じ全開で動く（既定の挙動は変わらない）。
#
# 窓の数は git worktree の数で数える。go build も docker も要らず make のパース時に即決まるため、
# ゲート自身の起動を遅らせない。スロットのリースではなく worktree を数えるのは、スロット取得は
# opt-in で「窓はあるがスロットは取っていない」状態が普通にあるためで、負荷を出すのは窓のほう。

# GOBP_LOAD は auto / low / full。auto は窓の数から決める。
GOBP_LOAD ?= auto

# low へ落とす窓数の下限。1 窓なら分割しても意味がなく、2 窓程度ならホストはまだ余裕がある。
GOBP_LOAD_THRESHOLD ?= 3

GOBP_WINDOWS := $(shell git worktree list 2>/dev/null | grep -c . || echo 1)
GOBP_CPUS := $(shell sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)

GOBP_LOAD_RESOLVED := $(strip $(if $(filter auto,$(GOBP_LOAD)), \
	$(shell [ "$(GOBP_WINDOWS)" -ge "$(GOBP_LOAD_THRESHOLD)" ] && echo low || echo full), \
	$(GOBP_LOAD)))

# 1 窓あたりの CPU share。low のときだけ分割し、full ではホスト全体を使う。
# 最低 1 は保証する（0 を渡すとツールによっては「無制限」と解釈され、絞る意図が反転する）。
GOBP_SHARE := $(strip $(if $(filter low,$(GOBP_LOAD_RESOLVED)), \
	$(shell s=$$(( $(GOBP_CPUS) / $(GOBP_WINDOWS) )); [ "$$s" -lt 1 ] && s=1; echo $$s), \
	$(GOBP_CPUS)))

# 低負荷時は他の窓の対話操作（git / docker / エディタ）を待たせないよう優先度も下げる。
# ゲートはバックグラウンド作業で、人間の操作より待てる。
GOBP_NICE := $(if $(filter low,$(GOBP_LOAD_RESOLVED)),nice -n 10,)

# 各ツールへ渡す並列度。full では従来どおりフラグを足さず、ツールの既定（golangci-lint は
# 設定ファイルの concurrency、go test は GOMAXPROCS）に委ねる。
GOLANGCI_CONCURRENCY_FLAG := $(if $(filter low,$(GOBP_LOAD_RESOLVED)),--concurrency $(GOBP_SHARE),)
GO_TEST_P_FLAG := $(if $(filter low,$(GOBP_LOAD_RESOLVED)),-p $(GOBP_SHARE),)

# go test は -p でパッケージの同時実行数を絞れるが、各テストバイナリ内部の並列度
# （t.Parallel() の上限＝GOMAXPROCS）は絞れない。-race はここが特に重いため両方を絞る。
GO_TEST_ENV_LOAD := $(if $(filter low,$(GOBP_LOAD_RESOLVED)),GOMAXPROCS=$(GOBP_SHARE),)

.PHONY: load-status ## 現在の負荷モードと CPU シェアを表示する

load-status:
	@echo "load        : $(GOBP_LOAD_RESOLVED)  (GOBP_LOAD=$(GOBP_LOAD), 閾値 $(GOBP_LOAD_THRESHOLD) 窓)"
	@echo "windows     : $(GOBP_WINDOWS) worktree"
	@echo "cpus        : $(GOBP_CPUS)"
	@echo "cpu share   : $(GOBP_SHARE) / 窓"
	@echo "golangci    : $(if $(GOLANGCI_CONCURRENCY_FLAG),$(GOLANGCI_CONCURRENCY_FLAG),設定ファイルの concurrency に委譲)"
	@echo "go test     : $(if $(GO_TEST_P_FLAG),$(GO_TEST_P_FLAG) $(GO_TEST_ENV_LOAD),既定（ホスト全体）)"
	@echo "nice        : $(if $(GOBP_NICE),$(GOBP_NICE),なし)"
	@echo ""
	@if [ "$(GOBP_LOAD_RESOLVED)" = "low" ]; then \
		echo "💡 窓が多いため低負荷モードです。重いゲート（lint / test）はローカルで粘らず、"; \
		echo "   push して CI で検証するほうが速く確実です:"; \
		echo "     make gate-ci-first   # 手元では軽いゲートだけ通し、重い検証を CI へ委ねる"; \
	else \
		echo "💡 窓が少ないためホスト全体を使います（従来どおり）。"; \
	fi
