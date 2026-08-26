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
	codeUnauthorized = "UNAUTHORIZED"
	// codeAccessDenied は、アクセス権限がない操作を試みた場合に使用されるエラーコードです。
	codeAccessDenied = "ACCESS_DENIED"
	// codeNotFound は、指定されたリソースが存在しない場合に使用されるエラーコードです。
	codeNotFound = "NOT_FOUND"
	// codeMethodNotAllowed は、パスに対して許可されていないHTTPメソッドが使われた場合に使用されるエラーコードです。
	codeMethodNotAllowed = "METHOD_NOT_ALLOWED"
	// codeResourceConflict は、既に存在するリソースとの競合が発生した場合に使用されるエラーコードです。
	codeResourceConflict = "RESOURCE_CONFLICT"
	// codeValidationFailed は、入力値の検証に失敗した場合に使用されるエラーコードです。
	codeValidationFailed = "VALIDATION_FAILED"
	// codeUnsupportedMediaType は、サポートされていない Content-Type / メディア形式の場合に使用されるエラーコードです。
	codeUnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"
	// codePayloadTooLarge は、リクエストペイロードが許容サイズを超える場合に使用されるエラーコードです。
	codePayloadTooLarge = "PAYLOAD_TOO_LARGE"
	// codeTooManyRequests は、リクエストが多すぎる場合に使用されるエラーコードです。
	codeTooManyRequests = "TOO_MANY_REQUESTS"
	// codeClientClosedRequest は、クライアントがリクエストをキャンセル/切断した場合に使用されるエラーコードです。
	codeClientClosedRequest = "CLIENT_CLOSED_REQUEST"
	// codeInternalError は、サーバー内部で予期しないエラーが発生した場合に使用されるエラーコードです。
	codeInternalError = "INTERNAL_ERROR"
	// codeNotImplemented は、機能が未実装の場合に使用されるエラーコードです。
	codeNotImplemented = "NOT_IMPLEMENTED"
	// codeServiceUnavailable は、サービスが一時的に利用不可な場合に使用されるエラーコードです。
	codeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// statusClientClosedRequest は、クライアント切断時の非標準ステータス(499, nginx 由来)です。net/http に定数が無いため定義します。
const statusClientClosedRequest = 499

