package realtime

import (
	"context"
	"sync/atomic"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// healthProbeStreamID は、EventLog へ到達できるかを確かめる読み取りに使う stream です。
// 存在しない stream の Latest は「無い」を返すだけなので、返るエラーは到達できないことだけを意味します。
const healthProbeStreamID rt.StreamID = "_readiness"

// SubsystemName は、この subsystem の名前です。名乗る場所が違っても同じ値を使います。
const SubsystemName = "realtime"

// Health は、Realtime Delivery が稼働中に配信を続けられる状態かを答えます。EventLog は Check のたびに
// 読んで確かめ、fan-out は consumer が ObserveFanout で報告した直近の結果を使います — Check 自身が
// 受信を試みてはなりません（consumer と取り合いになります）。答えるのは degraded かどうかだけで、
// instance を落とすかどうかではありません（README の Health 行、docs/design/realtime-delivery.md §2.6）。
type Health struct {
	log rt.EventLogStore
	// fanoutDegraded は、consumer が最後に観測した受信の可否です。
	fanoutDegraded atomic.Bool
}

// NewHealth は、EventLog を log で確かめる Health を返します。fan-out は健全な状態から始めます。
func NewHealth(log rt.EventLogStore) *Health {
	return &Health{log: log}
}

// ObserveFanout は、consumer が受信を試みた結果を記録します。err が nil なら健全へ戻します。
func (h *Health) ObserveFanout(err error) {
	h.fanoutDegraded.Store(err != nil)
}

// FanoutDegraded は、fan-out が不調と観測されているかを返します
// （新規接続の受け入れ判断に使う縮退の扱いは docs/design/realtime-delivery.md §2.6）。
func (h *Health) FanoutDegraded() bool {
	return h.fanoutDegraded.Load()
}

// Check は、Realtime Delivery の依存へ到達できるかを確かめます。
// 起動時の fail-fast と稼働中の readiness が同じ定義を使うよう、EventLog の到達性はここだけで判定します。
func (h *Health) Check(ctx context.Context) error {
	if _, _, err := h.log.Latest(ctx, healthProbeStreamID); err != nil {
		return err
	}

	if h.FanoutDegraded() {
		return ErrFanoutUnreachable
	}

	return nil
}
