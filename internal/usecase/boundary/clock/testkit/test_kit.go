// Package testkit は、clock バウンダリのテストに関連するユーティリティを提供します。
package testkit

import (
	"testing"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"

	"go.uber.org/mock/gomock"
)

// NewMockClock は、常に now を返すテスト用のクロックを生成します。
func NewMockClock(t *testing.T, now time.Time) clock.Clock {
	t.Helper()
	ctrl := gomock.NewController(t)
	clk := mock_clock.NewMockClock(ctrl)
	clk.EXPECT().Now().Return(now).AnyTimes()
	return clk
}
