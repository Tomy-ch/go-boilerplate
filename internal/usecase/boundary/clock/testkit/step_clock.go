package testkit

import (
	"context"
	"sync"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock"
)

var (
	_ clock.Clock   = (*StepClock)(nil)
	_ clock.Sleeper = (*StepClock)(nil)
)

// StepClock は、Sleep のたびに固定 step だけ現在時刻を進める、決定的なテスト用の
// clock.Clock 兼 clock.Sleeper です。retry / deadline 規律を実時間や backoff の乱数
// (jitter) に依存させず決定的に検証するために用います。Sleep は要求された待機時間 d では
// なく固定 step だけ時刻を進めるため、jitter で d がばらついても時刻の進みは一定です。
type StepClock struct {
	mu   sync.Mutex
	now  time.Time
	step time.Duration
}

// NewStepClock は、開始時刻 start から Sleep ごとに step だけ進む StepClock を生成します。
func NewStepClock(start time.Time, step time.Duration) *StepClock {
	return &StepClock{now: start, step: step}
}

// Now は、現在の（fake）時刻を返します。
func (c *StepClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Sleep は、実際には待機しません。呼び出し時点で ctx が既にキャンセル / 期限切れの場合は
// 現在時刻を前進させずその error を返し、そうでなければ step だけ進めて nil を返します。
func (c *StepClock) Sleep(ctx context.Context, _ time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	c.now = c.now.Add(c.step)
	c.mu.Unlock()
	return nil
}
