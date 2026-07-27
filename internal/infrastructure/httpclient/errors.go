package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

const (
	// HTTP ステータスクラスの下限値。
	statusSuccessMin     = 200
	statusRedirectMin    = 300
	statusClientErrorMin = 400
	statusServerErrorMin = 500
)

// 静的な apperror ラップ（sentinel）を集約します。動的ラップ・分類(Is)は呼び出し側で行います。
var (
	// errCircuitOpen は、circuit breaker による fail-fast を表す内部マーカです。
	// ErrUnavailable を内包するため呼び出し側の分類は従来どおりですが、metrics では transport 失敗と
	// 区別して計上するために使います。
	errCircuitOpen = xerrors.Wrap(apperror.ErrUnavailable, "circuit open")

	// 型で防げない Request の precondition 違反（いずれも ErrInvalidArgument）。
	errDownstreamRequired     = xerrors.Wrap(apperror.ErrInvalidArgument, "Downstream is required")
	errMethodRequired         = xerrors.Wrap(apperror.ErrInvalidArgument, "Method is required")
	errIdempotencyKeyRequired = xerrors.Wrap(apperror.ErrInvalidArgument, "AllowRetry requires IdempotencyKey")

	// errResponseTooLarge は、レスポンスボディが上限を超過したことを表します。
	// 再試行しても同じ結果になる決定的失敗なので、transport 失敗(ErrUnavailable)ではなく
	// 非 retry 系(buildRequest の precondition 違反と同じ ErrInvalidArgument)を内包させます。
	// これにより isRetryableOutcome が false となり、無駄な再試行と breaker の自傷 open を防ぎます。
	errResponseTooLarge = xerrors.Wrap(apperror.ErrInvalidArgument, "response body exceeds max bytes")
)

// statusToAppError は、HTTP ステータスコードを apperror sentinel に写像します。
// 2xx は nil を返します。3xx はリダイレクト非追従のため未解決とみなし ErrUnavailable を返します
// （resp に Location を残すので、追従が必要な呼び出し側はそれを参照できます）。
// 写像は substrate 内部に閉じます。
func statusToAppError(statusCode int) error {
	switch statusCode {
	case http.StatusBadRequest:
		return apperror.ErrInvalidArgument
	case http.StatusUnauthorized:
		return apperror.ErrUnauthenticated
	case http.StatusForbidden:
		return apperror.ErrPermissionDenied
	case http.StatusNotFound:
		return apperror.ErrNotFound
	case http.StatusConflict:
		return apperror.ErrConflict
	case http.StatusUnprocessableEntity:
		return apperror.ErrValidation
	case http.StatusTooManyRequests:
		return apperror.ErrTooManyRequests
	}

	switch {
	case statusCode >= statusServerErrorMin:
		return apperror.ErrUnavailable
	case statusCode >= statusClientErrorMin:
		return apperror.ErrInvalidArgument
	case statusCode >= statusRedirectMin:
		return apperror.ErrUnavailable // 3xx: リダイレクト非追従のため未解決
	default:
		return nil // 2xx
	}
}

// normalizeTransportError は、応答取得前の transport 事象を apperror sentinel に写像します。
// ctx cancel は ErrCanceled、それ以外（network / DNS / TLS / deadline 超過）は ErrUnavailable です。
func normalizeTransportError(err error) error {
	if xerrors.Is(err, context.Canceled) {
		return xerrors.Wrap(apperror.ErrCanceled, redactErrMessage(err))
	}
	return xerrors.Wrap(apperror.ErrUnavailable, redactErrMessage(err))
}

// redactErrMessage は、エラーメッセージから URL のクエリ等の機密になり得る情報を除去します。
// *url.Error は完全 URL（クエリ込み）を出力するため、クエリ・userinfo・fragment を除去し scheme/host/path を残します。
func redactErrMessage(err error) string {
	var urlErr *url.Error
	if xerrors.As(err, &urlErr) {
		return fmt.Sprintf("%s %s: %v", urlErr.Op, redactURL(urlErr.URL), urlErr.Err)
	}
	return err.Error()
}

// redactURL は、URL から機密になり得るクエリ・userinfo・fragment のみを除去し、
// 診断に有用な scheme/host/path は残します。パース不能時のみ "upstream" を返します。
func redactURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "upstream"
	}
	parsed.RawQuery = ""
	parsed.User = nil
	parsed.Fragment = ""
	return parsed.String()
}

// statusClass は、ステータスコードを metrics ラベル用のクラス文字列に変換します。
func statusClass(statusCode int) string {
	switch {
	case statusCode >= statusServerErrorMin:
		return "5xx"
	case statusCode >= statusClientErrorMin:
		return "4xx"
	case statusCode >= statusRedirectMin:
		return "3xx"
	case statusCode >= statusSuccessMin:
		return "2xx"
	default:
		return "1xx"
	}
}
