// Package testkit は、clock バウンダリのテストに関連するユーティリティを提供します。
package testkit

import (
	"testing"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"

	"go.uber.org/mock/gomock"
)

// NewMockClock は、常に now を返すテスト用のクロックを生成します（呼び出し回数は問いません）。
func NewMockClock(t *testing.T, now time.Time) clock.Clock {
	t.Helper()
	ctrl := gomock.NewController(t)
	clk := mock_clock.NewMockClock(ctrl)
	clk.EXPECT().Now().Return(now).AnyTimes()
	return clk
}

// NewMockClockOnce は、now を返すテスト用のクロックを生成します（Now が 1 回だけ呼ばれることを検証）。
func NewMockClockOnce(t *testing.T, now time.Time) clock.Clock {
	t.Helper()
	ctrl := gomock.NewController(t)
	clk := mock_clock.NewMockClock(ctrl)
	clk.EXPECT().Now().Return(now).Times(1)
	return clk
}
