// Package system は、システム全体で使用される共通のインフラストラクチャを提供します。
package system

import (
	"context"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock"
)

type systemClock struct{}

// NewClock は、clockを生成します。
func NewClock() clock.Clock {
	return &systemClock{}
}

// NewSleeper は、clock.Sleeper の実装を生成します。
func NewSleeper() clock.Sleeper {
	return &systemClock{}
}

// Now は、現在の時刻を返します。
func (sc *systemClock) Now() time.Time {
	return time.Now()
}

// Sleep は、d 経過まで待機します。ctx が先に done になった場合は ctx.Err() を返します。
func (sc *systemClock) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
