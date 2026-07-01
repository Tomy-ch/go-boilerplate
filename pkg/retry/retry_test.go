package retry

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSleeper は、Sleeper のテスト用 fake です。
// 呼び出し回数を記録し、待機は即時化します。errAt 指定時はその回数目で error を返します。
type fakeSleeper struct {
	calls int
	errAt int   // 0 なら常に nil。N なら N 回目の Sleep で sleepErr を返す。
	err   error // errAt 到達時に返す error。
}

func (s *fakeSleeper) Sleep(_ context.Context, _ time.Duration) error {
	s.calls++
	if s.errAt > 0 && s.calls == s.errAt {
		return s.err
	}
	return nil
}

// alwaysRetryable / neverRetryable は、テスト用の分類関数です。
func alwaysRetryable(error) bool { return true }
func neverRetryable(error) bool  { return false }

func TestDo(t *testing.T) {
	t.Parallel()

	errRetryable := errors.New("retryable")
	errFatal := errors.New("fatal")
	errSleep := errors.New("ctx canceled")

	policy := Policy{
		MaxAttempts: 3,
		Backoff:     func(attempt int) time.Duration { return time.Duration(attempt+1) * time.Millisecond },
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("初回成功時は1回だけ試行しnilを返す", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{}
			calls := 0
			err := Do(context.Background(), sleeper, policy, alwaysRetryable, func(context.Context) error {
				calls++
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, 1, calls)
			assert.Equal(t, 0, sleeper.calls)
		})

		t.Run("N-1回リトライ可能失敗後に成功するとnilを返す", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{}
			calls := 0
			err := Do(context.Background(), sleeper, policy, alwaysRetryable, func(context.Context) error {
				calls++
				if calls < policy.MaxAttempts {
					return errRetryable
				}
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, policy.MaxAttempts, calls)
			assert.Equal(t, policy.MaxAttempts-1, sleeper.calls)
		})

		t.Run("リトライ不可エラーは即座に返し待機しない", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{}
			calls := 0
			err := Do(context.Background(), sleeper, policy, neverRetryable, func(context.Context) error {
				calls++
				return errFatal
			})

			require.ErrorIs(t, err, errFatal)
			assert.Equal(t, 1, calls)
			assert.Equal(t, 0, sleeper.calls)
		})

		t.Run("Backoffがnilなら待機0で再試行する", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{}
			calls := 0
			err := Do(context.Background(), sleeper, Policy{MaxAttempts: 2}, alwaysRetryable, func(context.Context) error {
				calls++
				if calls < 2 {
					return errRetryable
				}
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, 2, calls)
			assert.Equal(t, 1, sleeper.calls)
		})

		t.Run("isRetryableがnilでもfn成功時はパニックせずnilを返す", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{}
			calls := 0
			// fn が nil を返すと err==nil の短絡で isRetryable は呼ばれないため、nil でもパニックしない。
			err := Do(context.Background(), sleeper, policy, nil, func(context.Context) error {
				calls++
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, 1, calls)
			assert.Equal(t, 0, sleeper.calls)
		})

		t.Run("MaxAttemptsが1未満でも最低1回試行する", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{}
			calls := 0
			err := Do(context.Background(), sleeper, Policy{MaxAttempts: 0}, alwaysRetryable, func(context.Context) error {
				calls++
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, 1, calls)
			assert.Equal(t, 0, sleeper.calls)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全試行がリトライ可能失敗なら最後のエラーを返す", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{}
			calls := 0
			err := Do(context.Background(), sleeper, policy, alwaysRetryable, func(context.Context) error {
				calls++
				return errRetryable
			})

			require.ErrorIs(t, err, errRetryable)
			assert.Equal(t, policy.MaxAttempts, calls)
			assert.Equal(t, policy.MaxAttempts-1, sleeper.calls)
		})

		t.Run("isRetryableがnilかつfnがエラーを返す場合パニックせず最初のエラーを即返す", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{}
			calls := 0
			// isRetryable が nil でも「リトライ対象なし」へ正規化されるため、
			// fn の非 nil error はリトライされず最初の error で即座に返る。
			err := Do(context.Background(), sleeper, policy, nil, func(context.Context) error {
				calls++
				return errFatal
			})

			require.ErrorIs(t, err, errFatal)
			assert.Equal(t, 1, calls)
			assert.Equal(t, 0, sleeper.calls)
		})

		t.Run("Sleep打ち切り時はsleepエラーではなく直前のfnエラーを返す", func(t *testing.T) {
			t.Parallel()

			sleeper := &fakeSleeper{errAt: 1, err: errSleep}
			calls := 0
			err := Do(context.Background(), sleeper, policy, alwaysRetryable, func(context.Context) error {
				calls++
				return errRetryable
			})

			require.ErrorIs(t, err, errRetryable)
			require.NotErrorIs(t, err, errSleep)
			assert.Equal(t, 1, calls)
			assert.Equal(t, 1, sleeper.calls)
		})
	})
}

func TestFull(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("0以下は決定論的に0を返す", func(t *testing.T) {
			t.Parallel()

			t.Run("0は0", func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, time.Duration(0), Full(0))
			})

			t.Run("負数は0", func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, time.Duration(0), Full(-time.Second))
			})

			t.Run("最小負数は0", func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, time.Duration(0), Full(time.Duration(math.MinInt64)))
			})
		})

		t.Run("正の値は0からその値の範囲に収まる", func(t *testing.T) {
			t.Parallel()

			for range 50 {
				got := Full(100 * time.Millisecond)
				assert.GreaterOrEqual(t, got, time.Duration(0))
				assert.LessOrEqual(t, got, 100*time.Millisecond)
			}
		})

		t.Run("d=1でも多数回試行で常に0からdの範囲に収まる", func(t *testing.T) {
			t.Parallel()

			for range 100 {
				got := Full(time.Duration(1))
				assert.GreaterOrEqual(t, got, time.Duration(0))
				assert.LessOrEqual(t, got, time.Duration(1))
			}
		})

		t.Run("dがMaxInt64でもオーバーフローでパニックせず範囲に収まる", func(t *testing.T) {
			t.Parallel()

			// [0, d] 閉区間化の +1 が int64 をオーバーフローする境界。
			// 本体のガードにより panic せず [0, d) に収まる。
			assert.NotPanics(t, func() {
				for range 100 {
					got := Full(time.Duration(math.MaxInt64))
					assert.GreaterOrEqual(t, got, time.Duration(0))
					assert.Less(t, got, time.Duration(math.MaxInt64))
				}
			})
		})
	})
}
