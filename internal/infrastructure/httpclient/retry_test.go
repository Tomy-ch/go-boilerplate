package httpclient

import (
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isRetrySafe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GETは常に安全", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetrySafe(&Request{method: MethodGet()}))
		})

		t.Run("PUTは常に安全", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetrySafe(&Request{method: MethodPut()}))
		})

		t.Run("DELETEは常に安全", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetrySafe(&Request{method: MethodDelete()}))
		})

		t.Run("POSTはAllowRetryなしで非安全", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetrySafe(&Request{method: MethodPost()}))
		})

		t.Run("POSTはAllowRetryありで安全", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetrySafe(&Request{method: MethodPost(), allowRetry: true}))
		})

		t.Run("PATCHはAllowRetryなしで非安全", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetrySafe(&Request{method: MethodPatch()}))
		})

		t.Run("PATCHはAllowRetryありで安全", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetrySafe(&Request{method: MethodPatch(), allowRetry: true}))
		})

		t.Run("ゼロ値(未設定)メソッドは非安全", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetrySafe(&Request{method: Method{}}))
		})
	})
}

func Test_isRetryableOutcome(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("成功は対象外", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetryableOutcome(&Response{StatusCode: 200}, nil))
		})

		t.Run("ctxキャンセルは対象外", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetryableOutcome(nil, apperror.ErrCanceled))
		})

		t.Run("500は対象", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetryableOutcome(&Response{StatusCode: 500}, apperror.ErrUnavailable))
		})

		t.Run("503は対象", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetryableOutcome(&Response{StatusCode: 503}, apperror.ErrUnavailable))
		})

		t.Run("429は対象", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetryableOutcome(&Response{StatusCode: 429}, apperror.ErrTooManyRequests))
		})

		t.Run("404は対象外", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetryableOutcome(&Response{StatusCode: 404}, apperror.ErrNotFound))
		})

		t.Run("400は対象外", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetryableOutcome(&Response{StatusCode: 400}, apperror.ErrInvalidArgument))
		})

		t.Run("応答未取得のtransport失敗(ErrUnavailable)は対象", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetryableOutcome(nil, xerrors.Wrap(apperror.ErrUnavailable, "dial error")))
		})

		t.Run("応答未取得でもErrInvalidArgument(不正URL等)は対象外", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetryableOutcome(nil, xerrors.Wrap(apperror.ErrInvalidArgument, "bad url")))
		})

		t.Run("応答未取得のcircuit_open(ErrUnavailable内包)は対象", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isRetryableOutcome(nil, errCircuitOpen))
		})

		t.Run("レスポンス上限超過(errResponseTooLarge=ErrUnavailable内包)は決定的失敗なので対象外", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isRetryableOutcome(nil, errResponseTooLarge))
		})
	})
}

func Test_computeBackoff(t *testing.T) {
	t.Parallel()

	profile := Profile{
		BaseBackoff: 10 * time.Millisecond,
		MaxBackoff:  1 * time.Second,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("各attemptのバックオフが指数cap内のfull_jitter範囲に収まる", func(t *testing.T) {
			t.Parallel()

			for attempt := 1; attempt <= 8; attempt++ {
				upperBound := profile.BaseBackoff << (attempt - 1)
				if upperBound <= 0 || upperBound > profile.MaxBackoff {
					upperBound = profile.MaxBackoff
				}

				// jitter は乱数のため複数回試行して常に [0, upperBound] に収まることを確認する。
				for range 50 {
					got := computeBackoff(attempt, profile)
					assert.GreaterOrEqual(t, got, time.Duration(0))
					assert.LessOrEqual(t, got, upperBound)
				}
			}
		})

		t.Run("attemptが0以下でも1として扱う", func(t *testing.T) {
			t.Parallel()

			got := computeBackoff(0, profile)
			assert.GreaterOrEqual(t, got, time.Duration(0))
			assert.LessOrEqual(t, got, profile.BaseBackoff)
		})
	})
}

