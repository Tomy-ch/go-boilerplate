package exchangerate_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/webapi/exchangerate"
	"go-boilerplate/internal/usecase/boundary/clock/testkit"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
	mock_exchangerate "go-boilerplate/internal/usecase/boundary/exchangerate/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// clockStart は、テスト用 StepClock の基準時刻です。
var clockStart = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func Test_cacheGateway_GetRate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TTL内の2回目要求はinner_gatewayを再呼び出ししない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			inner := mock_exchangerate.NewMockGateway(ctrl)
			// 2 回要求しても inner（＝実 HTTP gateway 相当）の呼び出しは 1 回だけであることを検証する。
			inner.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: 150.5, Date: "2026-07-21"}, nil).
				Times(1)

			// step を進めない（Sleep を呼ばない）ので時刻は TTL 内に留まる。
			cached := exchangerate.NewCache(inner, testkit.NewStepClock(clockStart, 0))

			first, err := cached.GetRate(context.Background(), "USD", "JPY")
			require.NoError(t, err)

			second, err := cached.GetRate(context.Background(), "USD", "JPY")
			require.NoError(t, err)

			assert.Equal(t, first, second)
		})

		t.Run("TTL失効後は再度inner_gatewayを呼ぶ", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			inner := mock_exchangerate.NewMockGateway(ctrl)
			inner.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: 150.5}, nil).
				Times(2)

			// Sleep 1 回で 25h 進み、24h TTL を跨いで失効する。
			clk := testkit.NewStepClock(clockStart, 25*time.Hour)
			cached := exchangerate.NewCache(inner, clk)

			_, err := cached.GetRate(context.Background(), "USD", "JPY")
			require.NoError(t, err)

			require.NoError(t, clk.Sleep(context.Background(), 0)) // 時刻を TTL 超へ前進

			_, err = cached.GetRate(context.Background(), "USD", "JPY")
			require.NoError(t, err)
		})

		t.Run("通貨ペアが異なればそれぞれinnerを呼ぶ", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			inner := mock_exchangerate.NewMockGateway(ctrl)
			inner.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: 150.5}, nil).
				Times(1)
			inner.EXPECT().
				GetRate(gomock.Any(), "USD", "EUR").
				Return(&boundary.Rate{Base: "USD", Quote: "EUR", Value: 0.92}, nil).
				Times(1)

			cached := exchangerate.NewCache(inner, testkit.NewStepClock(clockStart, 0))

			jpy, err := cached.GetRate(context.Background(), "USD", "JPY")
			require.NoError(t, err)
			assert.Equal(t, "JPY", jpy.Quote)

			eur, err := cached.GetRate(context.Background(), "USD", "EUR")
			require.NoError(t, err)
			assert.Equal(t, "EUR", eur.Quote)
		})

		t.Run("同一キーへの並行要求でもデータ競合しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			inner := mock_exchangerate.NewMockGateway(ctrl)
			// TOCTOU で inner が複数回呼ばれうるため回数は固定しない。
			inner.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: 150.5}, nil).
				AnyTimes()

			cached := exchangerate.NewCache(inner, testkit.NewStepClock(clockStart, 0))

			const n = 32
			rates := make([]*boundary.Rate, n)
			errs := make([]error, n)

			// 各 goroutine は異なる添字へ書き込むため競合しない。検証はメイン goroutine で行う。
			var wg sync.WaitGroup
			for i := range n {
				wg.Go(func() {
					rates[i], errs[i] = cached.GetRate(context.Background(), "USD", "JPY")
				})
			}
			wg.Wait()

			for i := range n {
				require.NoError(t, errs[i])
				require.NotNil(t, rates[i])
				assert.InEpsilon(t, 150.5, rates[i].Value, 1e-9)
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("innerのエラーはキャッシュせず都度伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			inner := mock_exchangerate.NewMockGateway(ctrl)
			// エラーはキャッシュしないため、2 回目要求でも inner が再度呼ばれる。
			inner.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "downstream down")).
				Times(2)

			cached := exchangerate.NewCache(inner, testkit.NewStepClock(clockStart, 0))

			_, err := cached.GetRate(context.Background(), "USD", "JPY")
			require.ErrorIs(t, err, apperror.ErrUnavailable)

			_, err = cached.GetRate(context.Background(), "USD", "JPY")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}
