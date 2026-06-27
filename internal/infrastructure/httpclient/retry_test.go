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

func TestIsRetrySafe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		cases := map[string]struct {
			req  *Request
			want bool
		}{
			"GETは常に安全":               {req: &Request{Method: MethodGet}, want: true},
			"PUTは常に安全":               {req: &Request{Method: MethodPut}, want: true},
			"DELETEは常に安全":            {req: &Request{Method: MethodDelete}, want: true},
			"POSTはAllowRetryなしで非安全":  {req: &Request{Method: MethodPost}, want: false},
			"POSTはAllowRetryありで安全":   {req: &Request{Method: MethodPost, AllowRetry: true}, want: true},
			"PATCHはAllowRetryなしで非安全": {req: &Request{Method: MethodPatch}, want: false},
			"PATCHはAllowRetryありで安全":  {req: &Request{Method: MethodPatch, AllowRetry: true}, want: true},
			"未知メソッドは非安全":             {req: &Request{Method: Method("TRACE")}, want: false},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tc.want, isRetrySafe(tc.req))
			})
		}
	})
}

func TestIsRetryableOutcome(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		cases := map[string]struct {
			resp *Response
			err  error
			want bool
		}{
			"成功は対象外":       {resp: &Response{StatusCode: 200}, err: nil, want: false},
			"ctxキャンセルは対象外": {resp: nil, err: apperror.ErrCanceled, want: false},
			"500は対象":       {resp: &Response{StatusCode: 500}, err: apperror.ErrUnavailable, want: true},
			"503は対象":       {resp: &Response{StatusCode: 503}, err: apperror.ErrUnavailable, want: true},
			"429は対象":       {resp: &Response{StatusCode: 429}, err: apperror.ErrTooManyRequests, want: true},
			"404は対象外":      {resp: &Response{StatusCode: 404}, err: apperror.ErrNotFound, want: false},
			"400は対象外":      {resp: &Response{StatusCode: 400}, err: apperror.ErrInvalidArgument, want: false},
			"応答未取得のtransport失敗(ErrUnavailable)は対象":  {resp: nil, err: xerrors.Wrap(apperror.ErrUnavailable, "dial error"), want: true},
			"応答未取得でもErrInvalidArgument(不正URL等)は対象外": {resp: nil, err: xerrors.Wrap(apperror.ErrInvalidArgument, "bad url"), want: false},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tc.want, isRetryableOutcome(tc.resp, tc.err))
			})
		}
	})
}

func TestComputeBackoff(t *testing.T) {
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

func TestRetryAfter(t *testing.T) {
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

		cases := map[string]Header{
			"ヘッダ不在":    nil,
			"キー無し":     {"X-Other": {"1"}},
			"空文字":      {retryAfterHeader: {""}},
			"負の秒数":     {retryAfterHeader: {"-1"}},
			"解釈不能な文字列": {retryAfterHeader: {"soon"}},
		}

		for name, header := range cases {
			t.Run(name+"_はfalseを返す", func(t *testing.T) {
				t.Parallel()
				_, ok := retryAfter(header, now)
				assert.False(t, ok)
			})
		}
	})
}
