package httpclient

import (
	"context"
	"errors"
	"net/http"

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

// statusToAppError は、HTTP ステータスコードを apperror sentinel に写像します。
// 2xx / 3xx（成功・リダイレクト）は nil を返します。
// 位置づけは RDB の pgerror.NormalizeError の HTTP 版で、写像は substrate 内部に閉じます。
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
	default:
		return nil
	}
}

// normalizeTransportError は、応答取得前の transport 事象を apperror sentinel に写像します。
// ctx cancel は ErrCanceled、それ以外（network / DNS / TLS / deadline 超過）は ErrUnavailable です。
func normalizeTransportError(err error) error {
	if errors.Is(err, context.Canceled) {
		return xerrors.Wrap(apperror.ErrCanceled, err.Error())
	}
	return xerrors.Wrap(apperror.ErrUnavailable, err.Error())
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