func Test_retryWait(t *testing.T) {
	t.Parallel()

	profile := Profile{
		BaseBackoff: 10 * time.Millisecond,
		MaxBackoff:  time.Second,
	}

	now := time.Unix(1000, 0).UTC()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("respがnilならバックオフ範囲の待機時間を返す", func(t *testing.T) {
			t.Parallel()

			// jitter があるため複数回試行し attempt1 の cap 範囲に収まることを確認する。
			for range 50 {
				got := retryWait(1, profile, nil, now)
				assert.GreaterOrEqual(t, got, time.Duration(0))
				assert.LessOrEqual(t, got, profile.BaseBackoff)
			}
		})

		t.Run("respはあるがRetry-After不在ならバックオフ範囲の待機時間を返す", func(t *testing.T) {
			t.Parallel()

			resp := &Response{StatusCode: 503, Header: Header{}}
			for range 50 {
				got := retryWait(1, profile, resp, now)
				assert.GreaterOrEqual(t, got, time.Duration(0))
				assert.LessOrEqual(t, got, profile.BaseBackoff)
			}
		})

		t.Run("Retry-Afterがあればバックオフより優先する", func(t *testing.T) {
			t.Parallel()

			resp := &Response{StatusCode: 503, Header: Header{retryAfterHeader: {"5"}}}
			got := retryWait(1, profile, resp, now)
			assert.Equal(t, 5*time.Second, got) // バックオフ(<=10ms)ではなく Retry-After(5s)
		})

		t.Run("HTTP-date形式のRetry-Afterは固定時刻との差分を待機時間とする", func(t *testing.T) {
			t.Parallel()

			future := now.Add(7 * time.Second).Format(http.TimeFormat)
			resp := &Response{StatusCode: 503, Header: Header{retryAfterHeader: {future}}}
			got := retryWait(1, profile, resp, now)
			assert.Equal(t, 7*time.Second, got) // 渡した now を基準に決定論的に算出する
		})
	})
}

func Test_retryAfter(t *testing.T) {
	t.Parallel()

	now := time.Unix(1000, 0).UTC()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("delay-seconds形式を秒数として解釈する", func(t *testing.T) {
			t.Parallel()

			d, ok := retryAfter(Header{retryAfterHeader: {"3"}}, now)
			require.True(t, ok)
			assert.Equal(t, 3*time.Second, d)
		})

		t.Run("HTTP-date形式は現在時刻との差分として解釈する", func(t *testing.T) {
			t.Parallel()

			future := now.Add(5 * time.Second).Format(http.TimeFormat)
			d, ok := retryAfter(Header{retryAfterHeader: {future}}, now)
			require.True(t, ok)
			assert.Equal(t, 5*time.Second, d)
		})

		t.Run("過去のHTTP-dateは0として扱う", func(t *testing.T) {
			t.Parallel()

			past := now.Add(-5 * time.Second).Format(http.TimeFormat)
			d, ok := retryAfter(Header{retryAfterHeader: {past}}, now)
			require.True(t, ok)
			assert.Equal(t, time.Duration(0), d)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘッダ不在_はfalseを返す", func(t *testing.T) {
			t.Parallel()
			_, ok := retryAfter(nil, now)
			assert.False(t, ok)
		})

		t.Run("キー無し_はfalseを返す", func(t *testing.T) {
			t.Parallel()
			_, ok := retryAfter(Header{"X-Other": {"1"}}, now)
			assert.False(t, ok)
		})

		t.Run("空文字_はfalseを返す", func(t *testing.T) {
			t.Parallel()
			_, ok := retryAfter(Header{retryAfterHeader: {""}}, now)
			assert.False(t, ok)
		})

		t.Run("負の秒数_はfalseを返す", func(t *testing.T) {
			t.Parallel()
			_, ok := retryAfter(Header{retryAfterHeader: {"-1"}}, now)
			assert.False(t, ok)
		})

		t.Run("解釈不能な文字列_はfalseを返す", func(t *testing.T) {
			t.Parallel()
			_, ok := retryAfter(Header{retryAfterHeader: {"soon"}}, now)
			assert.False(t, ok)
		})
	})
}
