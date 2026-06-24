package httpclient

import (
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// retryableStatusTooManyRequests / retryableStatusServerErrorMin は、retry 対象のステータス境界です。
const (
	retryableStatusTooManyRequests = 429
	retryableStatusServerErrorMin  = 500
	// retryAfterHeader は、サーバが推奨する再試行待機時間を伝える HTTP ヘッダ名です。
	retryAfterHeader = "Retry-After"
)

// isRetrySafe は、req が retry してよいリクエストかを返します。
// 冪等メソッド(GET/PUT/DELETE)は常に安全、非冪等メソッド(POST/PATCH)は AllowRetry 明示時のみ安全です。
func isRetrySafe(req *Request) bool {
	switch req.Method {
	case MethodGet, MethodPut, MethodDelete:
		return true
	case MethodPost, MethodPatch:
		return req.AllowRetry
	default:
		return false
	}
}

// isRetryableOutcome は、試行結果が retry 対象かを返します。
// 5xx / 429 / 応答未取得の transport 失敗は retry 対象、4xx / 成功 / ctx cancel は対象外です。
func isRetryableOutcome(resp *Response, err error) bool {
	if err == nil {
		return false
	}
	if xerrors.Is(err, apperror.ErrCanceled) {
		return false
	}
	if resp != nil {
		return resp.StatusCode == retryableStatusTooManyRequests ||
			resp.StatusCode >= retryableStatusServerErrorMin
	}
	// 応答未取得（network / DNS / TLS / deadline 等）は retry 対象。
	return true
}

// computeBackoff は、attempt（1 起算）に対する指数バックオフ + full jitter の待機時間を返します。
func computeBackoff(attempt int, profile Profile) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	backoff := profile.BaseBackoff << (attempt - 1)
	if backoff <= 0 || backoff > profile.MaxBackoff {
		backoff = profile.MaxBackoff
	}
	return fullJitter(backoff)
}

// retryWait は、次の retry までの待機時間を決定します。
// レスポンスに Retry-After があればそれを優先し、無ければ指数バックオフ + full jitter を用います。
// Retry-After が過大でも overall deadline 規律（canRetryWithin）が別途打ち切るため、ここでは上限を設けません。
func retryWait(attempt int, profile Profile, resp *Response) time.Duration {
	if resp != nil {
		if d, ok := retryAfter(resp.Header, time.Now()); ok {
			return d
		}
	}
	return computeBackoff(attempt, profile)
}

// retryAfter は、Retry-After ヘッダ（delay-seconds 形式 / HTTP-date 形式）を解釈し待機時間を返します。
// 解釈できない / 不在の場合は (0, false) を返し、呼び出し側は通常のバックオフへ fallback します。
func retryAfter(header Header, now time.Time) (time.Duration, bool) {
	if header == nil {
		return 0, false
	}
	values, ok := header[retryAfterHeader]
	if !ok || len(values) == 0 {
		return 0, false
	}

	raw := strings.TrimSpace(values[0])
	if raw == "" {
		return 0, false
	}

	// delay-seconds 形式（非負整数秒）。
	if secs, err := strconv.Atoi(raw); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}

	// HTTP-date 形式。
	if at, err := http.ParseTime(raw); err == nil {
		return max(0, at.Sub(now)), true
	}
	return 0, false
}

// fullJitter は、[0, d] の一様乱数を返します。バックオフの thundering herd を避けます。
func fullJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d) + 1)) //nolint:gosec // jitter は暗号強度不要
}
