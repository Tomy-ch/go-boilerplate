package response

import (
	"net/http"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

const (
	// codeBadRequest は、リクエストに不正または不足がある場合に使用されるエラーコードです。
	codeBadRequest = "BAD_REQUEST"
	// codeUnauthorized は、認証されていないアクセスに対して使用されるエラーコードです。
	codeUnauthorized = "UNAUTHORISED"
	// codeAccessDenied は、アクセス権限がない操作を試みた場合に使用されるエラーコードです。
	codeAccessDenied = "ACCESS_DENIED"
	// codeNotFound は、指定されたリソースが存在しない場合に使用されるエラーコードです。
	codeNotFound = "NOT_FOUND"
	// codeResourceConflict は、既に存在するリソースとの競合が発生した場合に使用されるエラーコードです。
	codeResourceConflict = "RESOURCE_CONFLICT"
	// codeValidationFailed は、入力値の検証に失敗した場合に使用されるエラーコードです。
	codeValidationFailed = "VALIDATION_FAILED"
	// codeTooManyRequests は、リクエストが多すぎる場合に使用されるエラーコードです。
	codeTooManyRequests = "TOO_MANY_REQUESTS"
	// codeInternalError は、サーバー内部で予期しないエラーが発生した場合に使用されるエラーコードです。
	codeInternalError = "INTERNAL_ERROR"
	// codeNotAvailable は、機能が未実装または一時的に利用不可な場合に使用されるエラーコードです。
	codeNotAvailable = "NOT_AVAILABLE"
)

const (
	// errorBadRequest は、リクエストの内容に誤りがあることを示すエラーコードです。
	errorMessageBadRequest = "入力内容に誤りがあります。再度ご確認ください。"
	// errorUnauthorized は、認証が必要な操作に対して認証されていない場合に使用されるエラーコードです。
	errorMessageUnauthorized = "ログインが必要です。ログインして再度お試しください。"
	// errorAccessDenied は、アクセス権限がない操作を試みた場合に使用されるエラーコードです。
	errorMessageAccessDenied = "この操作を行う権限がありません。"
	// errorNotFound は、指定されたリソースが存在しない場合に使用されるエラーコードです。
	errorMessageNotFound = "お探しの情報が見つかりませんでした。"
	// errorResourceConflict は、既に存在するリソースとの競合が発生した場合に使用されるエラーコードです。
	errorMessageResourceConflict = "既に同じ情報が登録されています。"
	// errorMessageValidationFailed は、入力値の検証に失敗した場合のメッセージです。
	errorMessageValidationFailed = "入力内容の検証に失敗しました。修正して再度お試しください。"
	// errorMessageTooManyRequests は、リクエストが多すぎる場合に使用されるエラーコードです。
	errorMessageTooManyRequests = "リクエストが多すぎます。しばらくしてから再度お試しください。"
	// errorInternalError は、サーバー内部で予期しないエラーが発生した場合に使用されるエラーコードです。
	errorMessageInternalError = "サーバーで予期しないエラーが発生しました。時間をおいて再度お試しください。"
	// errorNotAvailable は、機能が未実装または一時的に利用不可な場合に使用されるエラーコードです。
	errorMessageNotAvailable = "現在この機能はご利用いただけません。しばらくしてから再度お試しください。"
)

var errorMeta = map[int]httpErrorMeta{
	http.StatusBadRequest: {
		Status:  http.StatusBadRequest,
		Code:    codeBadRequest,
		Message: errorMessageBadRequest,
	},
	http.StatusUnauthorized: {
		Status:  http.StatusUnauthorized,
		Code:    codeUnauthorized,
		Message: errorMessageUnauthorized,
	},
	http.StatusForbidden: {
		Status:  http.StatusForbidden,
		Code:    codeAccessDenied,
		Message: errorMessageAccessDenied,
	},
	http.StatusNotFound: {
		Status:  http.StatusNotFound,
		Code:    codeNotFound,
		Message: errorMessageNotFound,
	},
	http.StatusConflict: {
		Status:  http.StatusConflict,
		Code:    codeResourceConflict,
		Message: errorMessageResourceConflict,
	},
	http.StatusUnprocessableEntity: {
		Status:  http.StatusUnprocessableEntity,
		Code:    codeValidationFailed,
		Message: errorMessageValidationFailed,
	},
	http.StatusTooManyRequests: {
		Status:  http.StatusTooManyRequests,
		Code:    codeTooManyRequests,
		Message: errorMessageTooManyRequests,
	},
	http.StatusInternalServerError: {
		Status:  http.StatusInternalServerError,
		Code:    codeInternalError,
		Message: errorMessageInternalError,
	},
	http.StatusNotImplemented: {
		Status:  http.StatusNotImplemented,
		Code:    codeNotAvailable,
		Message: errorMessageNotAvailable,
	},
	http.StatusServiceUnavailable: {
		Status:  http.StatusServiceUnavailable,
		Code:    codeNotAvailable,
		Message: errorMessageNotAvailable,
	},
}

type httpErrorMeta struct {
	Status  int
	Code    string
	Message string
}

// lookupErrorMetaByHTTPStatus は、HTTPステータスコードに対応するエラーメタデータを取得します。
// 存在しない場合は、サーバーエラーのメタデータを返します。
func lookupErrorMetaByHTTPStatus(status int) httpErrorMeta {
	if meta, ok := errorMeta[status]; ok {
		return meta
	}
	return httpErrorMeta{
		Status:  http.StatusInternalServerError,
		Code:    codeInternalError,
		Message: errorMessageInternalError,
	}
}

// lookupErrorMetaByAppError は、アプリケーションエラーに対応するエラーメタデータを取得します。
func lookupErrorMetaByAppError(err error) httpErrorMeta {
	switch {
	case xerrors.Is(err, apperror.ErrInvalidArgument): // 400
		return lookupErrorMetaByHTTPStatus(http.StatusBadRequest)
	case xerrors.Is(err, apperror.ErrValidation): // 422
		return lookupErrorMetaByHTTPStatus(http.StatusUnprocessableEntity)
	case xerrors.Is(err, apperror.ErrUnauthenticated): // 401
		return lookupErrorMetaByHTTPStatus(http.StatusUnauthorized)
	case xerrors.Is(err, apperror.ErrPermissionDenied): // 403
		return lookupErrorMetaByHTTPStatus(http.StatusForbidden)
	case xerrors.Is(err, apperror.ErrNotFound): // 404
		return lookupErrorMetaByHTTPStatus(http.StatusNotFound)
	case xerrors.Is(err, apperror.ErrConflict): // 409
		return lookupErrorMetaByHTTPStatus(http.StatusConflict)
	case xerrors.Is(err, apperror.ErrUnavailable): // 503
		return lookupErrorMetaByHTTPStatus(http.StatusServiceUnavailable)
	case xerrors.Is(err, apperror.ErrUnimplemented): // 501
		return lookupErrorMetaByHTTPStatus(http.StatusNotImplemented)
	case xerrors.Is(err, apperror.ErrTooManyRequests): // 429
		return lookupErrorMetaByHTTPStatus(http.StatusTooManyRequests)
	default:
		return lookupErrorMetaByHTTPStatus(http.StatusInternalServerError)
	}
}