const (
	// errorMessageBadRequest は、リクエストの内容に誤りがあることを示すエラーメッセージです。
	errorMessageBadRequest = "入力内容に誤りがあります。再度ご確認ください。"
	// errorMessageUnauthorized は、認証が必要な操作に対して認証されていない場合に使用されるエラーメッセージです。
	errorMessageUnauthorized = "ログインが必要です。ログインして再度お試しください。"
	// errorMessageAccessDenied は、アクセス権限がない操作を試みた場合に使用されるエラーメッセージです。
	errorMessageAccessDenied = "この操作を行う権限がありません。"
	// errorMessageNotFound は、指定されたリソースが存在しない場合に使用されるエラーメッセージです。
	errorMessageNotFound = "お探しの情報が見つかりませんでした。"
	// errorMessageMethodNotAllowed は、パスに対して許可されていないHTTPメソッドが使われた場合のエラーメッセージです。
	errorMessageMethodNotAllowed = "許可されていないリクエスト方法です。"
	// errorMessageResourceConflict は、既に存在するリソースとの競合が発生した場合に使用されるエラーメッセージです。
	errorMessageResourceConflict = "既に同じ情報が登録されています。"
	// errorMessageValidationFailed は、入力値の検証に失敗した場合のエラーメッセージです。
	errorMessageValidationFailed = "入力内容の検証に失敗しました。修正して再度お試しください。"
	// errorMessageUnsupportedMediaType は、サポートされていないファイル形式・Content-Type の場合のエラーメッセージです。
	errorMessageUnsupportedMediaType = "サポートされていないファイル形式です。形式をご確認のうえ再度お試しください。"
	// errorMessagePayloadTooLarge は、リクエストペイロードが許容サイズを超える場合のエラーメッセージです。
	errorMessagePayloadTooLarge = "ファイルサイズが大きすぎます。上限を超えないファイルで再度お試しください。"
	// errorMessageTooManyRequests は、リクエストが多すぎる場合に使用されるエラーメッセージです。
	errorMessageTooManyRequests = "リクエストが多すぎます。しばらくしてから再度お試しください。"
	// errorMessageClientClosedRequest は、クライアントがリクエストをキャンセル/切断した場合に使用されるエラーメッセージです。
	errorMessageClientClosedRequest = "リクエストがキャンセルされました。"
	// errorMessageInternalError は、サーバー内部で予期しないエラーが発生した場合に使用されるエラーメッセージです。
	errorMessageInternalError = "サーバーで予期しないエラーが発生しました。時間をおいて再度お試しください。"
	// errorMessageNotImplemented は、機能が未実装の場合に使用されるエラーメッセージです。
	errorMessageNotImplemented = "この機能は提供されていません。"
	// errorMessageServiceUnavailable は、サービスが一時的に利用不可な場合に使用されるエラーメッセージです。
	errorMessageServiceUnavailable = "現在この機能はご利用いただけません。しばらくしてから再度お試しください。"
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
	// 405 は apperror sentinel を持たず、ルーティングで解決される(echo.ErrMethodNotAllowed)。
	http.StatusMethodNotAllowed: {
		Status:  http.StatusMethodNotAllowed,
		Code:    codeMethodNotAllowed,
		Message: errorMessageMethodNotAllowed,
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
	http.StatusUnsupportedMediaType: {
		Status:  http.StatusUnsupportedMediaType,
		Code:    codeUnsupportedMediaType,
		Message: errorMessageUnsupportedMediaType,
	},
	http.StatusRequestEntityTooLarge: {
		Status:  http.StatusRequestEntityTooLarge,
		Code:    codePayloadTooLarge,
		Message: errorMessagePayloadTooLarge,
	},
	http.StatusTooManyRequests: {
		Status:  http.StatusTooManyRequests,
		Code:    codeTooManyRequests,
		Message: errorMessageTooManyRequests,
	},
	statusClientClosedRequest: {
		Status:  statusClientClosedRequest,
		Code:    codeClientClosedRequest,
		Message: errorMessageClientClosedRequest,
	},
	http.StatusInternalServerError: {
		Status:  http.StatusInternalServerError,
		Code:    codeInternalError,
		Message: errorMessageInternalError,
	},
	http.StatusNotImplemented: {
		Status:  http.StatusNotImplemented,
		Code:    codeNotImplemented,
		Message: errorMessageNotImplemented,
	},
	http.StatusServiceUnavailable: {
		Status:  http.StatusServiceUnavailable,
		Code:    codeServiceUnavailable,
		Message: errorMessageServiceUnavailable,
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
	return errorMeta[http.StatusInternalServerError]
}

// lookupErrorMetaByAppError は、アプリケーションエラーに対応するエラーメタデータを取得します。
// 既知のエラー型に一致しない場合は 500 Internal Server Error のメタを返します。
func lookupErrorMetaByAppError(err error) httpErrorMeta {
	switch {
	case xerrors.Is(err, apperror.ErrInvalidArgument): // 400
		return lookupErrorMetaByHTTPStatus(http.StatusBadRequest)
	case xerrors.Is(err, apperror.ErrValidation): // 422
		return lookupErrorMetaByHTTPStatus(http.StatusUnprocessableEntity)
	case xerrors.Is(err, apperror.ErrUnsupportedMediaType): // 415
		return lookupErrorMetaByHTTPStatus(http.StatusUnsupportedMediaType)
	case xerrors.Is(err, apperror.ErrPayloadTooLarge): // 413
		return lookupErrorMetaByHTTPStatus(http.StatusRequestEntityTooLarge)
	case xerrors.Is(err, apperror.ErrUnauthenticated): // 401
		return lookupErrorMetaByHTTPStatus(http.StatusUnauthorized)
	case xerrors.Is(err, apperror.ErrPermissionDenied): // 403
		return lookupErrorMetaByHTTPStatus(http.StatusForbidden)
	case xerrors.Is(err, apperror.ErrNotFound): // 404
		return lookupErrorMetaByHTTPStatus(http.StatusNotFound)
	case xerrors.Is(err, apperror.ErrConflict): // 409
		return lookupErrorMetaByHTTPStatus(http.StatusConflict)
	case xerrors.Is(err, apperror.ErrCanceled): // 499
		return lookupErrorMetaByHTTPStatus(statusClientClosedRequest)
	case xerrors.Is(err, apperror.ErrUnavailable): // 503
		return lookupErrorMetaByHTTPStatus(http.StatusServiceUnavailable)
	case xerrors.Is(err, apperror.ErrUnimplemented): // 501
		return lookupErrorMetaByHTTPStatus(http.StatusNotImplemented)
	case xerrors.Is(err, apperror.ErrTooManyRequests): // 429
		return lookupErrorMetaByHTTPStatus(http.StatusTooManyRequests)
	case xerrors.Is(err, apperror.ErrInternal): // 500
		return lookupErrorMetaByHTTPStatus(http.StatusInternalServerError)
	default:
		return lookupErrorMetaByHTTPStatus(http.StatusInternalServerError)
	}
}
